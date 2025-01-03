package k8sctx

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type (
	// KubeContext holds a single context.
	KubeContext struct {
		Name     string `yaml:"name"`
		*Context `yaml:"context"`
	}
	// Context item.
	Context struct {
		Cluster   string `yaml:"cluster"`
		User      string `yaml:"user"`
		Namespace string `yaml:"namespace,omitempty"`
	}
	// KubeConfig holds the content of the kube config file.
	KubeConfig struct {
		// Path holds the file path of the kube config.
		Path string `yaml:"-"`
		// APIVersion comes from the underlying K8s Kube Config specs.
		APIVersion string `yaml:"apiVersion"`
		// Kind comes from the underlying K8s Kube Config specs.
		Kind string `yaml:"kind"`
		// Preferences comes from the underlying K8s Kube Config specs.
		Preferences interface{} `yaml:"preferences"`
		// Contexts comes from the underlying K8s Kube Config specs and will be
		// next to the CurrentContext interesting part for us.
		Contexts []KubeContext `yaml:"contexts"`
		// CurrentContext contains the desired context name.
		CurrentContext string `yaml:"current-context"`
		// Clusters comes from the underlying K8s Kube Config specs.
		Clusters []struct {
			Name    string      `yaml:"name"`
			Cluster interface{} `yaml:"cluster"`
		} `yaml:"clusters"`
		// Users comes from the underlying K8s Kube Config specs.
		Users []struct {
			Name string      `yaml:"name"`
			User interface{} `yaml:"user"`
		} `yaml:"users"`
	}
	KubeConfigs []KubeConfig
)

var (
	ErrReadKubeConfig  = errors.New("failed to read from kube config file")
	ErrParseKubeConfig = errors.New("failed to parse kube config file")
	ErrDuplContext     = errors.New("duplicated context name")
	ErrNoContext       = errors.New("no context found")
)

func home(path string) string {
	if strings.HasPrefix(path, "~/") {
		dirname, _ := os.UserHomeDir()
		return filepath.Join(dirname, path[2:])
	}
	return path
}

func GetKubeConfig(kubeconfig string) (*KubeConfig, error) {
	k := &KubeConfig{Path: home(kubeconfig)}
	err := k.Read()
	return k, err
}

func (k *KubeConfig) Read() error {
	f, err := os.ReadFile(k.Path)
	if err != nil {
		return fmt.Errorf("%w: '%s', err: %w", ErrReadKubeConfig, k.Path, err)
	}

	err = yaml.Unmarshal(f, &k)
	if err != nil {
		return fmt.Errorf("%w: '%s', err: %w", ErrParseKubeConfig, k.Path, err)
	}

	dupl := make(map[string]int)
	for _, ctx := range k.Contexts {
		dupl[ctx.Name]++
	}
	for d, v := range dupl {
		if v > 1 {
			return fmt.Errorf("%w %s in %s", ErrDuplContext, d, k.Path)
		}
	}

	sort.Slice(k.Contexts, func(i, j int) bool {
		return k.Contexts[i].Name < k.Contexts[j].Name
	})
	return nil
}

// GetProfilesNames returns all context names
func (k *KubeConfig) GetContextNames() (names []string) {
	for _, p := range k.Contexts {
		names = append(names, p.Name)
	}
	return
}

// GetContextBy returns the kube context by a given name
func (k *KubeConfig) GetContextBy(name string) (*KubeContext, int, error) {
	for idx, p := range k.Contexts {
		if p.Name == name {
			if p.Context == nil {
				p.Context = &Context{}
			}
			return &p, idx, nil
		}
	}
	return &KubeContext{}, -1, fmt.Errorf("%w with name '%s'", ErrNoContext, name)
}

func (k *KubeConfig) SetContextTo(contextName string) error {
	_, _, err := k.GetContextBy(contextName)
	if err != nil {
		return err
	}
	k.CurrentContext = contextName
	return k.SaveContexts()
}

func (k *KubeConfig) RemoveCurrentContext() error {
	k.CurrentContext = ""
	return k.SaveContexts()
}

func (k *KubeConfig) AddNamespaceTo(contextName, namespace string) error {
	if namespace == "" {
		return nil
	}
	ctx, idx, err := k.GetContextBy(contextName)
	if err != nil {
		return err
	}

	if ctx.Namespace == namespace {
		return nil
	}
	ctx.Namespace = namespace
	k.Contexts[idx] = *ctx
	return k.SaveContexts()
}

func (k *KubeConfig) SaveContexts() error {
	cnf, err := yaml.Marshal(&k)
	if err != nil {
		return err
	}
	return os.WriteFile(k.Path, cnf, 0644)
}

func (k *KubeConfig) SyncContexts(c *KubeConf) error {
	names := k.GetContextNames()
	for _, n := range names {
		if c.Exists(n) {
			continue
		}
		ctx, _, err := k.GetContextBy(n)
		if err != nil {
			return err
		}
		c.Contexts = append(c.Contexts, map[string]string{"name": ctx.Name, "kubeconfig": k.Path})
	}
	return c.Save()
}
