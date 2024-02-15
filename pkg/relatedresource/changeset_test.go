package relatedresource

import (
	"context"
	"fmt"
	"slices"
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
	x := slices.Index(informer.reg, handlerReg)
	if x < 0 {
		return fmt.Errorf("handler not found")
	}

	close(informer.deleteChan[x])
	informer.reg = slices.Delete(informer.reg, x, x+1)
	informer.handlers = slices.Delete(informer.handlers, x, x+1)
	informer.deleteChan = slices.Delete(informer.deleteChan, x, x+1)
	return nil
}

func (informer *fakeInformer) add(obj runtime.Object) {
	for _, handler := range informer.handlers {
		handler.OnAdd(obj, false)
	}
}

func (informer *fakeInformer) waitUntilDeleted(handler cache.ResourceEventHandler, timeout time.Duration) {
	x := slices.Index(informer.handlers, handler)
	if x < 0 {
		return
	}

	select {
	case <-informer.deleteChan[x]:
	case <-time.After(timeout):
	}
}
