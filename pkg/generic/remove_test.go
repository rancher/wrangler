package generic

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_objectLifecycleAdapter_sync(t *testing.T) {
	const handlerName = "test"
	type fields struct {
		opts []OnRemoveOption
	}
	type args struct {
		key string
		obj runtime.Object
	}
	tests := []struct {
		name             string
		fields           fields
		args             args
		wantErr          bool
		wantFunc         func(runtime.Object) error
		wantHandlerCount int
		wantUpdaterCount int
	}{
		// TODO: Add test cases.
		{
			name: "nil object",
			args: args{
				"objectkey", nil,
			},
			wantFunc: func(obj runtime.Object) error {
				if obj != nil {
					return fmt.Errorf("obj should be nil")
				}
				return nil
			},
		},
		{
			name: "new object",
			args: args{
				"test/service", &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{Namespace: "test", Name: "service"},
				},
			},
			wantUpdaterCount: 1, // update to add finalizer
		},
		{
			name: "existing object with finalizer",
			args: args{
				"test/service", &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test", Name: "service",
						Finalizers: []string{"finalizer", finalizerKey + handlerName},
					},
				},
			},
			wantUpdaterCount: 0, // finalizer already set, no need to update
		},
		{
			name: "deleted object with finalizer",
			args: args{
				key: "test/service", obj: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test", Name: "service",
						Finalizers:        []string{"finalizer", finalizerKey + handlerName},
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
				},
			},
			wantHandlerCount: 1,
			wantUpdaterCount: 1, // update removing finalizer
		},
		{
			name: "deleted object without finalizer",
			args: args{
				key: "test/service", obj: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test", Name: "service",
						DeletionTimestamp: &metav1.Time{Time: time.Now()},
					},
				},
			},
			wantHandlerCount: 0,
			wantUpdaterCount: 0,
		},
		{
			name: "using condition to ignore resource",
			fields: fields{
				opts: []OnRemoveOption{
					WithCondition(func(obj runtime.Object) bool {
						metadata, err := meta.Accessor(obj)
						if err != nil {
							return false
						}
						if metadata.GetNamespace() == "test" && metadata.GetName() == "service" {
							return false
						}
						return true
					})},
			},
			args: args{
				key: "test/service", obj: &corev1.Service{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "test", Name: "service",
					},
				},
			},
			wantHandlerCount: 0,
			wantUpdaterCount: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCount, updaterCount int
			updater := func(obj runtime.Object) (runtime.Object, error) {
				updaterCount++
				return obj, nil
			}
			handler := func(key string, obj runtime.Object) (runtime.Object, error) {
				handlerCount++
				return obj, nil
			}

			sync := NewRemoveHandler("test", updater, handler, tt.fields.opts...)
			got, err := sync(tt.args.key, tt.args.obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("sync() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantFunc != nil {
				if err := tt.wantFunc(got); err != nil {
					t.Errorf("sync() unexpected value = %v, err: %v", got, err)
				}
			}
			if got, want := handlerCount, tt.wantHandlerCount; got != want {
				t.Errorf("handlerCount = %v, want %v", got, want)
			}
			if got, want := updaterCount, tt.wantUpdaterCount; got != want {
				t.Errorf("updaterCount = %v, want %v", got, want)
			}
		})
	}
}
