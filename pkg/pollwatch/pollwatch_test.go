package pollwatch

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// fakeLister returns a predefined list runtime.Object on each successive call.
type fakeLister struct {
	mu    sync.Mutex
	lists []runtime.Object
	calls int
	err   error
}

func (f *fakeLister) List(_ context.Context, _ metav1.ListOptions) (runtime.Object, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	idx := f.calls
	if idx >= len(f.lists) {
		idx = len(f.lists) - 1
	}
	f.calls++
	return f.lists[idx], nil
}

func item(ns, name, rv string) unstructured.Unstructured {
	u := unstructured.Unstructured{}
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetResourceVersion(rv)
	return u
}

func list(items ...unstructured.Unstructured) *unstructured.UnstructuredList {
	l := &unstructured.UnstructuredList{}
	l.Items = append(l.Items, items...)
	return l
}

func recv(t *testing.T, ch <-chan watch.Event) watch.Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
		return watch.Event{}
	}
}

func rv(obj runtime.Object) string {
	return obj.(*unstructured.Unstructured).GetResourceVersion()
}

func TestNewEmitsAddModifyDelete(t *testing.T) {
	lw := &fakeLister{lists: []runtime.Object{
		list(item("junk", "empty2", "1")), // Added
		list(item("junk", "empty2", "2")), // Modified (rv changed)
		list(),                            // Deleted
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(ctx, lw.List, 10*time.Millisecond)
	defer w.Stop()

	added := recv(t, w.ResultChan())
	assert.Equal(t, watch.Added, added.Type)
	assert.Equal(t, "1", rv(added.Object))

	modified := recv(t, w.ResultChan())
	assert.Equal(t, watch.Modified, modified.Type)
	assert.Equal(t, "2", rv(modified.Object))

	deleted := recv(t, w.ResultChan())
	assert.Equal(t, watch.Deleted, deleted.Type)
}

func TestNewNoDuplicateEventsWhenUnchanged(t *testing.T) {
	lw := &fakeLister{lists: []runtime.Object{list(item("ns", "a", "1"))}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(ctx, lw.List, 5*time.Millisecond)
	defer w.Stop()

	first := recv(t, w.ResultChan())
	assert.Equal(t, watch.Added, first.Type)

	select {
	case e := <-w.ResultChan():
		t.Fatalf("unexpected event for unchanged object: %v", e)
	case <-time.After(50 * time.Millisecond):
	}
}

func TestNewStopClosesChannel(t *testing.T) {
	lw := &fakeLister{lists: []runtime.Object{list()}}

	w := New(context.Background(), lw.List, 5*time.Millisecond)
	w.Stop()

	select {
	case _, ok := <-w.ResultChan():
		assert.False(t, ok, "channel should be closed after Stop")
	case <-time.After(2 * time.Second):
		t.Fatal("result channel not closed after Stop")
	}
}

func TestNewEndsOnListError(t *testing.T) {
	lw := &fakeLister{err: assert.AnError}

	w := New(context.Background(), lw.List, 5*time.Millisecond)
	defer w.Stop()

	select {
	case _, ok := <-w.ResultChan():
		assert.False(t, ok, "channel should close when list errors")
	case <-time.After(2 * time.Second):
		t.Fatal("watch did not end after list error")
	}
}

func TestNewDefaultsInterval(t *testing.T) {
	// A non-positive interval must not busy-loop; the first poll still happens
	// immediately, then subsequent polls use DefaultInterval.
	lw := &fakeLister{lists: []runtime.Object{list(item("ns", "a", "1"))}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := New(ctx, lw.List, 0)
	defer w.Stop()

	added := recv(t, w.ResultChan())
	assert.Equal(t, watch.Added, added.Type)
}
