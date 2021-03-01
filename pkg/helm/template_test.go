package helm

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

var (
	sampleDeployment = map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name": "nginx-deployment",
			"labels": map[string]interface{}{
				"app": "nginx",
			},
		},
		"spec": map[string]interface{}{
			"replicas": 3,
			"selector": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": "nginix",
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": map[string]interface{}{
						"app": "nginx",
					},
				},
				"spec": map[string]interface{}{
					"containers": []map[string]interface{}{
						{
							"name":  "nginx",
							"image": "nginx:1.14.2",
							"ports": []map[string]interface{}{
								{
									"containerPort": 80,
								},
							},
						},
					},
				},
			},
		},
	}
)

func Test_injectNamespace(t *testing.T) {
	type args struct {
		manifest  map[string]interface{}
		namespace string
	}
	tests := []struct {
		name    string
		args    args
		want    map[string]interface{}
		wantErr bool
	}{
		{
			name:    "empty",
			args:    args{},
			want:    nil,
			wantErr: false,
		},
		{
			name: "empty-map",
			args: args{
				manifest:  map[string]interface{}{},
				namespace: "foo",
			},
			want: map[string]interface{}{
				"metadata": map[string]interface{}{
					"namespace": "foo",
				},
			},
			wantErr: false,
		},

		{
			name: "with-metadata-no-namespace",
			args: args{
				manifest: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "nginx-deployment",
						"labels": map[string]interface{}{
							"app": "nginx",
						},
					},
				},
				namespace: "foo",
			},
			want: map[string]interface{}{
				"metadata": map[string]interface{}{
					"name": "nginx-deployment",
					"labels": map[string]interface{}{
						"app": "nginx",
					},
					"namespace": "foo",
				},
			},
			wantErr: false,
		},

		{
			name: "with-metadata-with-namespace",
			args: args{
				manifest: map[string]interface{}{
					"metadata": map[string]interface{}{
						"name": "nginx-deployment",
						"labels": map[string]interface{}{
							"app": "nginx",
						},
						"namespace": "already has a NS",
					},
				},
				namespace: "foo",
			},
			want:    nil,
			wantErr: true,
		},

		{
			name: "with-invalid-metadata-type",
			args: args{
				manifest: map[string]interface{}{
					"metadata": []map[string]interface{}{
						{
							"name": "nginx-deployment",
							"labels": map[string]interface{}{
								"app": "nginx",
							},
						},
					},
				},
				namespace: "foo",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := injectNamespace(tt.args.manifest, tt.args.namespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("injectNamespace() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("injectNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_cleanManifest(t *testing.T) {
	type args struct {
		manifest string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "empty",
			want:    "",
			wantErr: false,
		},
		{
			name: "single-valid-entry",
			args: args{
				manifest: `
foo: bar
`,
			},
			want: strings.TrimSpace(`
foo: bar
`),
			wantErr: false,
		},
		{
			name: "multiple-some-invalid-entries",
			args: args{
				manifest: `
foo: I am valid
---
I'm not valid
---
bar: I am valid
---
Im not a map
---
baz: I am valid
---
`,
			},
			want: strings.TrimSpace(`
foo: I am valid
---
bar: I am valid
---
baz: I am valid
`),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := cleanManifest(tt.args.manifest)
			if (err != nil) != tt.wantErr {
				t.Errorf("cleanManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("cleanManifest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTemplateWithCRDs(t *testing.T) {
	type args struct {
		opts TemplateOptions
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
		{
			"empty",
			args{},
			nil,
			true,
		},
		{
			"test-chart",
			args{TemplateOptions{
				Chart:   filepath.Join("testdata", "template", "test-chart"),
				Release: "random-chart",
				Set:     []string{"testValue=foobar"},
			}},
			[]interface{}{
				map[string]interface{}{
					"apiVersion": "apiextensions.k8s.io/v1beta1",
					"kind":       "CustomResourceDefinition",
					"metadata": map[string]interface{}{
						"name": "bar.fabrikate.microsoft.com",
					},
					"spec": map[string]interface{}{
						"group":   "fabrikate.microsoft.com",
						"version": "v1alpha1",
						"names": map[string]interface{}{
							"kind":     "Bar",
							"plural":   "bars",
							"singular": "bar",
						},
						"scope": "Namespaced",
					},
				},
				map[string]interface{}{
					"apiVersion": "apiextensions.k8s.io/v1beta1",
					"kind":       "CustomResourceDefinition",
					"metadata": map[string]interface{}{
						"name": "foo.fabrikate.microsoft.com",
					},
					"spec": map[string]interface{}{
						"group":   "fabrikate.microsoft.com",
						"version": "v1alpha1",
						"names": map[string]interface{}{
							"kind":     "Foo",
							"plural":   "foos",
							"singular": "foo",
						},
						"scope": "Namespaced",
					},
				},
				map[string]interface{}{
					"apiVersion": "v1",
					"kind":       "Service",
					"metadata": map[string]interface{}{
						"name": "random-chart-test-chart",
					},
					"spec": map[string]interface{}{
						"testValue": "foobar",
					},
				},
				map[string]interface{}{
					"apiVersion": "apps/v1",
					"kind":       "Deployment",
					"metadata": map[string]interface{}{
						"name": "random-chart-test-chart",
					},
					"spec": map[string]interface{}{
						"testValue": "foobar",
					},
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		cwd, _ := os.Getwd()
		fmt.Println(cwd)
		t.Run(tt.name, func(t *testing.T) {
			got, err := TemplateWithCRDs(tt.args.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("TemplateWithCRDs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TemplateWithCRDs() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
