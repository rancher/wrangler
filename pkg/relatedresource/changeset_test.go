package relatedresource

import (
	"context"
	"fmt"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

func Test_addResourceEventHandler(t *testing.T) {
	const expectedCalls = 5

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var counter int
	handler := &cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) { counter++ },
	}

	informer := &fakeInformer{}
	addResourceEventHandler(ctx, informer, handler)

	for i := 0; i < expectedCalls; i++ {
		informer.add(nil)
	}

	if want, got := expectedCalls, counter; want != got {
		t.Errorf("unexpected number of executions of handler func, want: %d, got: %d", want, got)
	}

	// Close enqueuer context and wait for unregistering goroutine
	cancel()
	informer.waitUntilDeleted(handler, 1*time.Second)

	// New informer calls should not trigger our handler
	informer.add(nil)
	if got, want := counter, expectedCalls; got != want {
		t.Errorf("resource event handler is not correctly removed")
	}
}

type handlerRegistration struct{}

func (h handlerRegistration) HasSynced() bool { return true }

// fakeInformer implements a subset of cache.SharedIndexInformer, only those methods used by addResourceEventHandler
type fakeInformer struct {
	handlers   []cache.ResourceEventHandler
	reg        []cache.ResourceEventHandlerRegistration
	deleteChan []chan struct{}
}

func (informer *fakeInformer) AddEventHandler(handler cache.ResourceEventHandler) (cache.ResourceEventHandlerRegistration, error) {
	handlerReg := handlerRegistration{}
	informer.handlers = append(informer.handlers, handler)
	informer.reg = append(informer.reg, handlerReg)
	informer.deleteChan = append(informer.deleteChan, make(chan struct{}))
	return handlerReg, nil
}

func (informer *fakeInformer) RemoveEventHandler(handlerReg cache.ResourceEventHandlerRegistration) error {
	x := slicesIndex(informer.reg, handlerReg)
	if x < 0 {
		return fmt.Errorf("handler not found")
	}

	close(informer.deleteChan[x])
	informer.reg = deleteIndex(informer.reg, x)
	informer.handlers = deleteIndex(informer.handlers, x)
	informer.deleteChan = deleteIndex(informer.deleteChan, x)
	return nil
}

func (informer *fakeInformer) add(obj runtime.Object) {
	for _, handler := range informer.handlers {
		handler.OnAdd(obj, false)
	}
}

func (informer *fakeInformer) waitUntilDeleted(handler cache.ResourceEventHandler, timeout time.Duration) {
	x := slicesIndex(informer.handlers, handler)
	if x < 0 {
		return
	}

	select {
	case <-informer.deleteChan[x]:
	case <-time.After(timeout):
	}
}

func slicesIndex[S ~[]E, E comparable](s S, v E) int {
	for i := range s {
		if v == s[i] {
			return i
		}
	}
	return -1
}

func deleteIndex[S ~[]E, E any](s S, i int) S {
	return append(s[:i], s[i+1:]...)
}
