package generator

var ControllerTemplate = `package {{.Version}}

import (
	"context"
	"time"

	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/lasso/pkg/controller"
	{{.Version}} "github.com/rancher/rancher/pkg/apis/{{.Group}}/{{.Version}}"
    "github.com/rancher/wrangler/pkg/apply"
    "github.com/rancher/wrangler/pkg/condition"
	"github.com/rancher/wrangler/pkg/generic"
    "github.com/rancher/wrangler/pkg/kv"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

type {{.Name}}Handler func(string, *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error)

type {{.Name}}Controller interface {
    generic.ControllerMeta
	{{.Name}}Client

	OnChange(ctx context.Context, name string, sync {{.Name}}Handler)
	OnRemove(ctx context.Context, name string, sync {{.Name}}Handler)
	Enqueue({{ if .namespaced}}namespace, {{end}}name string)
	EnqueueAfter({{ if .namespaced}}namespace, {{end}}name string, duration time.Duration)

	Cache() {{.Name}}Cache
}

type {{.Name}}Client interface {
	Create(*{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error)
	Update(*{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error)
{{ if .hasStatus -}}
	UpdateStatus(*{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error)
{{- end }}
	Delete({{ if .namespaced}}namespace, {{end}}name string, options *metav1.DeleteOptions) error
	Get({{ if .namespaced}}namespace, {{end}}name string, options metav1.GetOptions) (*{{.Version}}.{{.Name}}, error)
	List({{ if .namespaced}}namespace string, {{end}}opts metav1.ListOptions) (*{{.Version}}.{{.Name}}List, error)
	Watch({{ if .namespaced}}namespace string, {{end}}opts metav1.ListOptions) (watch.Interface, error)
	Patch({{ if .namespaced}}namespace, {{end}}name string, pt types.PatchType, data []byte, subresources ...string) (result *{{.Version}}.{{.Name}}, err error)
}

type {{.Name}}Cache interface {
	Get({{ if .namespaced}}namespace, {{end}}name string) (*{{.Version}}.{{.Name}}, error)
	List({{ if .namespaced}}namespace string, {{end}}selector labels.Selector) ([]*{{.Version}}.{{.Name}}, error)

	AddIndexer(indexName string, indexer {{.Name}}Indexer)
	GetByIndex(indexName, key string) ([]*{{.Version}}.{{.Name}}, error)
}

type {{.Name}}Indexer func(obj *{{.Version}}.{{.Name}}) ([]string, error)

type {{.Name | unCapitalize}}Controller struct {
	controller controller.SharedController
	client            *client.Client
	gvk               schema.GroupVersionKind
	groupResource     schema.GroupResource
}

func New{{.Name}}Controller(gvk schema.GroupVersionKind, resource string, namespaced bool, controller controller.SharedControllerFactory) {{.Name}}Controller {
	c := controller.ForResourceKind(gvk.GroupVersion().WithResource(resource), gvk.Kind, namespaced)
	return &{{.Name | unCapitalize}}Controller{
		controller: c,
		client:     c.Client(),
		gvk:        gvk,
		groupResource: schema.GroupResource{
			Group:    gvk.Group,
			Resource: resource,
		},
	}
}

func From{{.Name}}HandlerToHandler(sync {{.Name}}Handler) generic.Handler {
	return func(key string, obj runtime.Object) (ret runtime.Object, err error) {
		var v *{{.Version}}.{{.Name}}
		if obj == nil {
			v, err = sync(key, nil)
		} else {
			v, err = sync(key, obj.(*{{.Version}}.{{.Name}}))
		}
		if v == nil {
			return nil, err
		}
		return v, err
	}
}

func (c *{{.Name | unCapitalize}}Controller) Updater() generic.Updater {
	return func(obj runtime.Object) (runtime.Object, error) {
		newObj, err := c.Update(obj.(*{{.Version}}.{{.Name}}))
		if newObj == nil {
			return nil, err
		}
		return newObj, err
	}
}

func Update{{.Name}}DeepCopyOnChange(client {{.Name}}Client, obj *{{.Version}}.{{.Name}}, handler func(obj *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error)) (*{{.Version}}.{{.Name}}, error) {
	if obj == nil {
		return obj, nil
	}

	copyObj := obj.DeepCopy()
	newObj, err := handler(copyObj)
	if newObj != nil {
		copyObj = newObj
	}
	if obj.ResourceVersion == copyObj.ResourceVersion && !equality.Semantic.DeepEqual(obj, copyObj) {
		return client.Update(copyObj)
	}

	return copyObj, err
}

func (c *{{.Name | unCapitalize}}Controller) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	c.controller.RegisterHandler(ctx, name, controller.SharedControllerHandlerFunc(handler))
}

func (c *{{.Name | unCapitalize}}Controller) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), handler))
}

func (c *{{.Name | unCapitalize}}Controller) OnChange(ctx context.Context, name string, sync {{.Name}}Handler) {
	c.AddGenericHandler(ctx, name, From{{.Name}}HandlerToHandler(sync))
}

func (c *{{.Name | unCapitalize}}Controller) OnRemove(ctx context.Context, name string, sync {{.Name}}Handler) {
	c.AddGenericHandler(ctx, name, generic.NewRemoveHandler(name, c.Updater(), From{{.Name}}HandlerToHandler(sync)))
}

func (c *{{.Name | unCapitalize}}Controller) Enqueue({{ if .namespaced}}namespace, {{end}}name string) {
	c.controller.Enqueue({{ if .namespaced }}namespace, {{else}}"", {{end}}name)
}

func (c *{{.Name | unCapitalize}}Controller) EnqueueAfter({{ if .namespaced}}namespace, {{end}}name string, duration time.Duration) {
	c.controller.EnqueueAfter({{ if .namespaced }}namespace, {{else}}"", {{end}}name, duration)
}

func (c *{{.Name | unCapitalize}}Controller) Informer() cache.SharedIndexInformer {
	return c.controller.Informer()
}

func (c *{{.Name | unCapitalize}}Controller) GroupVersionKind() schema.GroupVersionKind {
	return c.gvk
}

func (c *{{.Name | unCapitalize}}Controller) Cache() {{.Name}}Cache {
	return &{{.Name | unCapitalize}}Cache{
		indexer:  c.Informer().GetIndexer(),
		resource: c.groupResource,
	}
}

func (c *{{.Name | unCapitalize}}Controller) Create(obj *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error) {
	result := &{{.Version}}.{{.Name}}{}
	return result, c.client.Create(context.TODO(), {{ if .namespaced}}obj.Namespace,{{else}}"",{{end}} obj, result, metav1.CreateOptions{})
}

func (c *{{.Name | unCapitalize}}Controller) Update(obj *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error) {
	result := &{{.Version}}.{{.Name}}{}
	return result, c.client.Update(context.TODO(), {{ if .namespaced}}obj.Namespace,{{else}}"",{{end}} obj, result, metav1.UpdateOptions{})
}

{{ if .hasStatus -}}
func (c *{{.Name | unCapitalize}}Controller) UpdateStatus(obj *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error) {
	result := &{{.Version}}.{{.Name}}{}
	return result, c.client.UpdateStatus(context.TODO(), {{ if .namespaced}}obj.Namespace,{{else}}"",{{end}} obj, result, metav1.UpdateOptions{})
}
{{- end }}

func (c *{{.Name | unCapitalize}}Controller) Delete({{ if .namespaced}}namespace, {{end}}name string, options *metav1.DeleteOptions) error {
	if options == nil {
		options = &metav1.DeleteOptions{}
	}
	return c.client.Delete(context.TODO(), {{ if .namespaced}}namespace,{{else}}"",{{end}} name, *options)
}

func (c *{{.Name | unCapitalize}}Controller) Get({{ if .namespaced}}namespace, {{end}}name string, options metav1.GetOptions) (*{{.Version}}.{{.Name}}, error) {
	result := &{{.Version}}.{{.Name}}{}
	return result, c.client.Get(context.TODO(), {{ if .namespaced}}namespace,{{else}}"",{{end}} name, result, options)
}

func (c *{{.Name | unCapitalize}}Controller) List({{ if .namespaced}}namespace string, {{end}}opts metav1.ListOptions) (*{{.Version}}.{{.Name}}List, error) {
	result := &{{.Version}}.{{.Name}}List{}
	return result, c.client.List(context.TODO(), {{ if .namespaced}}namespace,{{else}}"",{{end}} result, opts)
}

func (c *{{.Name | unCapitalize}}Controller) Watch({{ if .namespaced}}namespace string, {{end}}opts metav1.ListOptions) (watch.Interface, error) {
	return c.client.Watch(context.TODO(), {{ if .namespaced}}namespace,{{else}}"",{{end}} opts)
}

func (c *{{.Name | unCapitalize}}Controller) Patch({{ if .namespaced}}namespace, {{end}}name string, pt types.PatchType, data []byte, subresources ...string) (*{{.Version}}.{{.Name}}, error) {
	result := &{{.Version}}.{{.Name}}{}
	return result, c.client.Patch(context.TODO(), {{ if .namespaced}}namespace,{{else}}"",{{end}} name, pt, data, result, metav1.PatchOptions{}, subresources...)
}

type {{.Name | unCapitalize}}Cache struct {
	indexer  cache.Indexer
	resource schema.GroupResource
}

func (c *{{.Name | unCapitalize}}Cache) Get({{ if .namespaced}}namespace, {{end}}name string) (*{{.Version}}.{{.Name}}, error) {
	obj, exists, err := c.indexer.GetByKey({{ if .namespaced }}namespace + "/" + {{end}}name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.NewNotFound(c.resource, name)
	}
	return obj.(*{{.Version}}.{{.Name}}), nil
}

func (c *{{.Name | unCapitalize}}Cache) List({{ if .namespaced}}namespace string, {{end}}selector labels.Selector) (ret []*{{.Version}}.{{.Name}}, err error) {
	{{ if .namespaced }}
	err = cache.ListAllByNamespace(c.indexer, namespace, selector, func(m interface{}) {
		ret = append(ret, m.(*{{.Version}}.{{.Name}}))
	})
	{{else}}
	err = cache.ListAll(c.indexer, selector, func(m interface{}) {
		ret = append(ret, m.(*{{.Version}}.{{.Name}}))
	})
	{{end}}
	return ret, err
}

func (c *{{.Name | unCapitalize}}Cache) AddIndexer(indexName string, indexer {{.Name}}Indexer) {
	utilruntime.Must(c.indexer.AddIndexers(map[string]cache.IndexFunc{
		indexName: func(obj interface{}) (strings []string, e error) {
			return indexer(obj.(*{{.Version}}.{{.Name}}))
		},
	}))
}

func (c *{{.Name | unCapitalize}}Cache) GetByIndex(indexName, key string) (result []*{{.Version}}.{{.Name}}, err error) {
	objs, err := c.indexer.ByIndex(indexName, key)
	if err != nil {
		return nil, err
	}
	result = make([]*{{.Version}}.{{.Name}}, 0, len(objs))
	for _, obj := range objs {
		result = append(result, obj.(*{{.Version}}.{{.Name}}))
	}
	return result, nil
}

{{ if .hasStatus -}}
type {{.Name}}StatusHandler func(obj *{{.Version}}.{{.Name}}, status {{.Version}}.{{.statusType}}) ({{.Version}}.{{.statusType}}, error)

type {{.Name}}GeneratingHandler func(obj *{{.Version}}.{{.Name}}, status {{.Version}}.{{.statusType}}) ([]runtime.Object, {{.Version}}.{{.statusType}}, error)

func Register{{.Name}}StatusHandler(ctx context.Context, controller {{.Name}}Controller, condition condition.Cond, name string, handler {{.Name}}StatusHandler) {
	statusHandler := &{{.Name | unCapitalize}}StatusHandler{
		client:    controller,
		condition: condition,
		handler:   handler,
	}
	controller.AddGenericHandler(ctx, name, From{{.Name}}HandlerToHandler(statusHandler.sync))
}

func Register{{.Name}}GeneratingHandler(ctx context.Context, controller {{.Name}}Controller, apply apply.Apply,
	condition condition.Cond, name string, handler {{.Name}}GeneratingHandler, opts *generic.GeneratingHandlerOptions) {
	statusHandler := &{{.Name | unCapitalize}}GeneratingHandler{
		{{.Name}}GeneratingHandler: handler,
		apply:                            apply,
		name:                             name,
		gvk:                              controller.GroupVersionKind(),
	}
	if opts != nil {
		statusHandler.opts = *opts
	}
	controller.OnChange(ctx, name, statusHandler.Remove)
	Register{{.Name}}StatusHandler(ctx, controller, condition, name, statusHandler.Handle)
}

type {{.Name | unCapitalize}}StatusHandler struct {
	client    {{.Name}}Client
	condition condition.Cond
	handler   {{.Name}}StatusHandler
}

func (a *{{.Name | unCapitalize}}StatusHandler) sync(key string, obj *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error) {
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
		var newErr error
		obj.Status = newStatus
		obj, newErr = a.client.UpdateStatus(obj)
		if err == nil {
			err = newErr
		}
	}
	return obj, err
}

type {{.Name | unCapitalize}}GeneratingHandler struct {
	{{.Name}}GeneratingHandler
	apply apply.Apply
	opts  generic.GeneratingHandlerOptions
	gvk   schema.GroupVersionKind
	name  string
}

func (a *{{.Name | unCapitalize}}GeneratingHandler) Remove(key string, obj *{{.Version}}.{{.Name}}) (*{{.Version}}.{{.Name}}, error) {
	if obj != nil {
		return obj, nil
	}

	obj = &{{.Version}}.{{.Name}}{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(a.gvk)

	return nil, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects()
}

func (a *{{.Name | unCapitalize}}GeneratingHandler) Handle(obj *{{.Version}}.{{.Name}}, status {{.Version}}.{{.statusType}}) ({{.Version}}.{{.statusType}}, error) {
	objs, newStatus, err := a.{{.Name}}GeneratingHandler(obj, status)
	if err != nil {
		return newStatus, err
	}

	return newStatus, generic.ConfigureApplyForObject(a.apply, obj, &a.opts).
		WithOwner(obj).
		WithSetID(a.name).
		ApplyObjects(objs...)
}
{{- end }}`
