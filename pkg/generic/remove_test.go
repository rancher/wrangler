package generic

import (
	"fmt"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func Test_objectLifecycleAdapter_sync(t *testing.T) {
	const handlerName = "test"
	type args struct {
		key string
		obj runtime.Object
	}
	tests := []struct {
		name             string
		args             args
		wantErr          bool
		wantFunc         func(runtime.Object) error
		wantHandlerCount int
		wantUpdaterCount int
	}{
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

			sync := NewRemoveHandler("test", updater, handler)
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
			if handlerCount != tt.wantHandlerCount {
				t.Errorf("handlerCount = %v, want %v", handlerCount, tt.wantHandlerCount)
			}
			if updaterCount != tt.wantUpdaterCount {
				t.Errorf("updaterCount = %v, want %v", updaterCount, tt.wantUpdaterCount)
			}
		})
	}
}
