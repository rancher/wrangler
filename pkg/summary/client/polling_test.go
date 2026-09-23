package client

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rancher/wrangler/v3/pkg/summary"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

// fakeSummaryLister returns a predefined summary list on each call to List.
type fakeSummaryLister struct {
	mu    sync.Mutex
	lists [][]summary.SummarizedObject
	calls int
}

func (f *fakeSummaryLister) List(_ context.Context, _ metav1.ListOptions) (*summary.SummarizedObjectList, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	idx := f.calls
	if idx >= len(f.lists) {
		idx = len(f.lists) - 1
	}
	f.calls++
	l := &summary.SummarizedObjectList{}
	l.Items = append(l.Items, f.lists[idx]...)
	return l, nil
}

func summarized(ns, name, rv string) summary.SummarizedObject {
	so := summary.SummarizedObject{}
	so.Namespace = ns
	so.Name = name
	so.ResourceVersion = rv
	return so
}

// TestNewPollWatcherEmitsSummaryEvents verifies the summary adapter feeds a
// *summary.SummarizedObjectList through the generic pollwatch primitive, which
// extracts and diffs its items (i.e. meta.ExtractList understands the list type).
func TestNewPollWatcherEmitsSummaryEvents(t *testing.T) {
	lister := &fakeSummaryLister{lists: [][]summary.SummarizedObject{
		{summarized("junk", "empty2", "1")},
		{summarized("junk", "empty2", "2")},
	}}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w := newPollWatcher(ctx, lister, 10*time.Millisecond)
	defer w.Stop()

	added := waitEvent(t, w.ResultChan())
	assert.Equal(t, watch.Added, added.Type)
	require.IsType(t, &summary.SummarizedObject{}, added.Object)
	assert.Equal(t, "1", added.Object.(*summary.SummarizedObject).GetResourceVersion())

	modified := waitEvent(t, w.ResultChan())
	assert.Equal(t, watch.Modified, modified.Type)
	assert.Equal(t, "2", modified.Object.(*summary.SummarizedObject).GetResourceVersion())
}

// NewPollingClient must satisfy ExtendedInterface and report watch-list unsupported.
func TestPollingClientImplementsExtendedInterface(t *testing.T) {
	var _ ExtendedInterface = NewPollingClient(nil, 0)

	pc := NewPollingClient(nil, 0).(*pollingClient)
	assert.Equal(t, DefaultPollInterval, pc.interval)
	assert.True(t, pc.IsWatchListSemanticsUnSupported())
}

func waitEvent(t *testing.T, ch <-chan watch.Event) watch.Event {
	t.Helper()
	select {
	case e := <-ch:
		return e
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for watch event")
		return watch.Event{}
	}
}
