package objectset

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestObjectSet_Namespaces(t *testing.T) {
	type fields struct {
		errs        []error
		objects     ObjectByGVK
		objectsByGK ObjectByGK
		order       []runtime.Object
		gvkOrder    []schema.GroupVersionKind
		gvkSeen     map[schema.GroupVersionKind]bool
	}
	tests := []struct {
		name           string
		fields         fields
		wantNamespaces []string
	}{
		{
			name: "empty",
			fields: fields{
				objects: map[schema.GroupVersionKind]map[ObjectKey]runtime.Object{},
			},
			wantNamespaces: nil,
		},
		{
			name: "1 namespace",
			fields: fields{
				objects: map[schema.GroupVersionKind]map[ObjectKey]runtime.Object{
					schema.GroupVersionKind{}: {
						ObjectKey{Namespace: "ns1", Name: "a"}: nil,
						ObjectKey{Namespace: "ns1", Name: "b"}: nil,
					},
				},
			},
			wantNamespaces: []string{"ns1"},
		},
		{
			name: "many namespace",
			fields: fields{
				objects: map[schema.GroupVersionKind]map[ObjectKey]runtime.Object{
					schema.GroupVersionKind{}: {
						ObjectKey{Namespace: "ns1", Name: "a"}: nil,
						ObjectKey{Namespace: "ns2", Name: "b"}: nil,
					},
				},
			},
			wantNamespaces: []string{"ns1", "ns2"},
		},
		{
			name: "missing namespace",
			fields: fields{
				objects: map[schema.GroupVersionKind]map[ObjectKey]runtime.Object{
					schema.GroupVersionKind{}: {
						ObjectKey{Namespace: "ns1", Name: "a"}: nil,
						ObjectKey{Name: "b"}:                   nil,
					},
				},
			},
			wantNamespaces: []string{"", "ns1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &ObjectSet{
				errs:        tt.fields.errs,
				objects:     tt.fields.objects,
				objectsByGK: tt.fields.objectsByGK,
				order:       tt.fields.order,
				gvkOrder:    tt.fields.gvkOrder,
				gvkSeen:     tt.fields.gvkSeen,
			}

			gotNamespaces := o.Namespaces()
			assert.ElementsMatchf(t, tt.wantNamespaces, gotNamespaces, "Namespaces() = %v, want %v", gotNamespaces, tt.wantNamespaces)
		})
	}
}
