package yaml

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
)

func TestUnmarshalWithJSONDecoder_deployment(t *testing.T) {

	tests := []struct {
		name    string
		input   []byte
		want    func() []*appsv1.Deployment
		wantErr bool
	}{
		{
			name:  "single deployment",
			input: singleDeployment,
			want: func() []*appsv1.Deployment {
				dep := &appsv1.Deployment{}
				dep.Name = "singleDeployment"
				dep.APIVersion = appsv1.SchemeGroupVersion.String()
				dep.Kind = "Deployment"
				three := int32(3)
				dep.Spec.Replicas = &three
				return []*appsv1.Deployment{dep}
			},
		},
		{
			name:  "multiple deployment",
			input: multipleDeployments,
			want: func() []*appsv1.Deployment {
				dep := &appsv1.Deployment{}
				dep.Name = "dep1"
				dep.APIVersion = appsv1.SchemeGroupVersion.String()
				dep.Kind = "Deployment"
				three := int32(3)
				dep.Spec.Replicas = &three

				dep2 := &appsv1.Deployment{}
				dep2.APIVersion = appsv1.SchemeGroupVersion.String()
				dep2.Kind = "Deployment"
				dep2.Name = "dep2"
				dep2.Namespace = "dep2-ns"
				dep2.Spec.Paused = true

				dep3 := &appsv1.Deployment{}
				dep3.Name = "dep3"
				dep3.Namespace = "dep3-ns"
				dep3.Status.ReadyReplicas = 4
				dep3.Labels = map[string]string{"app": "testapp"}
				return []*appsv1.Deployment{dep, dep2, dep3}
			},
		},
		{
			name:  "empty document",
			input: emptyDoc,
			want: func() []*appsv1.Deployment {
				return nil
			},
		},
		{
			name:  "unknown document",
			input: unknownDoc,
			want: func() []*appsv1.Deployment {
				dep := &appsv1.Deployment{}
				return []*appsv1.Deployment{dep}
			},
		},
		{
			name:    "invalid YAML",
			input:   invalidYAML,
			wantErr: true,
		},
		{
			name:    "invalid JSON marshal",
			input:   singleString,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := UnmarshalWithJSONDecoder[*appsv1.Deployment](bytes.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected an error but got nil")
				return
			}
			require.NoError(t, err, "UnmarshalWithJSONDecoder received an unexpected error")
			var want []*appsv1.Deployment
			if tt.want != nil {
				want = tt.want()
			}
			require.Equal(t, got, want, "UnmarshalWithJSONDecoder received unexpected results")
		})
	}
}

func TestUnmarshalWithJSONDecoder_jsonSruct(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   []byte
		want    func() []*JSONstruct
		wantErr bool
	}{
		{
			name:  "all expected fields",
			input: jsonYAML,
			want: func() []*JSONstruct {
				obj := &JSONstruct{}
				obj.EmbeddedField = "embeddedValue"
				obj.NestedField.EmbeddedField = "nestedValue"
				obj.CustomField.Name = "testName"
				obj.CustomField.Namespace = "testNamespace"
				obj.NormalField = 28
				obj.MismatchField = true
				return []*JSONstruct{obj}
			},
		},
		{
			name:  "unknown type",
			input: singleDeployment,
			want: func() []*JSONstruct {
				obj := &JSONstruct{}
				return []*JSONstruct{obj}
			},
		},
		{
			name:  "empty document",
			input: emptyDoc,
		},
		{
			name:    "invalid YAML",
			input:   invalidYAML,
			wantErr: true,
		},
		{
			name:    "invalid JSON marshal",
			input:   singleString,
			wantErr: true,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := UnmarshalWithJSONDecoder[*JSONstruct](bytes.NewReader(tt.input))
			if tt.wantErr {
				require.Error(t, err, "expected an error but got nil")
				return
			}
			require.NoError(t, err, "UnmarshalWithJSONDecoder received an unexpected error")
			var want []*JSONstruct
			if tt.want != nil {
				want = tt.want()
			}
			require.Equal(t, got, want, "UnmarshalWithJSONDecoder received unexpected results")
		})
	}
}
