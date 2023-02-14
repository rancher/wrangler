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
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// MutatingWebhookConfigurationController interface for managing MutatingWebhookConfiguration resources.
type MutatingWebhookConfigurationController interface {
	generic.ControllerMeta
	MutatingWebhookConfigurationClient

	// OnChange runs the given handler when the controller detects a resource was changed.
	OnChange(ctx context.Context, name string, sync MutatingWebhookConfigurationHandler)

	// OnRemove runs the given handler when the controller detects a resource was changed.
	OnRemove(ctx context.Context, name string, sync MutatingWebhookConfigurationHandler)

	// Enqueue adds the resource with the given name to the worker queue of the controller.
	Enqueue(name string)

	// EnqueueAfter runs Enqueue after the provided duration.
	EnqueueAfter(name string, duration time.Duration)

	// Cache returns a cache for the resource type T.
	Cache() MutatingWebhookConfigurationCache
}

// MutatingWebhookConfigurationClient interface for managing MutatingWebhookConfiguration resources in Kubernetes.
type MutatingWebhookConfigurationClient interface {
	// Create creates a new object and return the newly created Object or an error.
	Create(obj *v1.MutatingWebhookConfiguration, options client.CreateOptions) (*v1.MutatingWebhookConfiguration, error)

	// Update updates the object and return the newly updated Object or an error.
	Update(obj *v1.MutatingWebhookConfiguration, options client.UpdateOptions) (*v1.MutatingWebhookConfiguration, error)

	// Delete deletes the Object in the given name.
	Delete(name string, options client.DeleteOptions) error

	// Get will attempt to retrieve the resource with the specified name.
	Get(name string, options client.GetOptions) (*v1.MutatingWebhookConfiguration, error)

	// List will attempt to find multiple resources.
	List(opts client.ListOptions) (*v1.MutatingWebhookConfigurationList, error)

	// Watch will start watching resources.
	Watch(opts client.ListOptions) (watch.Interface, error)

	// Patch will patch the resource with the matching name.
	Patch(name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (result *v1.MutatingWebhookConfiguration, err error)
}

// MutatingWebhookConfigurationCache interface for retrieving MutatingWebhookConfiguration resources in memory.
type MutatingWebhookConfigurationCache interface {
	// Get returns the resources with the specified name from the cache.
	Get(name string) (*v1.MutatingWebhookConfiguration, error)

	// List will attempt to find resources from the Cache.
	List(selector labels.Selector) ([]*v1.MutatingWebhookConfiguration, error)

	// AddIndexer adds  a new Indexer to the cache with the provided name.
	// If you call this after you already have data in the store, the results are undefined.
	AddIndexer(indexName string, indexer MutatingWebhookConfigurationIndexer)

	// GetByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value.
	GetByIndex(indexName, key string) ([]*v1.MutatingWebhookConfiguration, error)
}

// MutatingWebhookConfigurationHandler is function for performing any potential modifications to a MutatingWebhookConfiguration resource.
type MutatingWebhookConfigurationHandler func(string, *v1.MutatingWebhookConfiguration) (*v1.MutatingWebhookConfiguration, error)

// MutatingWebhookConfigurationIndexer computes a set of indexed values for the provided object.
type MutatingWebhookConfigurationIndexer func(obj *v1.MutatingWebhookConfiguration) ([]string, error)

// MutatingWebhookConfigurationGenericController wraps wrangler/pkg/generic.NonNamespacedController so that the function definitions adhere to MutatingWebhookConfigurationController interface.
type MutatingWebhookConfigurationGenericController struct {
	generic.NonNamespacedControllerInterface[*v1.MutatingWebhookConfiguration, *v1.MutatingWebhookConfigurationList]
}

// OnChange runs the given resource handler when the controller detects a resource was changed.
func (c *MutatingWebhookConfigurationGenericController) OnChange(ctx context.Context, name string, sync MutatingWebhookConfigurationHandler) {
	c.NonNamespacedControllerInterface.OnChange(ctx, name, generic.ObjectHandler[*v1.MutatingWebhookConfiguration](sync))
}

// OnRemove runs the given object handler when the controller detects a resource was changed.
func (c *MutatingWebhookConfigurationGenericController) OnRemove(ctx context.Context, name string, sync MutatingWebhookConfigurationHandler) {
	c.NonNamespacedControllerInterface.OnRemove(ctx, name, generic.ObjectHandler[*v1.MutatingWebhookConfiguration](sync))
}

// Cache returns a cache of resources in memory.
func (c *MutatingWebhookConfigurationGenericController) Cache() MutatingWebhookConfigurationCache {
	return &MutatingWebhookConfigurationGenericCache{
		c.NonNamespacedControllerInterface.Cache(),
	}
}

// MutatingWebhookConfigurationGenericCache wraps wrangler/pkg/generic.NonNamespacedCache so the function definitions adhere to MutatingWebhookConfigurationCache interface.
type MutatingWebhookConfigurationGenericCache struct {
	generic.NonNamespacedCacheInterface[*v1.MutatingWebhookConfiguration]
}

// AddIndexer adds  a new Indexer to the cache with the provided name.
// If you call this after you already have data in the store, the results are undefined.
func (c MutatingWebhookConfigurationGenericCache) AddIndexer(indexName string, indexer MutatingWebhookConfigurationIndexer) {
	c.NonNamespacedCacheInterface.AddIndexer(indexName, generic.Indexer[*v1.MutatingWebhookConfiguration](indexer))
}
