package informer

import (
	"context"
	"testing"
	"time"

	"github.com/rancher/wrangler/v3/pkg/summary"
	"github.com/rancher/wrangler/v3/pkg/summary/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

var _ client.ExtendedInterface = (*mockClient)(nil)

type mockClient struct {
	listCalled  chan struct{}
	watchCalled chan struct{}
}

func (m *mockClient) Resource(resource schema.GroupVersionResource) client.NamespaceableResourceInterface {
	return m
}

func (m *mockClient) ResourceWithOptions(resource schema.GroupVersionResource, opts *client.Options) client.NamespaceableResourceInterface {
	return m
}

func (m *mockClient) Namespace(string) client.ResourceInterface {
	return m
}

func (m *mockClient) List(ctx context.Context, opts metav1.ListOptions) (*summary.SummarizedObjectList, error) {
	if m.listCalled != nil {
		m.listCalled <- struct{}{}
	}
	return &summary.SummarizedObjectList{
		ListMeta: metav1.ListMeta{
			ResourceVersion: "1",
		},
	}, nil
}

func (m *mockClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	if m.watchCalled != nil {
		m.watchCalled <- struct{}{}
	}
	return watch.NewFake(), nil
}

type mockClientUnsupported struct {
	mockClient
}

func (m *mockClientUnsupported) IsWatchListSemanticsUnSupported() bool {
	return true
}

type mockClientSupported struct {
	mockClient
}

func (m *mockClientSupported) IsWatchListSemanticsUnSupported() bool {
	return false
}

func TestNewFilteredSummaryInformer_WatchListSupport(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "tests"}
	namespace := "default"

	tests := []struct {
		name        string
		client      client.Interface
		expectList  bool
		expectWatch bool
	}{
		{
			name: "Client supporting watchlist (default)",
			client: &mockClient{
				listCalled:  make(chan struct{}, 1),
				watchCalled: make(chan struct{}, 1),
			},
			expectList: false,
		},
		{
			name: "Client explicitly supporting watchlist",
			client: &mockClientSupported{
				mockClient: mockClient{
					listCalled:  make(chan struct{}, 1),
					watchCalled: make(chan struct{}, 1),
				},
			},
			expectList: false,
		},
		{
			name: "Client explicitly NOT supporting watchlist",
			client: &mockClientUnsupported{
				mockClient: mockClient{
					listCalled:  make(chan struct{}, 1),
					watchCalled: make(chan struct{}, 1),
				},
			},
			expectList: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			informer := NewFilteredSummaryInformer(tt.client, gvr, namespace, 0, cache.Indexers{}, nil)
			stopCh := make(chan struct{})
			defer close(stopCh)

			go informer.Informer().Run(stopCh)

			// Wait for either List or Watch to be called
			var mc *mockClient
			switch c := tt.client.(type) {
			case *mockClient:
				mc = c
			case *mockClientSupported:
				mc = &c.mockClient
			case *mockClientUnsupported:
				mc = &c.mockClient
			}

			time.Sleep(100 * time.Millisecond)
			listCalled := false
			watchCalled := false

			select {
			case <-time.After(100 * time.Millisecond):
			case <-mc.listCalled:
				listCalled = true
			}

			select {
			case <-time.After(100 * time.Millisecond):
			case <-mc.watchCalled:
				watchCalled = true
			}

			if tt.expectList && !listCalled {
				t.Fatal("Expected list call but didn't get it")
			}

			if !tt.expectList && listCalled {
				t.Fatal("Expected NO list call")
			}

			if !watchCalled {
				t.Fatal("Expected watch call but didn't get it")
			}
		})
	}
}

func TestNewFilteredSummaryInformerWithOptions_WatchListSupport(t *testing.T) {
	gvr := schema.GroupVersionResource{Group: "test", Version: "v1", Resource: "tests"}
	namespace := "default"

	tests := []struct {
		name        string
		client      client.ExtendedInterface
		expectList  bool
		expectWatch bool
	}{
		{
			name: "Client supporting watchlist (default)",
			client: &mockClient{
				listCalled:  make(chan struct{}, 1),
				watchCalled: make(chan struct{}, 1),
			},
			expectList: false,
		},
		{
			name: "Client explicitly supporting watchlist",
			client: &mockClientSupported{
				mockClient: mockClient{
					listCalled:  make(chan struct{}, 1),
					watchCalled: make(chan struct{}, 1),
				},
			},
			expectList: false,
		},
		{
			name: "Client explicitly NOT supporting watchlist",
			client: &mockClientUnsupported{
				mockClient: mockClient{
					listCalled:  make(chan struct{}, 1),
					watchCalled: make(chan struct{}, 1),
				},
			},
			expectList: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			informer := NewFilteredSummaryInformerWithOptions(tt.client, gvr, nil, namespace, 0, cache.Indexers{}, nil)
			stopCh := make(chan struct{})
			defer close(stopCh)

			go informer.Informer().Run(stopCh)

			// Wait for either List or Watch to be called
			var mc *mockClient
			switch c := tt.client.(type) {
			case *mockClient:
				mc = c
			case *mockClientSupported:
				mc = &c.mockClient
			case *mockClientUnsupported:
				mc = &c.mockClient
			}

			time.Sleep(100 * time.Millisecond)
			listCalled := false
			watchCalled := false

			select {
			case <-time.After(100 * time.Millisecond):
			case <-mc.listCalled:
				listCalled = true
			}

			select {
			case <-time.After(100 * time.Millisecond):
			case <-mc.watchCalled:
				watchCalled = true
			}

			if tt.expectList && !listCalled {
				t.Fatal("Expected list call but didn't get it")
			}

			if !tt.expectList && listCalled {
				t.Fatal("Expected NO list call")
			}

			if !watchCalled {
				t.Fatal("Expected watch call but didn't get it")
			}
		})
	}
}
