package controller

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/utils/ptr"
)

//go:generate mockgen --build_flags=--mod=mod -package controller -destination ./mock_shared_index_informer_test.go k8s.io/client-go/tools/cache SharedIndexInformer

func TestController_multiple_finalizers_race(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)

	ctrl := gomock.NewController(t)
	informer := NewMockSharedIndexInformer(ctrl)
	informer.EXPECT().GetStore().Return(store).AnyTimes()
	handler := &SharedHandler{ControllerName: corev1.SchemeGroupVersion.WithResource("ConfigMap").String()}
	c := &controller{
		informer:  informer,
		workqueue: queue,
		handler:   handler,
	}

	// Register a handler that simulates multiple handlers removing
	var count int
	handler.Register(ctx, "test-counter", SharedControllerHandlerFunc(func(key string, obj runtime.Object) (runtime.Object, error) {
		count++

		// Test is done once the handler finally receives a nil object
		if obj == nil {
			cancel()
			return nil, nil
		}
		cm := obj.(*corev1.ConfigMap).DeepCopy()
		if len(cm.Finalizers) == 1 {
			t.Error("handler was called with an undesired resource state")
			cancel()
			return nil, nil
		}

		// simulate handler update removing first finalizer
		cm.Finalizers = cm.Finalizers[1:]
		if err := store.Add(cm); err != nil {
			t.Fatal(err)
		}
		queue.Add(key)

		// a last update to remove finalizers won't get an "update" but a "delete" event from Kubernetes
		// intentionally adding a delay to simulate the race condition described in https://github.com/rancher/rancher/issues/49328
		cm = cm.DeepCopy()
		cm.Finalizers = nil
		go func() {
			<-time.After(200 * time.Millisecond)
			store.Delete(cm)
			queue.Add(key)
		}()

		return cm, nil
	}))
	go c.runWorker()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Namespace:         "test-ns",
			Name:              "test-cm",
			Finalizers:        []string{"test-finalizer-1", "test-finalizer-2"},
			DeletionTimestamp: ptr.To(metav1.Now()),
			UID:               uuid.NewUUID(),
		},
	}
	key := "test-ns/test-cm"

	if err := store.Add(cm); err != nil {
		t.Fatal(err)
	}
	queue.Add(key)

	<-ctx.Done()
	if got, want := count, 2; got != want {
		t.Errorf("unexpected number of handler executions, got %d, want %d", got, want)
	}
}

func TestController_no_registered_handlers(t *testing.T) {
	t.Parallel()

	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	defer queue.ShutDown()
	store := cache.NewStore(cache.MetaNamespaceKeyFunc)

	ctrl := gomock.NewController(t)
	informer := NewMockSharedIndexInformer(ctrl)
	informer.EXPECT().GetStore().Return(store).AnyTimes()
	handler := &SharedHandler{ControllerName: corev1.SchemeGroupVersion.WithResource("ConfigMap").String()}
	c := &controller{
		informer:  informer,
		workqueue: queue,
		handler:   handler,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	assert.NotPanics(t, func() {
		go c.runWorker()

		cm := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Namespace:         "test-ns",
				Name:              "test-cm",
				Finalizers:        []string{"test-finalizer-1", "test-finalizer-2"},
				DeletionTimestamp: ptr.To(metav1.Now()),
				UID:               uuid.NewUUID(),
			},
		}
		key := "test-ns/test-cm"

		if err := store.Add(cm); err != nil {
			t.Fatal(err)
		}
		queue.Add(key)

		// simulate object being "finalized"
		cm.Finalizers = []string{}
		if err := store.Update(cm); err != nil {
			t.Fatal(err)
		}
		queue.Add(key)

		<-ctx.Done()
	})
}
