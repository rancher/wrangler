package fake

import (
	"context"

	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/apply/injectors"
	"github.com/rancher/wrangler/pkg/objectset"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ apply.Apply = (*FakeApply)(nil)

type FakeApply struct {
	Objects []*objectset.ObjectSet
}

func (f *FakeApply) Apply(set *objectset.ObjectSet) error {
	f.Objects = append(f.Objects, set)
	return nil
}

func (f *FakeApply) ApplyObjects(objs ...runtime.Object) error {
	os := objectset.NewObjectSet()
	os.Add(objs...)
	f.Objects = append(f.Objects, os)
	return nil
}

func (f *FakeApply) WithCacheTypes(igs ...apply.InformerGetter) apply.Apply {
	return f
}

func (f *FakeApply) WithIgnorePreviousApplied() apply.Apply {
	return f
}

func (f *FakeApply) WithGVK(gvks ...schema.GroupVersionKind) apply.Apply {
	return f
}

func (f *FakeApply) WithSetID(id string) apply.Apply {
	return f
}

func (f *FakeApply) WithOwner(obj runtime.Object) apply.Apply {
	return f
}

func (f *FakeApply) WithInjector(injs ...injectors.ConfigInjector) apply.Apply {
	return f
}

func (f *FakeApply) WithInjectorName(injs ...string) apply.Apply {
	return f
}

func (f *FakeApply) WithPatcher(gvk schema.GroupVersionKind, patchers apply.Patcher) apply.Apply {
	return f
}

func (f *FakeApply) WithReconciler(gvk schema.GroupVersionKind, reconciler apply.Reconciler) apply.Apply {
	return f
}

func (f *FakeApply) WithStrictCaching() apply.Apply {
	return f
}

func (f *FakeApply) WithDynamicLookup() apply.Apply {
	return f
}

func (f *FakeApply) WithDefaultNamespace(ns string) apply.Apply {
	return f
}

func (f *FakeApply) WithListerNamespace(ns string) apply.Apply {
	return f
}

func (f *FakeApply) WithRestrictClusterScoped() apply.Apply {
	return f
}

func (f *FakeApply) WithSetOwnerReference(controller, block bool) apply.Apply {
	return f
}

func (f *FakeApply) WithRateLimiting(ratelimitingQps float32) apply.Apply {
	return f
}

func (f *FakeApply) WithNoDelete() apply.Apply {
	return f
}

func (f *FakeApply) WithContext(ctx context.Context) apply.Apply {
	return f
}

func (f *FakeApply) WithCacheTypeFactory(factory apply.InformerFactory) apply.Apply {
	return f
}

func (f *FakeApply) DryRun(objs ...runtime.Object) (apply.Plan, error) {
	return apply.Plan{}, nil
}

func (f *FakeApply) FindOwner(obj runtime.Object) (runtime.Object, error) {
	return nil, nil
}

func (f *FakeApply) PurgeOrphan(obj runtime.Object) error {
	return nil
}

func (f *FakeApply) WithOwnerKey(key string, gvk schema.GroupVersionKind) apply.Apply {
	return f
}
