package k8sctx

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	kCnf = &KubeConfig{
		Path:        "testdata/kube.config",
		APIVersion:  "v1",
		Kind:        "Config",
		Preferences: map[string]interface{}{},
		Contexts: []KubeContext{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				Context: &Context{
					Cluster: "aws:dev:accountId:eu-central-1:cluster1",
					User:    "aws:dev:accountId:eu-central-1:cluster1",
				},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				Context: &Context{
					Cluster: "aws:prod:accountId:us-east-1:cluster1",
					User:    "aws:prod:accountId:us-east-1:cluster1",
				},
			},
		},
		CurrentContext: "aws:prod:accountId:us-east-1:cluster1",
		Clusters: []struct {
			Name    string      `yaml:"name"`
			Cluster interface{} `yaml:"cluster"`
		}{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				Cluster: map[string]interface{}{
					"certificate-authority-data": 1234,
					"server":                     "http://localhost",
				},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				Cluster: map[string]interface{}{
					"certificate-authority-data": "abcd",
					"server":                     "http://localhost",
				},
			},
		},
		Users: []struct {
			Name string      `yaml:"name"`
			User interface{} `yaml:"user"`
		}{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				User: map[string]interface{}{},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				User: map[string]interface{}{},
			},
		},
	}
	kCnfChanged = &KubeConfig{
		Path:        "testdata/kube.config.bak",
		APIVersion:  "v1",
		Kind:        "Config",
		Preferences: map[string]interface{}{},
		Contexts: []KubeContext{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				Context: &Context{
					Cluster: "aws:dev:accountId:eu-central-1:cluster1",
					User:    "aws:dev:accountId:eu-central-1:cluster1",
				},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				Context: &Context{
					Cluster:   "aws:prod:accountId:us-east-1:cluster1",
					User:      "aws:prod:accountId:us-east-1:cluster1",
					Namespace: "monitoring", // <---------------------- change
				},
			},
		},
		CurrentContext: "aws:prod:accountId:us-east-1:cluster1",
		Clusters: []struct {
			Name    string      `yaml:"name"`
			Cluster interface{} `yaml:"cluster"`
		}{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				Cluster: map[string]interface{}{
					"certificate-authority-data": 1234,
					"server":                     "http://localhost",
				},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				Cluster: map[string]interface{}{
					"certificate-authority-data": "abcd",
					"server":                     "http://localhost",
				},
			},
		},
		Users: []struct {
			Name string      `yaml:"name"`
			User interface{} `yaml:"user"`
		}{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				User: map[string]interface{}{},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				User: map[string]interface{}{},
			},
		},
	}
)

func TestGet(t *testing.T) {
	type args struct {
		contextConfig string
	}
	tests := []struct {
		name       string
		args       args
		want       *Config
		wantErr    bool
		wantErrMsg error
	}{
		{
			name: "positive - read jsonnet file",
			args: args{
				contextConfig: "testdata/config.jsonnet",
			},
			want: &Config{
				GlobalConfig: "testdata/config.jsonnet",
				KubeConfs: []*KubeConf{
					{
						Path:        "testdata/kube.config",
						Alias:       "t",
						ContextFile: "contexts.yaml",
						KubeConfig:  kCnf,
						Contexts: []map[string]string{
							{
								"alias":       "avap:1",
								"cluster":     "cluster1",
								"environment": "prod",
								"name":        "aws:prod:accountId:us-east-1:cluster1",
								"namespace":   "monitoring",
								"region":      "us-east-1",
							},
						},
					},
				},
			},
			wantErr:    false,
			wantErrMsg: nil,
		},
		{
			name: "negative - ErrReadContextConfig",
			args: args{
				contextConfig: "testdata/config_broken.jsonnet",
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: ErrReadConfig,
		},
		{
			name: "negative - ErrParseContextConfig",
			args: args{
				// if file not found, fallback is to empty array which cannot be read by json
				contextConfig: "testdata/does not exists",
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: ErrParseConfig,
		},
		{
			name: "negative - ErrReadKubeConfig",
			args: args{
				contextConfig: "testdata/config_wrong_kubeconfig.jsonnet",
			},
			want:       nil,
			wantErr:    true,
			wantErrMsg: ErrReadKubeConfig,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Get(tt.args.contextConfig)
			if (err != nil) != tt.wantErr {
				t.Errorf("Get() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr == true {
				assert.ErrorIs(t, err, tt.wantErrMsg)
				return
			}
			assert.Equal(t, tt.want.GlobalConfig, got.GlobalConfig)
			if len(tt.want.KubeConfs) != len(got.KubeConfs) {
				t.Errorf("Get() KubeConfs are not the same")
				return
			}
			for idx, g := range got.KubeConfs {

				assert.Equal(t, tt.want.KubeConfs[idx].Contexts, g.Contexts)
				assert.Equal(t, tt.want.KubeConfs[idx].Alias, g.Alias)
				assert.Equal(t, tt.want.KubeConfs[idx].ContextFile, g.ContextFile)
				assert.Equal(t, tt.want.KubeConfs[idx].KubeConfig.Contexts, g.KubeConfig.Contexts)
				assert.Equal(t, tt.want.KubeConfs[idx].KubeConfig.Users, g.KubeConfig.Users)
				assert.Equal(t, tt.want.KubeConfs[idx].KubeConfig.APIVersion, g.KubeConfig.APIVersion)
			}
		})
	}
}

func TestKubeConf_SyncNamespaces(t *testing.T) {

	kubeConfig := kCnf
	// to not override the original one
	kubeConfig.Path = "testdata/kube.config.bak"

	type fields struct {
		Path        string
		Alias       string
		ContextFile string
		Contexts    []map[string]string
		kubeConfig  *KubeConfig
	}
	tests := []struct {
		name           string
		fields         fields
		wantErr        bool
		wantKubeConfig *KubeConfig
	}{
		{
			name: "positive",
			fields: fields{
				Path:        "testdata/kube.config",
				Alias:       "",
				ContextFile: "",
				Contexts: []map[string]string{
					{
						"alias":       "avap:1",
						"cluster":     "cluster1",
						"environment": "prod",
						"name":        "aws:prod:accountId:us-east-1:cluster1",
						"namespace":   "monitoring",
						"region":      "us-east-1",
					},
				},
				kubeConfig: kubeConfig,
			},
			wantErr:        false,
			wantKubeConfig: kCnfChanged,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			k := &KubeConf{
				Path:        tt.fields.Path,
				Alias:       tt.fields.Alias,
				ContextFile: tt.fields.ContextFile,
				Contexts:    tt.fields.Contexts,
				KubeConfig:  tt.fields.kubeConfig,
			}
			if err := k.SyncNamespaces(); (err != nil) != tt.wantErr {
				t.Errorf("KubeConf.SyncNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}
			got, err := GetKubeConfig("testdata/kube.config.bak")
			if (err != nil) != tt.wantErr {
				t.Fatalf("KubeConf.SyncNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}

			assert.Equal(t, tt.wantKubeConfig.Contexts, got.Contexts)
			assert.Equal(t, tt.wantKubeConfig.Users, got.Users)
			assert.Equal(t, tt.wantKubeConfig.Clusters, got.Clusters)
			assert.Equal(t, tt.wantKubeConfig.Preferences, got.Preferences)
			assert.Equal(t, tt.wantKubeConfig.APIVersion, got.APIVersion)
			assert.Equal(t, tt.wantKubeConfig.Kind, got.Kind)
			assert.Equal(t, tt.wantKubeConfig.CurrentContext, got.CurrentContext)
		})
	}
	t.Cleanup(func() { os.RemoveAll("testdata/kube.config.bak") })
}

func TestContextConfig_GetContextBy(t *testing.T) {
	type fields struct {
		Contexts []map[string]string
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *KubeConf
		want1  map[string]string
		want2  int
	}{
		{
			name: "positive",
			fields: fields{
				Contexts: []map[string]string{
					{"name": "a"},
					{"name": "b"},
					{"name": "c"},
				},
			},
			args: args{name: "b"},
			want: &KubeConf{
				Path:        "",
				Alias:       "",
				ContextFile: "",
				Contexts: []map[string]string{
					{"name": "a"},
					{"name": "b"},
					{"name": "c"},
				},
				KubeConfig: nil,
			},
			want1: map[string]string{"name": "b"},
			want2: 1,
		},
		{
			name: "positive - alias",
			fields: fields{
				Contexts: []map[string]string{
					{"name": "a"},
					{"name": "e", "alias": "b"},
					{"name": "c"},
				},
			},
			args: args{name: "b"},
			want: &KubeConf{
				Path:        "",
				Alias:       "",
				ContextFile: "",
				Contexts: []map[string]string{
					{"name": "a"},
					{"name": "e", "alias": "b"},
					{"name": "c"},
				},
				KubeConfig: nil,
			},
			want1: map[string]string{"name": "e", "alias": "b"},
			want2: 1,
		},
		{
			name: "positive - empty",
			fields: fields{
				Contexts: []map[string]string{},
			},
			args:  args{name: "b"},
			want:  nil,
			want1: nil,
			want2: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				KubeConfs: []*KubeConf{
					{
						Contexts: tt.fields.Contexts,
					},
				},
			}
			got, got1, got2 := c.GetContextBy(tt.args.name)
			if got2 != tt.want2 {
				t.Errorf("Config.GetContextBy() got = %v, want %v", got2, tt.want2)
			}
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.want1, got1)
			assert.Equal(t, tt.want2, got2)
		})
	}
}

func TestConfig_SyncNamespaces(t *testing.T) {
	kubeConfig := kCnf
	// to not override the original one
	kubeConfig.Path = "testdata/kube.config.bak"
	type fields struct {
		Dir          string
		GlobalConfig string
		KubeConfs    []*KubeConf
		State        *State
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantErrMsg error
	}{
		{
			name: "positive",
			fields: fields{
				Dir:          "testdata",
				GlobalConfig: "testdata/config.jsonnet",
				KubeConfs: []*KubeConf{
					{
						Path:        "testdata/kube.config",
						Alias:       "",
						ContextFile: "",
						Contexts: []map[string]string{
							{
								"alias":       "avap:1",
								"cluster":     "cluster1",
								"environment": "prod",
								"name":        "aws:prod:accountId:us-east-1:cluster1",
								"namespace":   "monitoring",
								"region":      "us-east-1",
							},
						},
						KubeConfig: kubeConfig,
					},
				},
				State: nil,
			},
			wantErr: false,
		},
		{
			name: "negative - ErrNoContext",
			fields: fields{
				Dir:          "testdata",
				GlobalConfig: "testdata/config.jsonnet",
				KubeConfs: []*KubeConf{
					{
						Path: "testdata/kube.config",
						Contexts: []map[string]string{
							{
								"alias":     "no exists",
								"namespace": "monitoring",
							},
						},
						KubeConfig: kubeConfig,
					},
				},
				State: nil,
			},
			wantErr:    true,
			wantErrMsg: ErrNoContext,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Dir:          tt.fields.Dir,
				GlobalConfig: tt.fields.GlobalConfig,
				KubeConfs:    tt.fields.KubeConfs,
				State:        tt.fields.State,
			}
			err := c.SyncNamespaces()
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.SyncNamespaces() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.wantErrMsg)
			}
		})
	}
}

func Test_home(t *testing.T) {
	type args struct {
		path string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "tilde",
			args: args{path: "~/.kube"},
			want: filepath.Join(os.Getenv("HOME"), ".kube"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := home(tt.args.path); got != tt.want {
				t.Errorf("home() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_setup(t *testing.T) {
	type fields struct {
		GlobalConfig string
		KubeConfs    []*KubeConf
	}
	tests := []struct {
		name   string
		fields fields
		want   *Config
	}{
		{
			name: "positive",
			fields: fields{
				GlobalConfig: "testdata/config.jsonnet",
				KubeConfs: []*KubeConf{
					{
						Alias: "m",
					},
				},
			},
			want: &Config{
				Dir:          "testdata",
				GlobalConfig: "testdata/config.jsonnet",
				State:        &State{Filename: "testdata/.state"},
				KubeConfs: []*KubeConf{
					{
						Alias:       "m",
						ContextFile: "testdata/contexts_m.yaml",
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				GlobalConfig: tt.fields.GlobalConfig,
				KubeConfs:    tt.fields.KubeConfs,
			}
			c.setup()

			assert.Equal(t, tt.want, c)
		})
	}
}

func TestConfig_GetKubeConfigBy(t *testing.T) {
	type fields struct {
		Dir          string
		GlobalConfig string
		KubeConfs    []*KubeConf
		State        *State
	}
	type args struct {
		path string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *KubeConf
	}{
		{
			name: "positive",
			fields: fields{
				KubeConfs: []*KubeConf{
					{
						Path: "testdata/kube.config",
					},
				},
				State: nil,
			},
			args: args{path: "testdata/kube.config"},
			want: &KubeConf{
				Path: "testdata/kube.config",
			},
		},
		{
			name: "empty",
			fields: fields{
				KubeConfs: []*KubeConf{
					{
						Path: "testdata/kube.config",
					},
				},
				State: nil,
			},
			args: args{path: ""},
			want: nil,
		},
		{
			name: "no match",
			fields: fields{
				KubeConfs: []*KubeConf{
					{
						Path: "testdata/kube.config",
					},
				},
				State: nil,
			},
			args: args{path: "not exists"},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Dir:          tt.fields.Dir,
				GlobalConfig: tt.fields.GlobalConfig,
				KubeConfs:    tt.fields.KubeConfs,
				State:        tt.fields.State,
			}
			if got := c.GetKubeConfigBy(tt.args.path); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Config.GetKubeConfigBy() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKubeConf_GetContextBy(t *testing.T) {
	type fields struct {
		Path        string
		Alias       string
		ContextFile string
		Contexts    []map[string]string
		KubeConfig  *KubeConfig
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]string
		want1  int
	}{
		{
			name: "alias",
			fields: fields{
				Contexts: []map[string]string{
					{
						"alias":     "alias1",
						"name":      "aws:prod:accountId:us-east-1:cluster1",
						"namespace": "monitoring",
					},
					{
						"alias":     "alias2",
						"name":      "aws:dev:accountId:us-east-1:cluster2",
						"namespace": "default",
					},
				},
			},
			args: args{
				name: "alias2",
			},
			want: map[string]string{
				"alias":     "alias2",
				"name":      "aws:dev:accountId:us-east-1:cluster2",
				"namespace": "default",
			},
			want1: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := &KubeConf{
				Path:        tt.fields.Path,
				Alias:       tt.fields.Alias,
				ContextFile: tt.fields.ContextFile,
				Contexts:    tt.fields.Contexts,
				KubeConfig:  tt.fields.KubeConfig,
			}
			got, got1 := k.GetContextBy(tt.args.name)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("KubeConf.GetContextBy() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("KubeConf.GetContextBy() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestConfig_UpdateState(t *testing.T) {
	type fields struct {
		Dir          string
		GlobalConfig string
		KubeConfs    []*KubeConf
		State        *State
	}
	type args struct {
		k              *KubeConf
		currentContext string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "positive",
			fields: fields{
				State: &State{
					Filename: "testdata/state",
				},
			},
			args:    args{k: &KubeConf{Path: "testdata/kube.config"}, currentContext: "a"},
			wantErr: false,
		},
		{
			name: "negative - yaml broken",
			fields: fields{
				State: &State{
					Filename: "testdata/state.broken",
				},
			},
			args:    args{k: &KubeConf{Path: "testdata/kube.config"}, currentContext: "a"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				Dir:          tt.fields.Dir,
				GlobalConfig: tt.fields.GlobalConfig,
				KubeConfs:    tt.fields.KubeConfs,
				State:        tt.fields.State,
			}
			if err := c.UpdateState(tt.args.k, tt.args.currentContext); (err != nil) != tt.wantErr {
				t.Errorf("Config.UpdateState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	t.Cleanup(func() { os.RemoveAll("testdata/state") })
}

func TestConfig_RemoveCurrentContexts(t *testing.T) {
	type fields struct {
		KubeConfs []*KubeConf
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "positive",
			fields: fields{
				KubeConfs: []*KubeConf{
					{
						KubeConfig: kCnfChanged,
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				KubeConfs: tt.fields.KubeConfs,
			}
			if err := c.RemoveCurrentContexts(); (err != nil) != tt.wantErr {
				t.Errorf("Config.RemoveCurrentContexts() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err := kCnfChanged.Read(); err != nil {
				t.Errorf("Config.RemoveCurrentContexts() error = %v", err)
			}
			if kCnfChanged.CurrentContext != "" {
				t.Errorf("Config.RemoveCurrentContexts() currentContext was not removed.")
			}

		})
	}
	t.Cleanup(func() { os.RemoveAll("testdata/kube.config.bak") })
}

func TestConfig_CreateListItems(t *testing.T) {
	type fields struct {
		KubeConfs []*KubeConf
	}
	type args struct {
		filterConfig  string
		filterContext string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []ContextItem
	}{
		{
			name: "positive - no filter",
			fields: fields{
				KubeConfs: []*KubeConf{
					{
						Contexts: []map[string]string{
							{
								"alias":     "alias1",
								"name":      "aws:prod:accountId:us-east-1:cluster1",
								"namespace": "monitoring",
							},
							{
								"alias":     "alias2",
								"name":      "aws:dev:accountId:us-east-1:cluster2",
								"namespace": "default",
							},
						},
					},
					{
						Contexts: []map[string]string{
							{
								"alias":     "alias3",
								"name":      "aws:prod:accountId:us-east-1:cluster3",
								"namespace": "monitoring",
							},
							{
								"alias":     "alias4",
								"name":      "aws:dev:accountId:us-east-1:cluster4",
								"namespace": "default",
							},
						},
					},
				},
			},
			args: args{
				filterConfig:  "",
				filterContext: "",
			},
			want: []ContextItem{
				{Name: "alias1", Description: "namespace: monitoring"}, {Name: "alias2", Description: "namespace: default"},
				{Name: "alias3", Description: "namespace: monitoring"}, {Name: "alias4", Description: "namespace: default"},
			},
		},
		{
			name: "positive - filter",
			fields: fields{
				KubeConfs: []*KubeConf{
					{
						Alias: "t",
						Contexts: []map[string]string{
							{
								"alias":     "alias1",
								"name":      "aws:prod:accountId:us-east-1:cluster1",
								"namespace": "monitoring",
							},
							{
								"alias":     "alias2",
								"name":      "aws:dev:accountId:us-east-1:cluster2",
								"namespace": "default",
							},
						},
					},
					{
						Alias: "x",
						Contexts: []map[string]string{
							{
								"alias":     "alias3",
								"name":      "aws:prod:accountId:us-east-1:cluster3",
								"namespace": "monitoring",
							},
							{
								"alias":     "alias4",
								"name":      "aws:dev:accountId:us-east-1:cluster4",
								"namespace": "default",
							},
						},
					},
				},
			},
			args: args{
				filterConfig:  "t",
				filterContext: "alias2",
			},
			want: []ContextItem{
				{Name: "alias2", Description: "namespace: default"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				KubeConfs: tt.fields.KubeConfs,
			}
			got := c.CreateListItems(tt.args.filterConfig, tt.args.filterContext)
			assert.Equal(t, tt.want, got)
		})
	}
}
