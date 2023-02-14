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
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

// NodeController interface for managing Node resources.
type NodeController interface {
	generic.ControllerMeta
	NodeClient

	// OnChange runs the given handler when the controller detects a resource was changed.
	OnChange(ctx context.Context, name string, sync NodeHandler)

	// OnRemove runs the given handler when the controller detects a resource was changed.
	OnRemove(ctx context.Context, name string, sync NodeHandler)

	// Enqueue adds the resource with the given name to the worker queue of the controller.
	Enqueue(name string)

	// EnqueueAfter runs Enqueue after the provided duration.
	EnqueueAfter(name string, duration time.Duration)

	// Cache returns a cache for the resource type T.
	Cache() NodeCache
}

// NodeClient interface for managing Node resources in Kubernetes.
type NodeClient interface {
	// Create creates a new object and return the newly created Object or an error.
	Create(obj *v1.Node, options client.CreateOptions) (*v1.Node, error)

	// Update updates the object and return the newly updated Object or an error.
	Update(obj *v1.Node, options client.UpdateOptions) (*v1.Node, error)
	// UpdateStatus updates the Status field of a the object and return the newly updated Object or an error.
	// Will always return an error if the object does not have a status field.
	UpdateStatus(obj *v1.Node, options client.UpdateOptions) (*v1.Node, error)

	// Delete deletes the Object in the given name.
	Delete(name string, options client.DeleteOptions) error

	// Get will attempt to retrieve the resource with the specified name.
	Get(name string, options client.GetOptions) (*v1.Node, error)

	// List will attempt to find multiple resources.
	List(opts client.ListOptions) (*v1.NodeList, error)

	// Watch will start watching resources.
	Watch(opts client.ListOptions) (watch.Interface, error)

	// Patch will patch the resource with the matching name.
	Patch(name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (result *v1.Node, err error)
}

// NodeCache interface for retrieving Node resources in memory.
type NodeCache interface {
	// Get returns the resources with the specified name from the cache.
	Get(name string) (*v1.Node, error)

	// List will attempt to find resources from the Cache.
	List(selector labels.Selector) ([]*v1.Node, error)

	// AddIndexer adds  a new Indexer to the cache with the provided name.
	// If you call this after you already have data in the store, the results are undefined.
	AddIndexer(indexName string, indexer NodeIndexer)

	// GetByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value.
	GetByIndex(indexName, key string) ([]*v1.Node, error)
}

// NodeHandler is function for performing any potential modifications to a Node resource.
type NodeHandler func(string, *v1.Node) (*v1.Node, error)

// NodeIndexer computes a set of indexed values for the provided object.
type NodeIndexer func(obj *v1.Node) ([]string, error)

// NodeGenericController wraps wrangler/pkg/generic.NonNamespacedController so that the function definitions adhere to NodeController interface.
type NodeGenericController struct {
	generic.NonNamespacedControllerInterface[*v1.Node, *v1.NodeList]
}

// OnChange runs the given resource handler when the controller detects a resource was changed.
func (c *NodeGenericController) OnChange(ctx context.Context, name string, sync NodeHandler) {
	c.NonNamespacedControllerInterface.OnChange(ctx, name, generic.ObjectHandler[*v1.Node](sync))
}

// OnRemove runs the given object handler when the controller detects a resource was changed.
func (c *NodeGenericController) OnRemove(ctx context.Context, name string, sync NodeHandler) {
	c.NonNamespacedControllerInterface.OnRemove(ctx, name, generic.ObjectHandler[*v1.Node](sync))
}

// Cache returns a cache of resources in memory.
func (c *NodeGenericController) Cache() NodeCache {
	return &NodeGenericCache{
		c.NonNamespacedControllerInterface.Cache(),
	}
}

// NodeGenericCache wraps wrangler/pkg/generic.NonNamespacedCache so the function definitions adhere to NodeCache interface.
type NodeGenericCache struct {
	generic.NonNamespacedCacheInterface[*v1.Node]
}

// AddIndexer adds  a new Indexer to the cache with the provided name.
// If you call this after you already have data in the store, the results are undefined.
func (c NodeGenericCache) AddIndexer(indexName string, indexer NodeIndexer) {
	c.NonNamespacedCacheInterface.AddIndexer(indexName, generic.Indexer[*v1.Node](indexer))
}

type NodeStatusHandler func(obj *v1.Node, status v1.NodeStatus) (v1.NodeStatus, error)

type NodeGeneratingHandler func(obj *v1.Node, status v1.NodeStatus) ([]runtime.Object, v1.NodeStatus, error)

func FromNodeHandlerToHandler(sync NodeHandler) generic.Handler {
	return generic.FromObjectHandlerToHandler(generic.ObjectHandler[*v1.Node](sync))
}

func RegisterNodeStatusHandler(ctx context.Context, controller NodeController, condition condition.Cond, name string, handler NodeStatusHandler) {
	statusHandler := &nodeStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromNodeHandlerToHandler(statusHandler.sync))
}

func RegisterNodeGeneratingHandler(ctx context.Context, controller NodeController, apply apply.Apply,
	condition condition.Cond, name string, handler NodeGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &nodeGeneratingHandler{
		NodeGeneratingHandler: handler,
		apply:                 apply,
		name:                  name,
		gvk:                   controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterNodeStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type nodeStatusHandler struct {
	client    NodeClient
	condition condition.Cond
	handler   NodeStatusHandler
}

func (a *nodeStatusHandler) sync(key string, obj *v1.Node) (*v1.Node, error) {
	if obj == nil {
		return obj, nil
	}

	origStatus := obj.Status.DeepCopy()
	obj = obj.DeepCopy()
	newStatus, err := a.handler(obj, obj.Status)
	if err != nil {
		// Revert to old status on error
		newStatus = *origStatus.DeepCopy()
	}

	if a.condition != "" {
		if errors.IsConflict(err) {
			a.condition.SetError(&newStatus, "", nil)
		} else {
			a.condition.SetError(&newStatus, "", err)
		}
	}
	if !equality.Semantic.DeepEqual(origStatus, &newStatus) {
		if a.condition != "" {
			// Since status has changed, update the lastUpdatedTime
			a.condition.LastUpdated(&newStatus, time.Now().UTC().Format(time.RFC3339))
		}

		var newErr error
		obj.Status = newStatus
		newObj, newErr := a.client.UpdateStatus(obj, client.UpdateOptions{})
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type nodeGeneratingHandler struct {
	NodeGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *nodeGeneratingHandler) Remove(key string, obj *v1.Node) (*v1.Node, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1.Node{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *nodeGeneratingHandler) Handle(obj *v1.Node, status v1.NodeStatus) (v1.NodeStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.NodeGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
