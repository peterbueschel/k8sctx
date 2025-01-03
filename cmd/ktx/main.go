package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"html/template"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/peterbueschel/k8sctx"
)

var (
	version = "local"

	appStyle = lipgloss.NewStyle().Padding(1, 2)

	dimmedTitle = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#C2B8C2"}).
			Padding(0, 0, 0, 2) //nolint:mnd

	dimmedDesc = dimmedTitle.
			Foreground(lipgloss.AdaptiveColor{Light: "#C2B8C2", Dark: "#A49FA5"})

	statusBarFilterCount = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#00ACAC"})

	statusBar = lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#DDDADA", Dark: "#FFACAC"}).
			Padding(0, 0, 1, 2) //nolint:mnd

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFDF5")).
			Background(lipgloss.Color("#00AAA0")).
			Padding(0, 1)

	errorMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#ff0000", Dark: "#ff0000"}).
				Render

	noContextFound = "No previous context found in state file. You need to switch the kube context at least twice."

	//go:embed jsonnet/.libsonnet
	contextsLibsonnet string
	//go:embed jsonnet/config.jsonnet
	configJsonnet string
)

type listKeyMap struct {
	toggleHelpMenu key.Binding
}

type model struct {
	list             list.Model
	contexts         *k8sctx.Config
	keys             *listKeyMap
	delegateKeys     *delegateKeyMap
	quitting         bool
	useInitialFilter bool
}

type item struct {
	title       string
	description string
}

func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }
func (i item) FilterValue() string { return i.title }

type delegateKeyMap struct {
	choose key.Binding
}

func newItemDelegate(keys *delegateKeyMap, c *k8sctx.Config) (list.DefaultDelegate, tea.Cmd) {
	d := list.NewDefaultDelegate()

	d.UpdateFunc = func(msg tea.Msg, m *list.Model) tea.Cmd {
		var title string

		if i, ok := m.SelectedItem().(item); ok {
			title = i.Title()
		} else {
			return nil
		}
		m.StatusMessageLifetime = 10 * time.Second

		switch msg := msg.(type) {
		case tea.KeyMsg:
			if key.Matches(msg, keys.choose) {
				kcnf, ctx, idx := c.GetContextBy(title)
				if idx == -1 {
					return m.NewStatusMessage(
						errorMessageStyle(fmt.Sprintf("Context '%s' not found in kube config files", title)),
					)
				}
				if err := c.RemoveCurrentContexts(); err != nil {
					return m.NewStatusMessage(
						errorMessageStyle(
							fmt.Sprintf("Failed to remove old current-contexts: '%s'", err.Error())),
					)
				}
				if err := kcnf.KubeConfig.SetContextTo(ctx["name"]); err != nil {
					return m.NewStatusMessage(
						errorMessageStyle(
							fmt.Sprintf("Failed to set current-context to '%s': '%s'", title, err.Error())),
					)
				}
				if err := c.UpdateState(kcnf, ctx["name"]); err != nil {
					return m.NewStatusMessage(
						errorMessageStyle(
							fmt.Sprintf("Failed to update state file'%s': '%s'", c.Filename, err.Error())),
					)
				}
				return tea.Quit
			}
		}
		return nil
	}

	help := []key.Binding{keys.choose}

	d.ShortHelpFunc = func() []key.Binding {
		return help
	}

	d.FullHelpFunc = func() [][]key.Binding {
		return [][]key.Binding{help}
	}

	return d, nil
}

// Additional short help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		d.choose,
	}
}

// Additional full help entries. This satisfies the help.KeyMap interface and
// is entirely optional.
func (d delegateKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{
			d.choose,
		},
	}
}

func newListKeyMap() *listKeyMap {
	return &listKeyMap{
		toggleHelpMenu: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "toggle help"),
		),
	}
}

func newDelegateKeyMap() *delegateKeyMap {
	return &delegateKeyMap{
		choose: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "choose"),
		),
	}
}

func modelFrom(c *k8sctx.Config, configFilter, contextFilter string) model {
	var (
		delegateKeys = newDelegateKeyMap()
		listKeys     = newListKeyMap()
	)
	contexts := c.CreateListItems(configFilter, contextFilter)
	items := make([]list.Item, len(contexts))
	for idx, ctx := range contexts {
		i := item{
			title:       ctx.Name,
			description: ctx.Description,
		}
		items[idx] = i
	}

	// Setup list
	delegate, _ := newItemDelegate(delegateKeys, c)
	delegate.Styles.DimmedTitle = dimmedTitle
	delegate.Styles.DimmedDesc = dimmedDesc
	contextList := list.New(items, delegate, 0, 0)
	contextList.Styles.StatusBarFilterCount = statusBarFilterCount
	contextList.Styles.StatusBar = statusBar
	contextList.Title = "Kube Contexts"
	contextList.Styles.Title = titleStyle
	contextList.FilterInput.Prompt = `Filter: `
	contextList.AdditionalFullHelpKeys = func() []key.Binding {
		return []key.Binding{
			listKeys.toggleHelpMenu,
		}
	}

	return model{
		list:             contextList,
		keys:             listKeys,
		delegateKeys:     delegateKeys,
		contexts:         c,
		useInitialFilter: contextFilter == "",
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := appStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)

	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch {
		case key.Matches(msg, m.keys.toggleHelpMenu):
			m.list.SetShowHelp(!m.list.ShowHelp())
			return m, nil
		}
	}

	// This will also call our delegate's update function.
	newListModel, cmd := m.list.Update(msg)
	m.list = newListModel

	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) Init() tea.Cmd {
	if m.useInitialFilter {
		return list.EnableLiveFiltering
	}
	return nil
}

func (m model) View() string {
	if m.quitting {
		return ""
	}
	return appStyle.Render(m.list.View())
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getOS(testing string) string {
	if testing != "" {
		return testing
	}
	return runtime.GOOS
}

func getConfigDir(testing string) (string, error) {
	// Use OS-specific environment variable to determine config dir
	configDir := os.Getenv("KTX_CONFIG_DIR")
	if configDir != "" {
		return configDir, nil
	}
	// If environment variable is not set, use OS-specific default locations
	switch getOS(testing) {
	case "windows":
		configDir = filepath.Join(os.Getenv("APPDATA"), "ktx")
	case "darwin":
		configDir = filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ktx")
	case "linux", "freebsd":
		configDir = filepath.Join(os.Getenv("HOME"), ".config", "ktx")
		if configPath, exists := os.LookupEnv("XDG_CONFIG_HOME"); exists && configPath != "" {
			configDir = filepath.Join(configPath, "ktx")
		}
	default:
		// Default to current working directory
		var err error
		configDir, err = os.Getwd()
		if err != nil {
			return "", err
		}
		configDir = filepath.Join(configDir, "ktx")
	}
	if testing == "" {
		// Create the config directory if it doesn't exist
		err := os.MkdirAll(configDir, 0755)
		if err != nil {
			return "", err
		}
	}
	return configDir, nil
}

func findKubeConfigs(testing string) []string {
	knf := os.Getenv("KUBECONFIG")
	if knf != "" {
		return strings.Split(knf, ":")
	}

	switch getOS(testing) {
	case "windows":
		knf = filepath.Join(os.Getenv("USERPROFILE"), ".kube", "config")
	default:
		knf = filepath.Join(os.Getenv("HOME"), ".kube", "config")
	}
	return []string{knf}
}

func generateAlias(path string) string {
	alias := strings.ReplaceAll(filepath.Ext(path), ".", "")
	if alias == "" {
		return "x"
	}
	return alias
}

func generateConfig(kubeConfigs []string) []*k8sctx.KubeConf {
	kConfs := []*k8sctx.KubeConf{}
	for _, knf := range kubeConfigs {
		k := &k8sctx.KubeConf{
			Path:  knf,
			Alias: generateAlias(knf),
		}
		kConfs = append(kConfs, k)
	}
	return kConfs
}

func initConfigFile(testing string) (string, error) {
	tmpl, err := template.New("").Parse(configJsonnet)
	if err != nil {
		return "", fmt.Errorf("parse config template %w", err)
	}
	kubeConfigs := findKubeConfigs(testing)
	var tpl bytes.Buffer
	if err := tmpl.Execute(&tpl, generateConfig(kubeConfigs)); err != nil {
		return "", fmt.Errorf("execute config template %w", err)
	}
	return tpl.String(), nil
}

func loadConfigs() (*k8sctx.Config, error) {
	configDir, err := getConfigDir("")
	if err != nil {
		return nil, err
	}
	configFile := filepath.Join(configDir, "config.jsonnet")
	if !fileExists(configFile) {
		cnf, err := initConfigFile("")
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(configFile, []byte(cnf), 0644); err != nil {
			return nil, err
		}
	}
	contextsFile := filepath.Join(configDir, ".libsonnet")
	if !fileExists(contextsFile) {
		if err := os.WriteFile(contextsFile, []byte(contextsLibsonnet), 0644); err != nil {
			return nil, err
		}
	}

	c, err := k8sctx.Get(configFile)
	if err != nil {
		return nil, err
	}
	if err := c.SyncNamespaces(); err != nil {
		return nil, err
	}
	reload := false
	for _, kcnf := range c.KubeConfs {
		if !fileExists(kcnf.ContextFile) {
			reload = true
		}
		if err := kcnf.KubeConfig.SyncContexts(kcnf); err != nil {
			return nil, fmt.Errorf("sync contexts: %w", err)
		}
	}

	if reload {
		c, err = k8sctx.Get(configFile)
		if err != nil {
			return nil, fmt.Errorf("read contexts file: %w", err)
		}
	}
	return c, nil
}

func run(filters []string) (string, error) {
	c, err := loadConfigs()
	if err != nil {
		return "", err
	}
	configFilter := ""
	contextFilter := ""

	switch len(filters) {
	case 1:
		configFilter = filters[0]
	case 2:
		configFilter, contextFilter = filters[0], filters[1]
	}

	if _, err := tea.NewProgram(modelFrom(c, configFilter, contextFilter), tea.WithAltScreen()).Run(); err != nil {
		return "", fmt.Errorf("error running program: %w", err)
	}
	return getCurrentContext()
}

func directlyUse(context string) (string, error) {
	c, err := loadConfigs()
	if err != nil {
		return "", err
	}

	kcnf, ctx, idx := c.GetContextBy(context)
	if idx == -1 {
		return "", fmt.Errorf("context '%s' not found in kube config files", context)
	}
	if err := c.RemoveCurrentContexts(); err != nil {
		return "", fmt.Errorf("failed to remove old current-contexts: %w", err)
	}
	if err := kcnf.KubeConfig.SetContextTo(ctx["name"]); err != nil {
		return "", fmt.Errorf("failed to set current-context to '%s': %w", context, err)
	}
	if err := c.UpdateState(kcnf, ctx["name"]); err != nil {
		return "", fmt.Errorf("failed to update state file '%s': %w", c.Filename, err)
	}
	return context, nil
}

func switchBack() (string, error) {
	c, err := loadConfigs()
	if err != nil {
		return "", err
	}
	if err := c.GetState(); err != nil {
		return "", err
	}
	kcnf := c.GetKubeConfigBy(c.LastConf)
	if kcnf == nil {
		return noContextFound, nil
	}
	lastContext := c.LastContext

	if err := c.RemoveCurrentContexts(); err != nil {
		return "", err
	}

	if err := kcnf.KubeConfig.SetContextTo(lastContext); err != nil {
		return "", fmt.Errorf("using previous context failed while setting context: %w", err)
	}
	if err := c.UpdateState(kcnf, lastContext); err != nil {
		return "", fmt.Errorf("using previous context failed while updating state file: %w", err)
	}
	return lastContext, nil
}

func getCurrentContext() (string, error) {
	c, err := loadConfigs()
	if err != nil {
		return "", err
	}

	if err := c.GetState(); err != nil {
		return "", fmt.Errorf("getting current context failed while reading state file: %w", err)
	}
	kcnf := c.GetKubeConfigBy(c.CurrentConf)
	if kcnf == nil {
		return noContextFound, nil
	}
	return c.CurrentContext, nil
}

func runWith(args []string) (string, error) {
	if len(args) > 1 {
		switch args[1] {
		case "-h", "-help":
			return helpText, nil
		case "-v", "-version":
			return version, nil
		case "-is", "-current", "-c":
			if len(args) > 2 {
				return directlyUse(args[2])
			}
			return getCurrentContext()
		case "-":
			return switchBack()
		}
	}
	return run(args[1:])
}

func main() {
	msg, err := runWith(os.Args)
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Println(msg)
}
