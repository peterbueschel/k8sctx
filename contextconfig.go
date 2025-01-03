package k8sctx

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	jsonnet "github.com/google/go-jsonnet"
	importer "github.com/peterbueschel/jsonnet-custom-importers"
	"gopkg.in/yaml.v3"
)

var (
	ErrReadConfig     = errors.New("failed to read from context config file")
	ErrParseConfig    = errors.New("failed to parse context config file")
	ErrReadStateFile  = errors.New("failed to parse state file")
	ErrParseStateFile = errors.New("failed to parse state file")
)

type (
	// Config is the main struct, which is generated from the config file and
	// holds one or more kube configs.
	Config struct {
		// Dir of the config files
		Dir string
		// GlobalConfig holds the jsonnet file path of the context config
		GlobalConfig string
		// KubeConfs is list of kube config items. With allows you to
		// handle multiple kube configs with a single "GlobalConfig".
		KubeConfs []*KubeConf `json:"kube_configs"`
		// State contains the name of the state file
		// LastConf, CurrentContext and LastContext
		*State `json:"state"`
	}
	// State is used to switch back to the previous contexts.
	State struct {
		// Filename of the state file
		Filename string `yaml:"filename"`
		// CurrentConf holds the name of the curently used kube config. This
		// is used to jump back via "-" argument of the ktx cli command.
		CurrentConf string `yaml:"currentKubeConfig"`
		// LastConf stores the name of the previous used kube config. This
		// is used to jump back via "-" argument of the ktx cli command.
		LastConf string `yaml:"lastKubeConfig"`
		// CurrentContext stores the name of the currently used Kubernetes context.
		CurrentContext string `yaml:"currentContext"`
		// LastContext together with LastConf will be used to switch between
		// two contexts.
		LastContext string `yaml:"lastContext"`
	}
	// KubeConf stores the content of a single kube config file.
	KubeConf struct {
		// Path of the kube config file
		Path string `json:"path"`
		// Alias the kube config file
		Alias string `json:"alias"`
		// ContextFile holds the path of the context file in yaml format
		ContextFile string `json:"context_file"`
		// Contexts will be synced with the kube config file
		// and extended by the GlobalConfig file
		Contexts []map[string]string `json:"contexts"`
		// KubeConfig holds the kube config file content
		KubeConfig *KubeConfig `json:"-"`
	}
	// ContextItem is used in the "list" Model in the cmd/ktx.
	ContextItem struct {
		Name        string
		Description string
	}
)

// Get reads the config.jsonnet file and the bound kube config files.
// In addition, if not exist, it creates the contexts and state files.
func Get(config string) (*Config, error) {
	cnf, err := read(config)
	if err != nil {
		return nil, fmt.Errorf("%w: '%s', err: %w", ErrReadConfig, config, err)
	}
	parsedConfig := &Config{}
	if err = json.Unmarshal([]byte(cnf), &parsedConfig); err != nil {
		return nil, fmt.Errorf("%w: '%s', err: %w", ErrParseConfig, config, err)
	}
	parsedConfig.GlobalConfig = config
	parsedConfig.setup()

	for _, cnf := range parsedConfig.KubeConfs {
		k, err := GetKubeConfig(cnf.Path)
		if err != nil {
			return nil, err
		}
		cnf.KubeConfig = k
	}
	return parsedConfig, nil
}

// setup generates the file name and path for the .state and the contexts_...
// files and stores them in the Config.
func (c *Config) setup() {
	c.Dir = filepath.Dir(c.GlobalConfig)
	if c.State == nil {
		c.State = &State{
			Filename: filepath.Join(c.Dir, ".state"),
		}
	}
	for _, cnf := range c.KubeConfs {
		if cnf.ContextFile == "" {
			cnf.ContextFile = filepath.Join(
				c.Dir, fmt.Sprintf("contexts_%s.yaml", cnf.Alias),
			)
		}
	}
}

// read evaluates the jsonnet config file with the help of some custom importers.
func read(jsonnetFile string) (string, error) {
	jPath := filepath.Dir(jsonnetFile)
	g := importer.NewGlobImporter(jPath)
	f := importer.NewFallbackFileImporter(jPath)
	m := importer.NewMultiImporter(g, f)
	m.IgnoreImportCycles()
	m.OnMissingFile("'[]'")

	vm := jsonnet.MakeVM()
	vm.Importer(m)
	vm.ErrorFormatter.SetColorFormatter(color.New(color.FgRed).Fprintf)
	return vm.EvaluateFile(jsonnetFile)
}

// SyncNamespaces loops over the KubeConf and runs in turn the SyncNamespaces
// for every kube config.
func (c *Config) SyncNamespaces() error {
	for _, cnf := range c.KubeConfs {
		if err := cnf.SyncNamespaces(); err != nil {
			return err
		}
	}
	return nil
}

// SyncNamespaces loops over the contexts and updates the namespace in the
// underlying kube config file.
func (k *KubeConf) SyncNamespaces() error {
	for _, ctx := range k.Contexts {
		if err := k.KubeConfig.AddNamespaceTo(ctx["name"], ctx["namespace"]); err != nil {
			return err
		}
	}
	return nil
}

// GetContextBy takes a name or alias of the desired context as argument and
// returns the KubeConf, the context and its index.
func (c *Config) GetContextBy(name string) (*KubeConf, map[string]string, int) {
	for _, cnf := range c.KubeConfs {
		for idx, ctx := range cnf.Contexts {
			if ctx["name"] == name {
				return cnf, ctx, idx
			}
			if alias, exists := ctx["alias"]; exists {
				if alias == name {
					return cnf, ctx, idx
				}
			}
		}
	}
	return nil, nil, -1
}

// RemoveCurrentContexts removes from every kube config the setting for the
// currentContext.
func (c *Config) RemoveCurrentContexts() error {
	for _, k := range c.KubeConfs {
		if err := k.KubeConfig.RemoveCurrentContext(); err != nil {
			return err
		}
	}
	return nil
}

// GetKubeConfigBy returns the KubeConf by a given path.
func (c *Config) GetKubeConfigBy(path string) *KubeConf {
	if path == "" {
		return nil
	}
	for _, k := range c.KubeConfs {
		if k.Path == path {
			return k
		}
	}
	return nil
}

// GetContextBy returns the context and its index within a single KubeConf given
// by name.
func (k *KubeConf) GetContextBy(name string) (map[string]string, int) {
	for idx, ctx := range k.Contexts {
		if ctx["name"] == name {
			return ctx, idx
		}
		if alias, exists := ctx["alias"]; exists && alias == name {
			return ctx, idx
		}
	}
	return nil, -1
}

// Exists returns true if a context with the given name exists.
func (k *KubeConf) Exists(name string) bool {
	if _, exists := k.GetContextBy(name); exists == -1 {
		return false
	}
	return true
}

// Save writes the contexts stored in a KubeConfig into a contexts_<alias>.yaml
// file.
func (k *KubeConf) Save() error {
	cnf, err := yaml.Marshal(&k.Contexts)
	if err != nil {
		return err
	}
	return os.WriteFile(k.ContextFile, cnf, 0644)
}

// UpdateState stores the actual kube config and context under the lastConfig
// and lastContext inside the .state file. At the same time it updates the
// values for the current config and current context.
func (c *Config) UpdateState(k *KubeConf, currentContext string) error {
	if err := c.GetState(); err != nil {
		return err
	}
	lastConf := c.CurrentConf
	lastContext := c.CurrentContext

	c.CurrentConf = k.Path
	c.CurrentContext = currentContext

	c.LastConf = lastConf
	c.LastContext = lastContext

	cnf, err := yaml.Marshal(&c.State)
	if err != nil {
		return err
	}
	return os.WriteFile(c.Filename, cnf, 0644)
}

// GetState stores the current state file content into the config. It returns
// ErrReadStateFile or ErrReadStateFile if the .state file cannot be read or
// parsed.
func (c *Config) GetState() error {
	f, err := os.ReadFile(c.Filename)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%w: '%s', err: %w", ErrReadStateFile, c.Filename, err)
	}
	err = yaml.Unmarshal(f, &c.State)
	if err != nil {
		return fmt.Errorf("%w: '%s', err: %w", ErrParseStateFile, c.Filename, err)
	}
	return nil
}

// CreateListItems is a helper function for the TUI and creates the list of
// context names and a description.
func (c *Config) CreateListItems(filterConfig, filterContext string) []ContextItem {
	items := []ContextItem{}
	for _, cnf := range c.KubeConfs {
		if filterConfig != "" && cnf.Alias != filterConfig {
			continue
		}
		for _, ctx := range cnf.Contexts {
			name := ctx["name"]
			if alias, exists := ctx["alias"]; exists {
				name = alias
			}
			if filterContext != "" && !strings.Contains(name, filterContext) {
				continue
			}
			descriptions := []string{}
			for k, v := range ctx {
				if k != "name" && k != "alias" {
					descriptions = append(descriptions, fmt.Sprintf("%s: %s", k, v))
				}
			}
			i := ContextItem{
				Name:        name,
				Description: strings.Join(descriptions, ", "),
			}
			items = append(items, i)
		}
	}
	return items
}
