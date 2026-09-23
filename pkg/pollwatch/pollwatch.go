// Package pollwatch provides a watch.Interface that synthesizes watch events by
// periodically listing a resource. It lets resources that support "list" but not
// "watch" be consumed by informers and other watch clients (for example CRDs that
// only expose get/list/create/delete verbs).
//
// It is client-agnostic: any List function returning a list runtime.Object works,
// whether that is an unstructured list, a typed client-go list, or a summarized
// list. Items are extracted with meta.ExtractList and identified via their
// object metadata, so no per-type code is required.
package pollwatch

import (
	"context"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
)

// DefaultInterval is the poll interval used when a non-positive interval is given.
const DefaultInterval = 30 * time.Second

// ListFunc lists a resource and returns a list runtime.Object whose items are
// extracted via meta.ExtractList.
type ListFunc func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error)

// New returns a watch.Interface that synthesizes Added/Modified/Deleted events by
// periodically calling list and diffing successive results.
//
// Objects are keyed by namespace/name; modifications are detected via
// resourceVersion. The watch runs until ctx is cancelled, Stop is called, or list
// returns an error (which ends the watch so a reflector will relist and retry). A
// non-positive interval falls back to DefaultInterval.
func New(ctx context.Context, list ListFunc, interval time.Duration) watch.Interface {
	if interval <= 0 {
		interval = DefaultInterval
	}
	ctx, cancel := context.WithCancel(ctx)
	w := &pollWatcher{
		result: make(chan watch.Event),
		cancel: cancel,
	}
	go w.run(ctx, list, interval)
	return w
}

type pollWatcher struct {
	result chan watch.Event
	cancel context.CancelFunc
	stop   sync.Once
}

func (w *pollWatcher) ResultChan() <-chan watch.Event { return w.result }

func (w *pollWatcher) Stop() { w.stop.Do(w.cancel) }

func (w *pollWatcher) run(ctx context.Context, list ListFunc, interval time.Duration) {
	defer close(w.result)

	// known maps a namespace/name key to the last observed object, used to detect
	// additions, modifications (by resourceVersion) and deletions.
	known := map[string]runtime.Object{}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for first := true; ; first = false {
		if !first {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}

		listObj, err := list(ctx, metav1.ListOptions{})
		if err != nil {
			return
		}
		items, err := meta.ExtractList(listObj)
		if err != nil {
			return
		}

		seen := make(map[string]struct{}, len(items))
		for _, obj := range items {
			acc, err := meta.Accessor(obj)
			if err != nil {
				continue
			}
			key := acc.GetNamespace() + "/" + acc.GetName()
			seen[key] = struct{}{}

			switch prev, ok := known[key]; {
			case !ok:
				if !w.send(ctx, watch.Added, obj) {
					return
				}
			case resourceVersion(prev) != acc.GetResourceVersion():
				if !w.send(ctx, watch.Modified, obj) {
					return
				}
			}
			known[key] = obj
		}

		for key, obj := range known {
			if _, ok := seen[key]; ok {
				continue
			}
			if !w.send(ctx, watch.Deleted, obj) {
				return
			}
			delete(known, key)
		}
	}
}

func (w *pollWatcher) send(ctx context.Context, t watch.EventType, obj runtime.Object) bool {
	select {
	case w.result <- watch.Event{Type: t, Object: obj}:
		return true
	case <-ctx.Done():
		return false
	}
}

func resourceVersion(obj runtime.Object) string {
	acc, err := meta.Accessor(obj)
	if err != nil {
		return ""
	}
	return acc.GetResourceVersion()
}
