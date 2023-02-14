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

// PersistentVolumeClaimController interface for managing PersistentVolumeClaim resources.
type PersistentVolumeClaimController interface {
	generic.ControllerMeta
	PersistentVolumeClaimClient

	// OnChange runs the given handler when the controller detects a resource was changed.
	OnChange(ctx context.Context, name string, sync PersistentVolumeClaimHandler)

	// OnRemove runs the given handler when the controller detects a resource was changed.
	OnRemove(ctx context.Context, name string, sync PersistentVolumeClaimHandler)

	// Enqueue adds the resource with the given name to the worker queue of the controller.
	Enqueue(namespace, name string)

	// EnqueueAfter runs Enqueue after the provided duration.
	EnqueueAfter(namespace, name string, duration time.Duration)

	// Cache returns a cache for the resource type T.
	Cache() PersistentVolumeClaimCache
}

// PersistentVolumeClaimClient interface for managing PersistentVolumeClaim resources in Kubernetes.
type PersistentVolumeClaimClient interface {
	// Create creates a new object and return the newly created Object or an error.
	Create(obj *v1.PersistentVolumeClaim, options client.CreateOptions) (*v1.PersistentVolumeClaim, error)

	// Update updates the object and return the newly updated Object or an error.
	Update(obj *v1.PersistentVolumeClaim, options client.UpdateOptions) (*v1.PersistentVolumeClaim, error)
	// UpdateStatus updates the Status field of a the object and return the newly updated Object or an error.
	// Will always return an error if the object does not have a status field.
	UpdateStatus(obj *v1.PersistentVolumeClaim, options client.UpdateOptions) (*v1.PersistentVolumeClaim, error)

	// Delete deletes the Object in the given name.
	Delete(namespace, name string, options client.DeleteOptions) error

	// Get will attempt to retrieve the resource with the specified name.
	Get(namespace, name string, options client.GetOptions) (*v1.PersistentVolumeClaim, error)

	// List will attempt to find multiple resources.
	List(namespace string, opts client.ListOptions) (*v1.PersistentVolumeClaimList, error)

	// Watch will start watching resources.
	Watch(namespace string, opts client.ListOptions) (watch.Interface, error)

	// Patch will patch the resource with the matching name.
	Patch(namespace, name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (result *v1.PersistentVolumeClaim, err error)
}

// PersistentVolumeClaimCache interface for retrieving PersistentVolumeClaim resources in memory.
type PersistentVolumeClaimCache interface {
	// Get returns the resources with the specified name from the cache.
	Get(namespace, name string) (*v1.PersistentVolumeClaim, error)

	// List will attempt to find resources from the Cache.
	List(namespace string, selector labels.Selector) ([]*v1.PersistentVolumeClaim, error)

	// AddIndexer adds  a new Indexer to the cache with the provided name.
	// If you call this after you already have data in the store, the results are undefined.
	AddIndexer(indexName string, indexer PersistentVolumeClaimIndexer)

	// GetByIndex returns the stored objects whose set of indexed values
	// for the named index includes the given indexed value.
	GetByIndex(indexName, key string) ([]*v1.PersistentVolumeClaim, error)
}

// PersistentVolumeClaimHandler is function for performing any potential modifications to a PersistentVolumeClaim resource.
type PersistentVolumeClaimHandler func(string, *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error)

// PersistentVolumeClaimIndexer computes a set of indexed values for the provided object.
type PersistentVolumeClaimIndexer func(obj *v1.PersistentVolumeClaim) ([]string, error)

// PersistentVolumeClaimGenericController wraps wrangler/pkg/generic.Controller so that the function definitions adhere to PersistentVolumeClaimController interface.
type PersistentVolumeClaimGenericController struct {
	generic.ControllerInterface[*v1.PersistentVolumeClaim, *v1.PersistentVolumeClaimList]
}

// OnChange runs the given resource handler when the controller detects a resource was changed.
func (c *PersistentVolumeClaimGenericController) OnChange(ctx context.Context, name string, sync PersistentVolumeClaimHandler) {
	c.ControllerInterface.OnChange(ctx, name, generic.ObjectHandler[*v1.PersistentVolumeClaim](sync))
}

// OnRemove runs the given object handler when the controller detects a resource was changed.
func (c *PersistentVolumeClaimGenericController) OnRemove(ctx context.Context, name string, sync PersistentVolumeClaimHandler) {
	c.ControllerInterface.OnRemove(ctx, name, generic.ObjectHandler[*v1.PersistentVolumeClaim](sync))
}

// Cache returns a cache of resources in memory.
func (c *PersistentVolumeClaimGenericController) Cache() PersistentVolumeClaimCache {
	return &PersistentVolumeClaimGenericCache{
		c.ControllerInterface.Cache(),
	}
}

// PersistentVolumeClaimGenericCache wraps wrangler/pkg/generic.Cache so the function definitions adhere to PersistentVolumeClaimCache interface.
type PersistentVolumeClaimGenericCache struct {
	generic.CacheInterface[*v1.PersistentVolumeClaim]
}

// AddIndexer adds  a new Indexer to the cache with the provided name.
// If you call this after you already have data in the store, the results are undefined.
func (c PersistentVolumeClaimGenericCache) AddIndexer(indexName string, indexer PersistentVolumeClaimIndexer) {
	c.CacheInterface.AddIndexer(indexName, generic.Indexer[*v1.PersistentVolumeClaim](indexer))
}

type PersistentVolumeClaimStatusHandler func(obj *v1.PersistentVolumeClaim, status v1.PersistentVolumeClaimStatus) (v1.PersistentVolumeClaimStatus, error)

type PersistentVolumeClaimGeneratingHandler func(obj *v1.PersistentVolumeClaim, status v1.PersistentVolumeClaimStatus) ([]runtime.Object, v1.PersistentVolumeClaimStatus, error)

func FromPersistentVolumeClaimHandlerToHandler(sync PersistentVolumeClaimHandler) generic.Handler {
	return generic.FromObjectHandlerToHandler(generic.ObjectHandler[*v1.PersistentVolumeClaim](sync))
}

func RegisterPersistentVolumeClaimStatusHandler(ctx context.Context, controller PersistentVolumeClaimController, condition condition.Cond, name string, handler PersistentVolumeClaimStatusHandler) {
	statusHandler := &persistentVolumeClaimStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, FromPersistentVolumeClaimHandlerToHandler(statusHandler.sync))
}

func RegisterPersistentVolumeClaimGeneratingHandler(ctx context.Context, controller PersistentVolumeClaimController, apply apply.Apply,
	condition condition.Cond, name string, handler PersistentVolumeClaimGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &persistentVolumeClaimGeneratingHandler{
		PersistentVolumeClaimGeneratingHandler: handler,
		apply:                                  apply,
		name:                                   name,
		gvk:                                    controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterPersistentVolumeClaimStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type persistentVolumeClaimStatusHandler struct {
	client    PersistentVolumeClaimClient
	condition condition.Cond
	handler   PersistentVolumeClaimStatusHandler
}

func (a *persistentVolumeClaimStatusHandler) sync(key string, obj *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
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

type persistentVolumeClaimGeneratingHandler struct {
	PersistentVolumeClaimGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *persistentVolumeClaimGeneratingHandler) Remove(key string, obj *v1.PersistentVolumeClaim) (*v1.PersistentVolumeClaim, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1.PersistentVolumeClaim{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *persistentVolumeClaimGeneratingHandler) Handle(obj *v1.PersistentVolumeClaim, status v1.PersistentVolumeClaimStatus) (v1.PersistentVolumeClaimStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.PersistentVolumeClaimGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
