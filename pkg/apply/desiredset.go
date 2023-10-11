package apply

import (
	"context"

	"github.com/rancher/wrangler/v2/pkg/apply/injectors"
	"github.com/rancher/wrangler/v2/pkg/kv"
	"github.com/rancher/wrangler/v2/pkg/merr"
	"github.com/rancher/wrangler/v2/pkg/objectset"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/tools/cache"
)

type patchKey struct {
	schema.GroupVersionKind
	objectset.ObjectKey
}

type desiredSet struct {
	a                        *apply
	ctx                      context.Context
	defaultNamespace         string
	listerNamespace          string
	ignorePreviousApplied    bool
	setOwnerReference        bool
	ownerReferenceController bool
	ownerReferenceBlock      bool
	strictCaching            bool
	restrictClusterScoped    bool
	pruneTypes               map[schema.GroupVersionKind]cache.SharedIndexInformer
	patchers                 map[schema.GroupVersionKind]Patcher
	reconcilers              map[schema.GroupVersionKind]Reconciler
	diffPatches              map[patchKey][][]byte
	informerFactory          InformerFactory
	remove                   bool
	noDelete                 bool
	noDeleteGVK              map[schema.GroupVersionKind]struct{}
	setID                    string
	objs                     *objectset.ObjectSet
	owner                    runtime.Object
	injectors                []injectors.ConfigInjector
	ratelimitingQps          float32
	injectorNames            []string
	errs                     []error

	createPlan bool
	plan       Plan

	fastApply       bool
	replacingFields []string
}

func (o *desiredSet) err(err error) error {
	o.errs = append(o.errs, err)
	return o.Err()
}

func (o desiredSet) Err() error {
	return merr.NewErrors(append(o.errs, o.objs.Err())...)
}

func (o desiredSet) DryRun(objs ...runtime.Object) (Plan, error) {
	o.objs = objectset.NewObjectSet()
	o.objs.Add(objs...)
	return o.dryRun()
}

func (o desiredSet) Apply(set *objectset.ObjectSet) error {
	if set == nil {
		set = objectset.NewObjectSet()
	}
	o.objs = set
	return o.apply()
}

func (o desiredSet) ApplyObjects(objs ...runtime.Object) error {
	os := objectset.NewObjectSet()
	os.Add(objs...)
	return o.Apply(os)
}

func (o desiredSet) WithDiffPatch(gvk schema.GroupVersionKind, namespace, name string, patch []byte) Apply {
	patches := map[patchKey][][]byte{}
	for k, v := range o.diffPatches {
		patches[k] = v
	}
	key := patchKey{
		GroupVersionKind: gvk,
		ObjectKey: objectset.ObjectKey{
			Name:      name,
			Namespace: namespace,
		},
	}
	patches[key] = append(patches[key], patch)
	o.diffPatches = patches
	return o
}

// WithGVK uses a known listing of existing gvks to modify the the prune types to allow for deletion of objects
func (o desiredSet) WithGVK(gvks ...schema.GroupVersionKind) Apply {
	pruneTypes := make(map[schema.GroupVersionKind]cache.SharedIndexInformer, len(gvks))
	for k, v := range o.pruneTypes {
		pruneTypes[k] = v
	}
	for _, gvk := range gvks {
		pruneTypes[gvk] = nil
	}
	o.pruneTypes = pruneTypes
	return o
}

func (o desiredSet) WithSetID(id string) Apply {
	o.setID = id
	return o
}

func (o desiredSet) WithOwnerKey(key string, gvk schema.GroupVersionKind) Apply {
	obj := &v1.PartialObjectMetadata{}
	obj.Namespace, obj.Name = kv.RSplit(key, "/")
	obj.SetGroupVersionKind(gvk)
	o.owner = obj
	return o
}

func (o desiredSet) WithOwner(obj runtime.Object) Apply {
	o.owner = obj
	return o
}

func (o desiredSet) WithSetOwnerReference(controller, block bool) Apply {
	o.setOwnerReference = true
	o.ownerReferenceController = controller
	o.ownerReferenceBlock = block
	return o
}

func (o desiredSet) WithInjector(injs ...injectors.ConfigInjector) Apply {
	o.injectors = append(o.injectors, injs...)
	return o
}

func (o desiredSet) WithInjectorName(injs ...string) Apply {
	o.injectorNames = append(o.injectorNames, injs...)
	return o
}

func (o desiredSet) WithCacheTypeFactory(factory InformerFactory) Apply {
	o.informerFactory = factory
	return o
}

func (o desiredSet) WithIgnorePreviousApplied() Apply {
	o.ignorePreviousApplied = true
	return o
}

func (o desiredSet) WithCacheTypes(igs ...InformerGetter) Apply {
	pruneTypes := make(map[schema.GroupVersionKind]cache.SharedIndexInformer, len(igs))
	for k, v := range o.pruneTypes {
		pruneTypes[k] = v
	}

	for _, ig := range igs {
		pruneTypes[ig.GroupVersionKind()] = ig.Informer()
	}

	o.pruneTypes = pruneTypes
	return o
}

func (o desiredSet) WithPatcher(gvk schema.GroupVersionKind, patcher Patcher) Apply {
	patchers := map[schema.GroupVersionKind]Patcher{}
	for k, v := range o.patchers {
		patchers[k] = v
	}
	patchers[gvk] = patcher
	o.patchers = patchers
	return o
}

func (o desiredSet) WithReconciler(gvk schema.GroupVersionKind, reconciler Reconciler) Apply {
	reconcilers := map[schema.GroupVersionKind]Reconciler{}
	for k, v := range o.reconcilers {
		reconcilers[k] = v
	}
	reconcilers[gvk] = reconciler
	o.reconcilers = reconcilers
	return o
}

func (o desiredSet) WithStrictCaching() Apply {
	o.strictCaching = true
	return o
}
func (o desiredSet) WithDynamicLookup() Apply {
	o.strictCaching = false
	return o
}

func (o desiredSet) WithRestrictClusterScoped() Apply {
	o.restrictClusterScoped = true
	return o
}

func (o desiredSet) WithDefaultNamespace(ns string) Apply {
	if ns == "" {
		o.defaultNamespace = defaultNamespace
	} else {
		o.defaultNamespace = ns
	}
	return o
}

func (o desiredSet) WithListerNamespace(ns string) Apply {
	o.listerNamespace = ns
	return o
}

func (o desiredSet) WithRateLimiting(ratelimitingQps float32) Apply {
	o.ratelimitingQps = ratelimitingQps
	return o
}

func (o desiredSet) WithNoDelete() Apply {
	o.noDelete = true
	return o
}

func (o desiredSet) WithNoDeleteGVK(gvks ...schema.GroupVersionKind) Apply {
	if o.noDeleteGVK == nil {
		o.noDeleteGVK = make(map[schema.GroupVersionKind]struct{})
	}
	for _, curr := range gvks {
		o.noDeleteGVK[curr] = struct{}{}
	}
	return o
}

func (o desiredSet) WithContext(ctx context.Context) Apply {
	o.ctx = ctx
	return o
}

// WithFastApply configures Apply to use 2-way merging, which is less CPU intensive than the default 3-way merging.
// WithFastApply has the same behavior as Apply (including honoring strategic merge patch tags) apart from one case:
//
// When applying an object with a nil field, WithFastApply will ignore the corresponding field in the cluster object.
// If clearing of the corresponding field is wanted, specify its path in dotted notation in replacingFields and
// WithFastApply will generate a deletion patch for that field on nil.
//
// Note that basic Apply decides whether to clear such a field or not depending on the **immediately previous** Apply.
// If the field set nil was also nil in the previous Apply, then it is not cleared, otherwise it is. Additionally,
// basic Apply is not guaranteed to work as expected in all cases in presence of nil fields because of limitations of
// the underlying algorithms (JSON Merge Patch, Strategic Merge Patch). For more details see desiredset_compare_test.go
func (o desiredSet) WithFastApply(replacingFields ...string) Apply {
	o.fastApply = true
	o.replacingFields = replacingFields
	return o
}
