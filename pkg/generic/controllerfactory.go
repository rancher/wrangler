package generic

import (
	"context"
	"sync"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"
)

type ControllerManager struct {
	lock        sync.Mutex
	controllers map[schema.GroupVersionKind]*Controller
	handlers    map[schema.GroupVersionKind]*Handlers
}

func (g *ControllerManager) Start(ctx context.Context, defaultThreadiness int, threadiness map[schema.GroupVersionKind]int) error {
	for gvk, controller := range g.controllers {
		threadiness, ok := threadiness[gvk]
		if !ok {
			threadiness = defaultThreadiness
		}
		if err := controller.Run(threadiness, ctx.Done()); err != nil {
			return err
		}
	}

	return nil
}

func (g *ControllerManager) Enqueue(gvk schema.GroupVersionKind, namespace, name string) {
	controller, ok := g.controllers[gvk]
	if ok {
		controller.Enqueue(namespace, name)
	}
}

func (g *ControllerManager) AddHandler(gvk schema.GroupVersionKind, informer cache.SharedIndexInformer, name string, handler Handler) {
	g.lock.Lock()
	defer g.lock.Unlock()

	entry := handlerEntry{
		name:    name,
		handler: handler,
	}

	handlers, ok := g.handlers[gvk]
	if ok {
		handlers.handlers = append(handlers.handlers, entry)
		return
	}

	handlers = &Handlers{
		handlers: []handlerEntry{entry},
	}

	queue := workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), gvk.String())
	controller := NewController(gvk.String(), informer, queue, handlers.Handle)

	if g.handlers == nil {
		g.handlers = map[schema.GroupVersionKind]*Handlers{}
	}

	if g.controllers == nil {
		g.controllers = map[schema.GroupVersionKind]*Controller{}
	}

	g.handlers[gvk] = handlers
	g.controllers[gvk] = controller
}
