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
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DaemonSetController interface for managing DaemonSet resources.
type DaemonSetController interface {
	generic.ControllerInterface[*v1.DaemonSet, *v1.DaemonSetList]
}

// DaemonSetClient interface for managing DaemonSet resources in Kubernetes.
type DaemonSetClient interface {
	generic.ClientInterface[*v1.DaemonSet, *v1.DaemonSetList]
}

// DaemonSetCache interface for retrieving DaemonSet resources in memory.
type DaemonSetCache interface {
	generic.CacheInterface[*v1.DaemonSet]
}

type DaemonSetStatusHandler func(obj *v1.DaemonSet, status v1.DaemonSetStatus) (v1.DaemonSetStatus, error)

type DaemonSetGeneratingHandler func(obj *v1.DaemonSet, status v1.DaemonSetStatus) ([]runtime.Object, v1.DaemonSetStatus, error)

func RegisterDaemonSetStatusHandler(ctx context.Context, controller DaemonSetController, condition condition.Cond, name string, handler DaemonSetStatusHandler) {
	statusHandler := &daemonSetStatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, generic.FromObjectHandlerToHandler(statusHandler.sync))
}

func RegisterDaemonSetGeneratingHandler(ctx context.Context, controller DaemonSetController, apply apply.Apply,
	condition condition.Cond, name string, handler DaemonSetGeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &daemonSetGeneratingHandler{
		DaemonSetGeneratingHandler: handler,
		apply:                      apply,
		name:                       name,
		gvk:                        controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	RegisterDaemonSetStatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type daemonSetStatusHandler struct {
	client    DaemonSetClient
	condition condition.Cond
	handler   DaemonSetStatusHandler
}

func (a *daemonSetStatusHandler) sync(key string, obj *v1.DaemonSet) (*v1.DaemonSet, error) {
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

type daemonSetGeneratingHandler struct {
	DaemonSetGeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
	seen  sync.Map
}

func (a *daemonSetGeneratingHandler) Remove(key string, obj *v1.DaemonSet) (*v1.DaemonSet, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &v1.DaemonSet{}
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

func (a *daemonSetGeneratingHandler) Handle(obj *v1.DaemonSet, status v1.DaemonSetStatus) (v1.DaemonSetStatus, error) {
	if !obj.DeletionTimestamp.IsZero() {
		return status, nil
	}

	objs, newStatus, err := a.DaemonSetGeneratingHandler(obj, status)
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

func (a *daemonSetGeneratingHandler) isNewResourceVersion(obj *v1.DaemonSet) bool {
	if !a.opts.UniqueApplyForResourceVersion {
		return true
	}

	// Apply once per resource version
	key := obj.Namespace + "/" + obj.Name
	previous, ok := a.seen.Load(key)
	return !ok || previous != obj.ResourceVersion
}

func (a *daemonSetGeneratingHandler) storeResourceVersion(obj *v1.DaemonSet) {
	if !a.opts.UniqueApplyForResourceVersion {
		return
	}

	key := obj.Namespace + "/" + obj.Name
	a.seen.Store(key, obj.ResourceVersion)
}
