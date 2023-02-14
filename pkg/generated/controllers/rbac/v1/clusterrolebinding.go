/*
Copyright The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by main. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/wrangler/pkg/generic"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// ClusterRoleBindingController interface for managing ClusterRoleBinding resources.
type ClusterRoleBindingController interface {
	generic.ControllerMeta
	ClusterRoleBindingClient

	// OnChange runs the given handler when the controller detects a resource was changed.
	OnChange(ctx context.Context, name string, sync ClusterRoleBindingHandler)

	// OnRemove runs the given handler when the controller detects a resource was changed.
	OnRemove(ctx context.Context, name string, sync ClusterRoleBindingHandler)

	// Enqueue adds the resource with the given name to the worker queue of the controller.
	Enqueue(name string)

	// EnqueueAfter runs Enqueue after the provided duration.
	EnqueueAfter(name string, duration time.Duration)

	// Cache returns a cache for the resource type T.
	Cache() ClusterRoleBindingCache
}

// ClusterRoleBindingClient interface for managing ClusterRoleBinding resources in Kubernetes.
type ClusterRoleBindingClient interface {
	// Create creates a new object and return the newly created Object or an error.
	Create(obj *v1.ClusterRoleBinding, options client.CreateOptions) (*v1.ClusterRoleBinding, error)

	// Update updates the object and return the newly updated Object or an error.
	Update(obj *v1.ClusterRoleBinding, options client.UpdateOptions) (*v1.ClusterRoleBinding, error)

	// Delete deletes the Object in the given name.
	Delete(name string, options client.DeleteOptions) error

	// Get will attempt to retrieve the resource with the specified name.
	Get(name string, options client.GetOptions) (*v1.ClusterRoleBinding, error)

	// List will attempt to find multiple resources.
	List(opts client.ListOptions) (*v1.ClusterRoleBindingList, error)

	// Watch will start watching resources.
	Watch(opts client.ListOptions) (watch.Interface, error)

	// Patch will patch the resource with the matching name.
	Patch(name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (result *v1.ClusterRoleBinding, err error)
}

// ClusterRoleBindingCache interface for retrieving ClusterRoleBinding resources in memory.
type ClusterRoleBindingCache interface {
	// Get returns the resources with the specified name from the cache.
	Get(name string) (*v1.ClusterRoleBinding, error)

	// List will attempt to find resources from the Cache.
	List(selector labels.Selector) ([]*v1.ClusterRoleBinding, error)

	// AddIndexer adds  a new Indexer to the cache with the provided name.
	// If you call this after you already have data in the store, the results are undefined.
	AddIndexer(indexName string, indexer ClusterRoleBindingIndexer)

	// GetByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value.
	GetByIndex(indexName, key string) ([]*v1.ClusterRoleBinding, error)
}

// ClusterRoleBindingHandler is function for performing any potential modifications to a ClusterRoleBinding resource.
type ClusterRoleBindingHandler func(string, *v1.ClusterRoleBinding) (*v1.ClusterRoleBinding, error)

// ClusterRoleBindingIndexer computes a set of indexed values for the provided object.
type ClusterRoleBindingIndexer func(obj *v1.ClusterRoleBinding) ([]string, error)

// ClusterRoleBindingGenericController wraps wrangler/pkg/generic.NonNamespacedController so that the function definitions adhere to ClusterRoleBindingController interface.
type ClusterRoleBindingGenericController struct {
	generic.NonNamespacedControllerInterface[*v1.ClusterRoleBinding, *v1.ClusterRoleBindingList]
}

// OnChange runs the given resource handler when the controller detects a resource was changed.
func (c *ClusterRoleBindingGenericController) OnChange(ctx context.Context, name string, sync ClusterRoleBindingHandler) {
	c.NonNamespacedControllerInterface.OnChange(ctx, name, generic.ObjectHandler[*v1.ClusterRoleBinding](sync))
}

// OnRemove runs the given object handler when the controller detects a resource was changed.
func (c *ClusterRoleBindingGenericController) OnRemove(ctx context.Context, name string, sync ClusterRoleBindingHandler) {
	c.NonNamespacedControllerInterface.OnRemove(ctx, name, generic.ObjectHandler[*v1.ClusterRoleBinding](sync))
}

// Cache returns a cache of resources in memory.
func (c *ClusterRoleBindingGenericController) Cache() ClusterRoleBindingCache {
	return &ClusterRoleBindingGenericCache{
		c.NonNamespacedControllerInterface.Cache(),
	}
}

// ClusterRoleBindingGenericCache wraps wrangler/pkg/generic.NonNamespacedCache so the function definitions adhere to ClusterRoleBindingCache interface.
type ClusterRoleBindingGenericCache struct {
	generic.NonNamespacedCacheInterface[*v1.ClusterRoleBinding]
}

// AddIndexer adds  a new Indexer to the cache with the provided name.
// If you call this after you already have data in the store, the results are undefined.
func (c ClusterRoleBindingGenericCache) AddIndexer(indexName string, indexer ClusterRoleBindingIndexer) {
	c.NonNamespacedCacheInterface.AddIndexer(indexName, generic.Indexer[*v1.ClusterRoleBinding](indexer))
}
