package generic

import "k8s.io/apimachinery/pkg/runtime"

// OnRemoveOption allows configuring OnRemove handlers
type OnRemoveOption func(*objectLifecycleAdapter)

// WithCondition will make OnRemove handlers to ignore resources not matching the provided condition
func WithCondition(f func(runtime.Object) bool) OnRemoveOption {
	return func(o *objectLifecycleAdapter) {
		o.condition = f
	}
}

func includeAll(_ runtime.Object) bool {
	return true
}
