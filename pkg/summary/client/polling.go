package client

import (
	"context"
	"time"

	"github.com/rancher/wrangler/v3/pkg/pollwatch"
	"github.com/rancher/wrangler/v3/pkg/summary"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

// DefaultPollInterval is the interval used to poll list-only resources when no
// explicit interval is provided.
const DefaultPollInterval = pollwatch.DefaultInterval

// pollingClient wraps an ExtendedInterface and synthesizes watch events by
// periodically listing resources that do not support the "watch" verb.
//
// Some resources (for example CRDs that only advertise get, list, create and
// delete) cannot be watched. A normal informer requires LIST and WATCH, so the
// data for these resources never populates. pollingClient bridges that gap by
// turning successive LIST results into synthetic Added/Modified/Deleted events,
// allowing the existing informer machinery to stay up to date without a server
// side watch.
type pollingClient struct {
	ExtendedInterface
	interval time.Duration
}

// NewPollingClient returns an ExtendedInterface that synthesizes watch events by
// polling List at the given interval. A non-positive interval falls back to
// DefaultPollInterval. It is intended to wrap the client used to build informers
// for resources that support "list" but not "watch".
func NewPollingClient(base ExtendedInterface, interval time.Duration) ExtendedInterface {
	if interval <= 0 {
		interval = DefaultPollInterval
	}
	return &pollingClient{ExtendedInterface: base, interval: interval}
}

// IsWatchListSemanticsUnSupported reports that watch-list streaming is not
// supported. Polling relies on repeated LIST calls, so the reflector must fall
// back to LIST+WATCH semantics.
func (p *pollingClient) IsWatchListSemanticsUnSupported() bool {
	return true
}

func (p *pollingClient) Resource(resource schema.GroupVersionResource) NamespaceableResourceInterface {
	return p.ResourceWithOptions(resource, nil)
}

func (p *pollingClient) ResourceWithOptions(resource schema.GroupVersionResource, opts *Options) NamespaceableResourceInterface {
	return &pollingResourceClient{
		NamespaceableResourceInterface: p.ExtendedInterface.ResourceWithOptions(resource, opts),
		interval:                       p.interval,
	}
}

// pollingResourceClient overrides Watch to poll while delegating List and
// namespacing to the wrapped client.
type pollingResourceClient struct {
	NamespaceableResourceInterface
	interval time.Duration
}

func (c *pollingResourceClient) Namespace(ns string) ResourceInterface {
	return &pollingNamespacedClient{
		ResourceInterface: c.NamespaceableResourceInterface.Namespace(ns),
		interval:          c.interval,
	}
}

func (c *pollingResourceClient) Watch(ctx context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return newPollWatcher(ctx, c.NamespaceableResourceInterface, c.interval), nil
}

// pollingNamespacedClient overrides Watch for a namespaced resource client.
type pollingNamespacedClient struct {
	ResourceInterface
	interval time.Duration
}

func (c *pollingNamespacedClient) Watch(ctx context.Context, _ metav1.ListOptions) (watch.Interface, error) {
	return newPollWatcher(ctx, c.ResourceInterface, c.interval), nil
}

// summaryLister is the subset of the summary client used to build a poll watcher.
// Both NamespaceableResourceInterface and ResourceInterface satisfy it.
type summaryLister interface {
	List(ctx context.Context, opts metav1.ListOptions) (*summary.SummarizedObjectList, error)
}

// newPollWatcher adapts a summary lister to the generic pollwatch primitive. The
// returned *summary.SummarizedObjectList is a runtime.Object, so pollwatch can
// extract and diff its items generically via meta.ExtractList.
func newPollWatcher(ctx context.Context, lister summaryLister, interval time.Duration) watch.Interface {
	return pollwatch.New(ctx, func(ctx context.Context, opts metav1.ListOptions) (runtime.Object, error) {
		return lister.List(ctx, opts)
	}, interval)
}
