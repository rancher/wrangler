package generic

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	finalizerKey = "wrangler.cattle.io/"
)

type Updater func(runtime.Object) (runtime.Object, error)

// objectLifecycleAdapter adds a finalizer to the resources to block resources deletion until the configured handler could be executed
type objectLifecycleAdapter struct {
	name      string
	handler   Handler
	updater   Updater
	condition func(runtime.Object) bool
}

func NewRemoveHandler(name string, updater Updater, handler Handler, opts ...OnRemoveOption) Handler {
	o := objectLifecycleAdapter{
		name:      name,
		handler:   handler,
		updater:   updater,
		condition: includeAll,
	}
	for _, opt := range opts {
		opt(&o)
	}
	return o.sync
}

func (o *objectLifecycleAdapter) sync(key string, obj runtime.Object) (runtime.Object, error) {
	if obj == nil {
		return nil, nil
	}

	metadata, err := meta.Accessor(obj)
	if err != nil {
		return obj, err
	}

	if metadata.GetDeletionTimestamp() == nil {
		return o.addFinalizer(obj)
	}

	if !o.hasFinalizer(obj) {
		return obj, nil
	}

	newObj, err := o.handler(key, obj)
	if err != nil {
		return newObj, err
	}

	if newObj != nil {
		obj = newObj
	}

	return o.removeFinalizer(obj)
}

func (o *objectLifecycleAdapter) constructFinalizerKey() string {
	return finalizerKey + o.name
}

func (o *objectLifecycleAdapter) hasFinalizer(obj runtime.Object) bool {
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return false
	}

	finalizerKey := o.constructFinalizerKey()
	finalizers := metadata.GetFinalizers()
	for _, finalizer := range finalizers {
		if finalizer == finalizerKey {
			return true
		}
	}

	return false
}

func (o *objectLifecycleAdapter) removeFinalizer(obj runtime.Object) (runtime.Object, error) {
	if !o.hasFinalizer(obj) {
		return obj, nil
	}

	obj = obj.DeepCopyObject()
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return obj, err
	}

	finalizerKey := o.constructFinalizerKey()
	finalizers := metadata.GetFinalizers()

	var newFinalizers []string
	for k, v := range finalizers {
		if v != finalizerKey {
			continue
		}
		newFinalizers = append(finalizers[:k], finalizers[k+1:]...)
	}

	metadata.SetFinalizers(newFinalizers)
	return o.updater(obj)
}

func (o *objectLifecycleAdapter) addFinalizer(obj runtime.Object) (runtime.Object, error) {
	if o.hasFinalizer(obj) || !o.condition(obj) {
		return obj, nil
	}

	obj = obj.DeepCopyObject()
	metadata, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	metadata.SetFinalizers(append(metadata.GetFinalizers(), o.constructFinalizerKey()))
	return o.updater(obj)
}
