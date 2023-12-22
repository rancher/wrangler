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
	"sync"
	"time"

	"github.com/rancher/wrangler/v2/pkg/apply"
	"github.com/rancher/wrangler/v2/pkg/condition"
	"github.com/rancher/wrangler/v2/pkg/generic"
	"github.com/rancher/wrangler/v2/pkg/kv"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// ServiceController interface for managing Service resources.
type ServiceController interface {
	generic.ControllerInterface[*v1.Service, *v1.ServiceList]
}

// ServiceClient interface for managing Service resources in Kubernetes.
type ServiceClient interface {
	generic.ClientInterface[*v1.Service, *v1.ServiceList]
}

// ServiceCache interface for retrieving Service resources in memory.
type ServiceCache interface {
	generic.CacheInterface[*v1.Service]
}

// ServiceStatusHandler is executed for every added or modified Service. Should return the new status to be updated
type ServiceStatusHandler func(obj *v1.Service, status v1.ServiceStatus) (v1.ServiceStatus, error)

// ServiceGeneratingHandler is the top-level handler that is executed for every Service event. It extends ServiceStatusHandler by a returning a slice of child objects to be passed to apply.Apply
type ServiceGeneratingHandler func(obj *v1.Service, status v1.ServiceStatus) ([]runtime.Object, v1.ServiceStatus, error)

// RegisterServiceStatusHandler configures a ServiceController to execute a ServiceStatusHandler for every events observed.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterServiceStatusHandler(ctx context.Context, controller ServiceController, condition condition.Cond, name string, handler ServiceStatusHandler) {
	statusHandler := &serviceStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, generic.FromObjectHandlerToHandler(statusHandler.sync))
}

// RegisterServiceGeneratingHandler configures a ServiceController to execute a ServiceGeneratingHandler for every events observed, passing the returned objects to the provided apply.Apply.
// If a non-empty condition is provided, it will be updated in the status conditions for every handler execution
func RegisterServiceGeneratingHandler(ctx context.Context, controller ServiceController, apply apply.Apply,
	condition condition.Cond, name string, handler ServiceGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &serviceGeneratingHandler{
		ServiceGeneratingHandler: handler,
		apply:                    apply,
		name:                     name,
		gvk:                      controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterServiceStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type serviceStatusHandler struct {
	client    ServiceClient
	condition condition.Cond
	handler   ServiceStatusHandler
}

// sync is executed on every resource addition or modification. Executes the configured handlers and sends the updated status to the Kubernetes API
func (a *serviceStatusHandler) sync(key string, obj *v1.Service) (*v1.Service, error) {
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
		newObj, newErr := a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
		if newErr == nil {
			obj = newObj
		}
	}
	return obj, err
}

type serviceGeneratingHandler struct {
	ServiceGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

// Remove handles the observed deletion of a resource, cascade deleting every associated resource previously applied
func (a *serviceGeneratingHandler) Remove(key string, obj *v1.Service) (*v1.Service, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1.Service{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	if a.opts.UniqueApplyForResourceVersion {
		a.seen.Delete(key)
	}

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

// Handle executes the configured ServiceGeneratingHandler and pass the resulting objects to apply.Apply, finally returning the new status of the resource
func (a *serviceGeneratingHandler) Handle(obj *v1.Service, status v1.ServiceStatus) (v1.ServiceStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.ServiceGeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}
	if !a.isNewResourceVersion(obj) {
		return newStatus, nil
	}

	err = generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
	if err != nil {
		return newStatus, err
	}
	a.storeResourceVersion(obj)
	return newStatus, nil
}

// isNewResourceVersion detects if a specific resource version was already successfully processed.
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *serviceGeneratingHandler) isNewResourceVersion(obj *v1.Service) bool {
	if !a.opts.UniqueApplyForResourceVersion {
		return true
	}

	// Apply once per resource version
	key := obj.Namespace + "/" + obj.Name
	previous, ok := a.seen.Load(key)
	return !ok || previous != obj.ResourceVersion
}

// storeResourceVersion keeps track of the latest resource version of an object for which Apply was executed
// Only used if UniqueApplyForResourceVersion is set in generic.GeneratingHandlerOptions
func (a *serviceGeneratingHandler) storeResourceVersion(obj *v1.Service) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
