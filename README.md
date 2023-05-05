# Wrangler

Most people writing controllers are a bit lost as they find that there is nothing in Kubernetes that is like `type Controller interface` where you can just do `NewController`.  Instead a controller is really just a pattern of how you use the generated clientsets, informers, and listers combined with some custom event handlers and a workqueue.

Wrangler is a framework for using controllers. Controllers wrap clients, informers, listers into a simple usable controller pattern that promotes some good practices.

<br>

## Some Projects that use Wrangler
[rancher](https://github.com/rancher/rancher)

[eks-operator](https://github.com/rancher/eks-operator)

[aks-operator](https://github.com/rancher/aks-operator)

[gke-operator](https://github.com/rancher/gke-operator)

## Versioning and Updates

Wrangler releases use [semantic versioning](https://semver.org/). New major releases are created for breaking changes, new minor releases are created for features, and patches are added for everything else.

The most recent Major.Minor.x release and any releases being used by the most recent patch version of a [supported rancher version](https://www.suse.com/lifecycle/#rancher) will be maintained. The most recent major will receive minor releases, along with patch releases on its most up to date minor release. Older Major.Minor.x releases still in use by rancher will receive security patches at minimum. Consequently, there will be 1-3 maintained releases of the form Major.Minor.x at a time. Currently maintained versions:

| Wrangler Version | Rancher Version | Update Level           |
| ---------------- | --------------- | ---------------------- |
| 1.0.x            | 2.6.x           | Security Fixes         |
| 1.1.x            | 2.7.x           | Bug and Security Fixes |

Wrangler releases are not from the default branch. Instead they are from branches with the naming pattern `release-MAJOR.MINOR`. The default branch (i.e. master) is where changes initially go. This includes bug fixes and new features. Bug fixes are cherry-picked to release branches to be included in patch releases. When it's time to create a new minor or major release, a new release branch is created from the default branch.

<br>

# Table of Contents
1. [How it Works](#How-it-works)
	1. [Useful Definitions](#useful-definitions)
2. [How to Use Wrangler](#how-to-use-wrangler)
	1. [How to Write and Register a Handler](#how-to-write-and-register-a-handler-to-a-controller)
		1. [Creating an Instance of a Controller](#creating-an-instance-of-a-controller)
	2. [How to Run Handlers](#how-to-run-handlers)
	3. [Different Ways of Interacting with Objects](#different-ways-of-interacting-with-objects)
	4. [A Look at Structures Used in Wrangler](#a-look-at-structures-used-in-wrangler)

<br>

# How it Works

Wrangler provides a code generator that will generate the clientset, informers, listers and
additionally generate a controller per resource type.  The interface to the controller can be seen in the [Looking at Structures Used in Wrangler](#a-look-at-structures-used-in-wrangler) section.

<br>

The controller interface along with other helpful structs, interfaces, and functions are provided by another project [lasso](https://github.com/rancher/lasso). Lasso ties together the aforementioned tools while wrangler leverages them in a user friendly way.

To use the controller to run custom code for Kubernetes resource types all one needs to do is register OnChange handlers and run the controller.  Also using the controller interface one can access the client and caches through a simple flat API.

A typical, non-wrangler Kubernetes application would most likely use an informer for a resource type to add an event handler. Instead, wrangler uses lasso to register each handler which then aggregates the handlers into one function that accepts an object for the controller's resource type and then runs that object through all the handlers. This function is then registered to the Kubernetes informer for that controller's respective resource type. This is done so that an object can run through the handlers in a serialized way. This allows each handler to receive the updated version of the object and avoid many conflicts that would otherwise occur if the handlers were not chained together in this fashion.

<br>

## Useful Definitions:
<dl>
	<dt>factory</dt>
	<dd>Factories manage controllers. Wrangler generates factories for each API group. Wrangler factories use lasso shared factories for caches and controllers underneath.
	The lasso factories do most of the heavy lifting but are more resource type agnostic. Wrangler wraps lasso's factories to provide resource type specific clients and controllers.
	When accessing a wrangler generated controller, a controller for that resource type is requested from a lasso factory. If the controller exists it will be returned. Otherwise, the lasso factory will create it, persist it, and return it. You can consult the [lasso](https://github.com/rancher/lasso) repository for more details on factories.</dd>
	<dt>informers</dt>
	<dd>Broadcasts events for a given resource type and can register handlers for those events.</dd>
	<dt>listers</dt>
	<dd>Sometimes referred to as a cache, uses informers to update a local list of objects for a certain resource type to avoid making requests to the K8s API.</dd>
	<dt>event handlers</dt>
	<dd>Functions that run when a particular event is applied to the resource type the event handler is assigned to.</dd>
	<dt>workqueue</dt>
	<dd>A queue of items to be processed. In this context a queue will usually be a queue of objects of a certain resource type waiting to be processed by all handlers assigned to that resource type.</dd>
</dl>
<br>

# How to Use Wrangler

Generate controllers for CRDs by using Run() from the controllergen package. This will look like the
following:

```golang
controllergen.Run(args.Options{
		OutputPackage: "github.com/rancher/rancher/pkg/generated",
		Boilerplate:   "scripts/boilerplate.go.txt",
		Groups: map[string]args.Group{
			"management.cattle.io": {
				PackageName: "management.cattle.io",
				Types: []interface{}{
					// All structs with an embedded ObjectMeta field will be picked up
					"./pkg/apis/management.cattle.io/v3",
					// ProjectCatalog and ClusterCatalog are named
					// explicitly here because they do not have an
					// ObjectMeta field in their struct. Instead
					// they embed type v3.Catalog{} which
					// is a valid object on its own and is generated
					// above.
					v3.ProjectCatalog{},
					v3.ClusterCatalog{},
				},
				GenerateTypes: true,
			},
			"ui.cattle.io": {
				PackageName: "ui.cattle.io",
				Types: []interface{}{
					"./pkg/apis/ui.cattle.io/v1",
				},
				GenerateTypes: true,
			},
		},
	})
```

For the structs to be used when generating controllers they must have the following comments above the structs (note the newline between the comment and struct so it is not rejected by linters):
```
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

```

Four types are shown below. This file would be located at
`"pkg/apis/management.cattle.io/v3"` relative to the project root
directory. The line passing the "./pkg/apis/management.cattle.io/v3"
path ensure that the Setting and Catalog controllers are generated.
The lines naming the ProjectCatalog and ClusterCatalog structs ensure
the respective controllers are generated since neither directly have
an ObjectMeta field.:

``` golang
import (
	"github.com/rancher/norman/types"
)

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Setting struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Value      string `json:"value" norman:"required"`
	Default    string `json:"default" norman:"nocreate,noupdate"`
	Customized bool   `json:"customized" norman:"nocreate,noupdate"`
	Source     string `json:"source" norman:"nocreate,noupdate,options=db|default|env"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ProjectCatalog struct {
	types.Namespaced

	Catalog     `json:",inline" mapstructure:",squash"`
	ProjectName string `json:"projectName,omitempty" norman:"type=reference[project]"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ClusterCatalog struct {
	types.Namespaced

	Catalog     `json:",inline" mapstructure:",squash"`
	ClusterName string `json:"clusterName,omitempty" norman:"required,type=reference[cluster]"`
}

// +genclient
// +genclient:nonNamespaced
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type Catalog struct {
	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the catalog. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Spec   CatalogSpec   `json:"spec"`
	Status CatalogStatus `json:"status"`
}
```

__Note:__ This is real code taken from [rancher](https://github.com/rancher/rancher) and may not run at the time of reading this. This is meant to provide an example of how one might begin to use wrangler.

<br>

### Creating an Instance of a Controller

Controllers are categorized by their API group and bundled into a struct called a factory. Functions to create factories are generated by the Run function discussed above. To run one of the functions that creates a factory, import the proper package from the output directory of the generated code.

```golang
import (
	"github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io"
	"k8s.io/client-go/rest"
)

func createFactory(config *rest.Config) {
	mgmt, err := management.NewFactoryFromConfig(restConfig)
	if err != nil {
		return nil, err
	}
}

// Running the functions Management() and V3(), which are the api group and version of the resource types I have generated in this example, is necessary
// to instantiate the controller factories for the group and version. User() instantiates the controller for the user resource. This
// can be done elsewhere, like when creating a struct but it must be done before the controller is run. Otherwise, the cache will
// not work. In this case we are registering a handler so we would have ended up using these methods by necessity, but if we wanted	
// to access a cache for another resource type in our handler then we also need to make sure it is instantiated in a similar fashion.
users := mgmt.Management().V3().User("")

```
## How to Write and Register a Handler to a Controller

Registering a handler means to assign a handler to a specific Kubernetes resource's controller. These handlers will then run when
the appropriate event occurs on an object of that controller's resource type.

This will be a continuation of our above example:

```golang
import (
	"context"

	"github.com/rancher/rancher/pkg/generated/controllers/management.cattle.io"
	"github.com/rancher/wrangler/pkg/generated/controllers/core"
	"k8s.io/client-go/rest"
)

mgmt, err := management.NewFactoryFromConfig(restConfig)
if err != nil {
	return nil, err
}

users := mgmt.Management().V3().User("")
// passing a namespace here is optional. If an empty string is passed then the client will look at
// all configmap objects from all namespaces
configmaps := core.Management().Core().Configmaps("examplenamespace")

syncHandler := func(id string, obj *v3.User) (*v3.User, error) {
	if obj == nil {
		return obj, nil
	}

	recordedNote := obj.Annotations != nil && obj.Annotations["wroteanoteaboutuser"] == "true"

	if recordedNote {
		// there already is a note, noop
		return obj, nil
	}

	// we are getting the "mainrecord" configmap from the configmap cache. The cache is maintained
	// locally and can try to fulfill requests without using the k8s api. This is much faster and
	// efficient, however it does not update immediately so it is possible that if the object
	// being requested was recently created that the cache will miss and return a not found error.
	// In this scenario you can either count on the handler reenqueueing and retrying or you can
	// just use the regular client.
	record, err := configmaps.Cache().Get("", "mainrecord")
	if err != nil {
		return obj, err
	}

	record.Data[obj.name] = "recorded"
	record, err = configmaps.Update(record)
	if err != nil {
		return obj, err
	}

	// This is done because obj is from the cache that is iterated over to run handlers and perform other tasks. If the subsequent
	// update fails then we will end up with an object on our cache that does not match the "truth" (how the object is in etcd).
	obj = obj.DeepCopy()

	if obj.Annotations == nil {
		obj.Anotations = make(map[string]string)
	}

	obj.Annotations["wroteanoteaboutuser"] = "true"

	// Here we are using the k8s client embedded onto the users controller to perform an update. This will go to the K8s API.
	return users.Update(obj)
}

users.OnChange(context.Background(), "user-example-annotate-note-handler", syncHandler)
```
### How to Run Handlers

Now that we have registered an OnChange handler, we can run it like so:
`mgmt.Start(context.Background(), 50)`

<br>

### Different Ways of Interacting With Objects
In the above example, two clients and one cache are being used to interact with objects. A client can
Create, Update, UpdateStatus, Delete, Get, Watch and Patch an object, or List and Watch objects of its respective resource type. A Cache can get an object or list the objects for its respective resource type and will try to get the data locally (from its cache) if possible. The client and cache are the most common ways to interact with an object using wrangler.

Another way to interact with objects is to use the Apply client. The apply client works similarly to applying yaml using kubectl. This has benefits such as not assuming the existence of an object like the Update method on a client does. Instead, you can apply a state and the object will be created if it does not exist already or be updated to match the passed desired state if the object does exist
already. Apply also allows the use of multiple Owner References in a way unique from the client- if any owner reference is deleted the object will be deleted.

<br>

## A Look at Structures Used in Wrangler
```golang
type FooController interface {
	FooClient

	// OnChange registers a handler that will run whenever an object of the matching resource type is created or updated. This function accepts a sync function specifically generated for the object type and then wraps the function in a function that is compatible with AddGenericHandler. It then uses AddGenericHandler to register the wrapped function.
	OnChange(ctx context.Context, name string, sync FooHandler)
	// OnRemove registers a handler that will run whenever an object of the matching resource type is removed. This function accepts a sync function specifically generated for the object type and then wraps the function in a function that is compatible with AddGenericRemoveHandler. It then uses AddGenericRemoveHandler to register the wrapped function.
	OnRemove(ctx context.Context, name string, sync FooHandler)
	// Enqueue will rerun all handlers registered to the object's type against the object
	Enqueue(namespace, name string)

	// Cache returns a locally maintained cache that can be used for get and list requests
	Cache() FooCache

	// Informer returns an informer for the resource type
	Informer() cache.SharedIndexInformer
	// GroupVersionKind returns the API group, version, and Kind of the resource type the controller is for
	GroupVersionKind() schema.GroupVersionKind

	// AddGenericHandler registers the handler function for the controller
	AddGenericHandler(ctx context.Context, name string, handler generic.Handler)
	// AddGenericRemoveHandler registers a handler that will happen when an object of the controller's resource type is removed
	AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler)
	// Updater returns a function that accepts a runtime.Object and asserts it as the controller's respective resource type struct. It then passes the object to the resource type's client's update function. This is mainly consumed internally by wrangler to implement other functionality.
	Updater() generic.Updater
}

type FooClient interface {
	// Create creates a new instance of resource type in kubernetes
	Create(*v1alpha1.Foo) (*v1alpha1.Foo, error)
	// Update updates the given object in kubernetes
	Update(*v1alpha1.Foo) (*v1alpha1.Foo, error)
	// Status of type's CRD must be a subresource for this method to be generated. Only updates
	// status and does not trigger OnChange handlers.
	UpdateStatus(*v1alpha1.Foo) (*v1alpha1.Foo, error)
	// Delete deletes the given object in kubernetes
	Delete(namespace, name string, options *metav1.DeleteOptions) error
	// Get gets the object of the given name and namespace in kubernetes
	Get(namespace, name string, options metav1.GetOptions) (*v1alpha1.Foo, error)
	// List lists all the objects matching the given namespace for the resource type
	List(namespace string, opts metav1.ListOptions) (*v1alpha1.FooList, error)
	// Watch returns a channel that will stream objects as they are created, removed, or updated
	Watch(namespace string, opts metav1.ListOptions) (watch.Interface, error)
	// Patch accepts a diff that can be applied to an existing object for the client's resource type. Depending on PatchType, which specifies the strategy
	// to be used when applying the diff, patch can also create a new object. See the following for more information: https://kubernetes.io/docs/tasks/manage-kubernetes-objects/update-api-object-kubectl-patch/, https://kubernetes.io/docs/reference/using-api/server-side-apply/.
	Patch(namespace, name string, pt types.PatchType, data []byte, subresources ...string) (result *v1alpha1.Foo, err error)
}

type FooCache interface {
	// Get gets the object of the given name and namespace from the local cache
	Get(namespace, name string) (*v1alpha1.Foo, error)
	// List lists all the objects matching the given namespace for the resource type from the cache
	List(namespace string, selector labels.Selector) ([]*v1alpha1.Foo, error)

	// AddIndexer is used to register a function will be used to organize objects in the cache. The indexer will return a string which indicates something about the object.
	AddIndexer(indexName string, indexer FooIndexer)
	// GetByIndex will search for objects that match the given key when the named indexer is applied to it
	GetByIndex(indexName, key string) ([]*v1alpha1.Foo, error)
}

type FooIndexer func(obj *v1alpha1.Foo) ([]string, error)

```
