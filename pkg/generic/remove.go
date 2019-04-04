package generic

import (
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	finalizerKey = "wrangler.cattle.io/"
)

type Updater func(runtime.Object) (runtime.Object, error)

type objectLifecycleAdapter struct {
	name    string
	handler Handler
	updater Updater
}

func NewRemoveHandler(name string, updater Updater, handler Handler) Handler {
	o := objectLifecycleAdapter{
		name:    name,
		handler: handler,
		updater: updater,
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

	finalizerKey := o.constructFinalizerKey()
	finalizers := metadata.GetFinalizers()

	for k, v := range finalizers {
		if v == finalizerKey {
			newFinalizers := append(finalizers[:k], finalizers[k+1:]...)
			newObj, err := o.handler(key, obj)
			if err != nil {
				return newObj, err
			}

			if newObj != nil {
				obj = newObj
			}

			obj = obj.DeepCopyObject()
			metadata, err = meta.Accessor(obj)

			if err != nil {
				return obj, err
			}

			metadata.SetFinalizers(newFinalizers)

			return o.updater(obj)
		}
	}

	return obj, nil
}

func (o *objectLifecycleAdapter) constructFinalizerKey() string {
	return finalizerKey + o.name
}

func (o *objectLifecycleAdapter) addFinalizer(obj runtime.Object) (runtime.Object, error) {

	metadata, err := meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	finalizerKey := o.constructFinalizerKey()
	finalizers := metadata.GetFinalizers()

	for _, finalizer := range finalizers {
		if finalizer == finalizerKey {
			return obj, nil
		}
	}

	obj = obj.DeepCopyObject()
	metadata, err = meta.Accessor(obj)
	if err != nil {
		return nil, err
	}

	metadata.SetFinalizers(append(finalizers, finalizerKey))

	return o.updater(obj)
}
