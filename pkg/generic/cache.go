package generic

import (
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/cache"
)

// Cache is a object cache stored in memory for objects of type T.
type Cache[T RuntimeMetaObject] struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

// NonNamespacedCache is a Cache for objects of type T that are not namespaced.
type NonNamespacedCache[T RuntimeMetaObject] struct {
	Cache[T]
}

// Get returns the given object in the given namespace if it is found in the cache.
func (c *Cache[T]) Get(namespace, name string) (T, error) {
	var nilObj T
	key := name
	if namespace != metav1.NamespaceAll {
		key = namespace + "/" + key
	}
	obj, exists, err := c.indexer.GetByKey(key)
	if err != nil {
		return nilObj, err
	}
	if !exists {
		return nilObj, errors.NewNotFound(c.resource, name)
	}
	return obj.(T), nil
}

// List will attempt to find resources in the given namespace from the Object Cache.
func (c *Cache[T]) List(namespace string, selector labels.Selector) (ret []T, err error) {
	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(T))
	})

	return ret, err
}

func (c *Cache[T]) AddIndexer(indexName string, indexer Indexer[T]) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(T))
		},
	}))
}

func (c *Cache[T]) GetByIndex(indexName, key string) (result []T, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]T, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(T))
	}
	return result, nil
}

// Get calls Cache.Get(...) with an empty namespace parameter.
func (c *NonNamespacedCache[T]) Get(name string) (T, error) {
	return c.Cache.Get(metav1.NamespaceAll, name)
}

// Get calls Cache.List(...) with an empty namespace parameter.
func (c *NonNamespacedCache[T]) List(selector labels.Selector) (ret []T, err error) {
	return c.Cache.List(metav1.NamespaceAll, selector)
}
