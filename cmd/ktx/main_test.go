package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"testing"

	"github.com/peterbueschel/k8sctx"
	"github.com/stretchr/testify/assert"
)

var (
	kCnf = &k8sctx.KubeConfig{
		Path:        "testdata/kube.config",
		APIVersion:  "v1",
		Kind:        "Config",
		Preferences: map[string]interface{}{},
		Contexts: []k8sctx.KubeContext{
			{
				Name: "aws:dev:accountId:eu-central-1:cluster1",
				Context: &k8sctx.Context{
					Cluster: "aws:dev:accountId:eu-central-1:cluster1",
					User:    "aws:dev:accountId:eu-central-1:cluster1",
				},
			},
			{
				Name: "aws:prod:accountId:us-east-1:cluster1",
				Context: &k8sctx.Context{
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
)

func Test_getConfigDir(t *testing.T) {

	currDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fallback := filepath.Join(currDir, "ktx")

	type args struct {
		testing string
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		setEnv      string
		setEnvValue string
	}{
		{
			name:        "per environment variable",
			args:        args{testing: "xxx"},
			want:        "conf/ktx",
			wantErr:     false,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "conf/ktx",
		},
		{
			name:        "windows",
			args:        args{testing: "windows"},
			want:        `%APPDATA%/ktx`, // using "/" instead of "\" for simplicity
			wantErr:     false,
			setEnv:      "APPDATA",
			setEnvValue: "%APPDATA%",
		},
		{
			name:    "darwin",
			args:    args{testing: "darwin"},
			want:    filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "ktx"),
			wantErr: false,
			setEnv:  "ignore",
		},
		{
			name:    "linux - no XDG_CONFIG_HOME",
			args:    args{testing: "linux"},
			want:    filepath.Join(os.Getenv("HOME"), ".config", "ktx"),
			wantErr: false,
			setEnv:  "XDG_CONFIG_HOME",
		},
		{
			name:        "linux",
			args:        args{testing: "linux"},
			want:        "local/ktx",
			wantErr:     false,
			setEnv:      "XDG_CONFIG_HOME",
			setEnvValue: "local",
		},
		{
			name:    "fallback",
			args:    args{testing: "xxx"},
			want:    fallback,
			wantErr: false,
			setEnv:  "ignore",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.setEnv, tt.setEnvValue)
			got, err := getConfigDir(tt.args.testing)
			if (err != nil) != tt.wantErr {
				t.Errorf("getConfigDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getConfigDir() = %v, want %v", got, tt.want)
			}
			t.Cleanup(func() { os.Unsetenv(tt.setEnv) })
		})
	}
}

func Test_loadConfigs(t *testing.T) {
	tests := []struct {
		name        string
		want        *k8sctx.Config
		wantErr     bool
		setEnv      string
		setEnvValue string
	}{
		{
			name: "positive - initial",
			want: &k8sctx.Config{
				Dir:          "testdata",
				GlobalConfig: "testdata/config.jsonnet",
				KubeConfs: []*k8sctx.KubeConf{
					{
						Path:        "testdata/kube.config",
						Alias:       "t",
						ContextFile: "testdata/contexts_t.yaml",
						KubeConfig:  kCnf,
						Contexts: []map[string]string{
							{
								"kubeconfig": "testdata/kube.config",
								"name":       "aws:dev:accountId:eu-central-1:cluster1",
								"index":      "0",
							},
							{
								"kubeconfig": "testdata/kube.config",
								"name":       "aws:prod:accountId:us-east-1:cluster1",
								"index":      "1",
							},
						},
					},
				},
				State: &k8sctx.State{
					Filename: "testdata/.state",
				},
			},
			wantErr:     false,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "testdata",
		},
		{
			name:        "negative - file not exists",
			want:        nil,
			wantErr:     true,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "somewhere over the rainbow",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.setEnv, tt.setEnvValue)
			got, err := loadConfigs()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadConfigs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Equal(t, tt.want, got)

			t.Cleanup(func() { os.Unsetenv(tt.setEnv) })
			t.Cleanup(func() { os.RemoveAll("testdata/contexts_t.yaml") })
		})
	}
}

func Test_getCurrentContext(t *testing.T) {
	tests := []struct {
		name        string
		want        string
		wantErr     bool
		setEnv      string
		setEnvValue string
	}{
		{
			name:        "positive",
			want:        "aws:prod:accountId:us-east-1:cluster1",
			wantErr:     false,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "testdata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.setEnv, tt.setEnvValue)
			got, err := getCurrentContext()
			if (err != nil) != tt.wantErr {
				t.Errorf("getCurrentContext() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getCurrentContext() = %v, want %v", got, tt.want)
			}
			t.Cleanup(func() { os.Unsetenv(tt.setEnv) })
		})
	}
}

func Test_switchBack(t *testing.T) {
	tests := []struct {
		name        string
		want        string
		wantErr     bool
		setEnv      string
		setEnvValue string
	}{
		{
			name:        "positive",
			want:        "aws:prod:accountId:us-east-1:cluster1",
			wantErr:     false,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "testdata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.setEnv, tt.setEnvValue)
			got, err := switchBack()
			if (err != nil) != tt.wantErr {
				t.Errorf("switchBack() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("switchBack() = %v, want %v", got, tt.want)
			}
			t.Cleanup(func() { os.Unsetenv(tt.setEnv) })
		})
	}
}

func Test_directlyUse(t *testing.T) {
	type args struct {
		context string
	}
	tests := []struct {
		name        string
		args        args
		want        string
		wantErr     bool
		setEnv      string
		setEnvValue string
	}{
		{
			name:        "positive",
			args:        args{context: "aws:prod:accountId:us-east-1:cluster1"},
			want:        "aws:prod:accountId:us-east-1:cluster1",
			wantErr:     false,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "testdata",
		},
		{
			name:        "negative - context no found",
			args:        args{context: "does not exists"},
			want:        "",
			wantErr:     true,
			setEnv:      "KTX_CONFIG_DIR",
			setEnvValue: "testdata",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(tt.setEnv, tt.setEnvValue)
			got, err := directlyUse(tt.args.context)
			if (err != nil) != tt.wantErr {
				t.Errorf("directlyUse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("directlyUse() = %v, want %v", got, tt.want)
			}
			t.Cleanup(func() { os.Unsetenv(tt.setEnv) })
		})
	}
}
