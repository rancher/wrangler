package generic

import (
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
)

func TestCache(t *testing.T) {
	indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	indexer.Add(&v1.Pod{ObjectMeta: metav1.ObjectMeta{Namespace: metav1.NamespaceDefault, Name: "test-01"}})

	// test cache with correct type for indexer contents
	podCache := NewCache[*v1.Pod](indexer, v1.SchemeGroupVersion.WithResource("pods").GroupResource())
	if _, err := podCache.Get(metav1.NamespaceDefault, "test-01"); err != nil {
		t.Fatalf("failed to get pod: %v", err)
	}
	if _, err := podCache.Get(metav1.NamespaceSystem, "test-01"); err == nil {
		t.Fatalf("unexpected success getting nonexistent pod")
	}
	if _, err := podCache.Get(metav1.NamespaceDefault, "test-02"); err == nil {
		t.Fatalf("unexpected success getting nonexistent pod")
	}

	// test cache with wrong type for indexer contents
	secretCache := NewCache[*v1.Secret](indexer, v1.SchemeGroupVersion.WithResource("secrets").GroupResource())
	if _, err := secretCache.Get(metav1.NamespaceDefault, "test-01"); err == nil {
		t.Fatalf("unexpected success getting secret from pod indexer")
	}
	if _, err := secretCache.Get(metav1.NamespaceSystem, "test-01"); err == nil {
		t.Fatalf("unexpected success getting secret from pod indexer")
	}
	if _, err := secretCache.Get(metav1.NamespaceDefault, "test-02"); err == nil {
		t.Fatalf("unexpected success getting secret from pod indexer")
	}
}

func TestNonNamespacedCache(t *testing.T) {
	indexer := cache.NewIndexer(cache.DeletionHandlingMetaNamespaceKeyFunc, nil)
	indexer.Add(&v1.Node{ObjectMeta: metav1.ObjectMeta{Name: "test-01"}})

	// test cache with correct type for indexer contents
	nodeCache := NewNonNamespacedCache[*v1.Node](indexer, v1.SchemeGroupVersion.WithResource("nodes").GroupResource())
	if _, err := nodeCache.Get("test-01"); err != nil {
		t.Fatalf("failed to get node: %v", err)
	}
	if _, err := nodeCache.Get("test-02"); err == nil {
		t.Fatalf("unexpected success getting nonexistent node")
	}

	// test cache with wrong type for indexer contents
	pvCache := NewNonNamespacedCache[*v1.PersistentVolume](indexer, v1.SchemeGroupVersion.WithResource("persistentvolumes").GroupResource())
	if _, err := pvCache.Get("test-01"); err == nil {
		t.Fatalf("unexpected success getting pv from node indexer")
	}
	if _, err := pvCache.Get("test-02"); err == nil {
		t.Fatalf("unexpected success getting pv from node indexer")
	}
}
