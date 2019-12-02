package relatedresource

import (
	"context"

	"k8s.io/apimachinery/pkg/api/meta"

	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
)

/*
	The related resource pkg allows for the chaining of disparate objects via a shared informer.
	It requires that the caller be able to provide a Resolver and a Enqueuer function.
	The Resolver should be able to return a Key for the specified object

*/
type Key struct {
	Namespace string
	Name      string
}

func NewKey(namespace, name string) Key {
	return Key{
		Namespace: namespace,
		Name:      name,
	}
}

type ControllerWrapper interface {
	Informer() cache.SharedIndexInformer
	AddGenericHandler(ctx context.Context, name string, handler generic.Handler)
}

type Enqueuer interface {
	Enqueue(namespace, name string)
}
type EnqueuerClusterScope interface {
	Enqueue(name string)
}

type Resolver func(namespace, name string, obj runtime.Object) ([]Key, error)
type ResolverClusterScope func(name string, obj runtime.Object) ([]Key, error)

func Watch(ctx context.Context, name string, resolve Resolver, enq Enqueuer, watching ...ControllerWrapper) {
	for _, c := range watching {
		watch(ctx, name, resolve, enq, c)
	}
}

func WatchClusterScope(ctx context.Context, name string, resolve ResolverClusterScope, enq EnqueuerClusterScope, watching ...ControllerWrapper) {
	for _, c := range watching {
		watchClusterScope(ctx, name, resolve, enq, c)
	}
}

func watch(ctx context.Context, name string, resolve Resolver, enq Enqueuer, controller ControllerWrapper) {
	runResolve := func(ns, name string, obj runtime.Object) error {
		keys, err := resolve(ns, name, obj)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key.Name != "" {
				enq.Enqueue(key.Namespace, key.Name)
			}
		}

		return nil
	}

	controller.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			ro, ok := obj.(runtime.Object)
			if !ok {
				return
			}

			meta, err := meta.Accessor(ro)
			if err != nil {
				return
			}

			runResolve(meta.GetNamespace(), meta.GetName(), ro)
		},
	})

	controller.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		ns, name := kv.Split(key, "/")
		return obj, runResolve(ns, name, obj)
	})
}

func watchClusterScope(ctx context.Context, name string, resolve ResolverClusterScope, enq EnqueuerClusterScope, controller ControllerWrapper) {
	runResolve := func(name string, obj runtime.Object) error {
		keys, err := resolve(name, obj)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key.Name != "" {
				enq.Enqueue(key.Name)
			}
		}

		return nil
	}

	controller.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		DeleteFunc: func(obj interface{}) {
			ro, ok := obj.(runtime.Object)
			if !ok {
				return
			}

			meta, err := meta.Accessor(ro)
			if err != nil {
				return
			}
			runResolve(meta.GetName(), ro)
		},
	})

	controller.AddGenericHandler(ctx, name, func(key string, obj runtime.Object) (runtime.Object, error) {
		_, name := kv.Split(key, "/")
		return obj, runResolve(name, obj)
	})
}
