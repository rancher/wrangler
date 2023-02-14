package apply

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func Test_multiNamespaceList(t *testing.T) {
	results := map[string]*unstructured.UnstructuredList{
		"ns1": {Items: []unstructured.Unstructured{
			{Object: map[string]interface{}{"name": "o1", "namespace": "ns1"}},
			{Object: map[string]interface{}{"name": "o2", "namespace": "ns1"}},
			{Object: map[string]interface{}{"name": "o3", "namespace": "ns1"}},
		}},
		"ns2": {Items: []unstructured.Unstructured{
			{Object: map[string]interface{}{"name": "o4", "namespace": "ns2"}},
			{Object: map[string]interface{}{"name": "o5", "namespace": "ns2"}},
		}},
		"ns3": {Items: []unstructured.Unstructured{}},
	}
	s := runtime.NewScheme()
	err := appsv1.SchemeBuilder.AddToScheme(s)
	assert.NoError(t, err, "Failed to build schema.")
	baseClient := fake.NewSimpleDynamicClient(s)
	baseClient.PrependReactor("list", "*", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		if strings.Contains(action.GetNamespace(), "error") {
			return true, nil, errors.New("simulated failure")
		}

		return true, results[action.GetNamespace()], nil
	})

	type args struct {
		namespaces []string
	}
	tests := []struct {
		name          string
		args          args
		expectedCalls int
		expectError   bool
	}{
		{
			name: "no namespaces",
			args: args{
				namespaces: []string{},
			},
			expectError:   false,
			expectedCalls: 0,
		},
		{
			name: "1 namespace",
			args: args{
				namespaces: []string{"ns1"},
			},
			expectError:   false,
			expectedCalls: 3,
		},
		{
			name: "many namespaces",
			args: args{
				namespaces: []string{"ns1", "ns2", "ns3"},
			},
			expectError:   false,
			expectedCalls: 5,
		},
		{
			name: "1 namespace error",
			args: args{
				namespaces: []string{"error", "ns2", "ns3"},
			},
			expectError:   true,
			expectedCalls: -1,
		},
		{
			name: "many namespace errors",
			args: args{
				namespaces: []string{"error", "error1", "error2"},
			},
			expectError:   true,
			expectedCalls: -1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var calls int
			err := multiNamespaceList(context.TODO(), tt.args.namespaces, baseClient.Resource(appsv1.SchemeGroupVersion.WithResource("deployments")), labels.NewSelector(), func(obj unstructured.Unstructured) {
				calls += 1
			})

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			if tt.expectedCalls >= 0 {
				assert.Equal(t, tt.expectedCalls, calls)
			}
		})
	}
}
