package k8sctx

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKubeConfig_SyncContexts(t *testing.T) {

	t.Parallel()
	testKubeConfig, err := GetKubeConfig("testdata/kube.config")
	if err != nil {
		t.Fatalf("failed to read test kube config: %s\n", err.Error())
	}

	type args struct {
		k *KubeConf
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "positive",
			args: args{
				k: &KubeConf{
					ContextFile: "testdata/contexts_sync.yaml",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := testKubeConfig
			if err := k.SyncContexts(tt.args.k); (err != nil) != tt.wantErr {
				t.Errorf("KubeConfig.SyncContexts() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestKubeConfig_Read(t *testing.T) {
	t.Parallel()
	type fields struct {
		Path string
	}
	tests := []struct {
		name       string
		fields     fields
		wantErr    bool
		wantErrMsg error
	}{
		{
			name: "negative - ErrDuplContext",
			fields: fields{
				Path: "testdata/kube.config.dupl",
			},
			wantErr:    true,
			wantErrMsg: ErrDuplContext,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := &KubeConfig{
				Path: tt.fields.Path,
			}
			err := k.Read()
			if (err != nil) != tt.wantErr {
				t.Errorf("KubeConfig.Read() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				assert.ErrorIs(t, err, tt.wantErrMsg)
			}
		})
	}
}

func TestKubeConfig_SetContextTo(t *testing.T) {
	t.Parallel()
	type fields struct {
		Path           string
		APIVersion     string
		Kind           string
		Preferences    interface{}
		Contexts       []KubeContext
		CurrentContext string
	}
	type args struct {
		contextName string
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
				Path: "testdata/kube.config.set",
			},
			args: args{
				contextName: "aws:prod:accountId:us-east-1:cluster1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			k := &KubeConfig{
				Path:           tt.fields.Path,
				APIVersion:     tt.fields.APIVersion,
				Kind:           tt.fields.Kind,
				Preferences:    tt.fields.Preferences,
				Contexts:       tt.fields.Contexts,
				CurrentContext: tt.fields.CurrentContext,
			}
			if err := k.SetContextTo(tt.args.contextName); (err != nil) != tt.wantErr {
				t.Errorf("KubeConfig.SetContextTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
	t.Cleanup(func() { os.RemoveAll("testdata/kube.config.set") })
}
