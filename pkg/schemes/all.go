package schemes

import (
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	// All is the scheme shared by every wrangler client and cache. Generated
	// packages register their types into it from their init functions, and it is
	// the scheme the lasso shared client factory falls back to when a caller
	// does not supply one of its own.
	All                = runtime.NewScheme()
	localSchemeBuilder = runtime.NewSchemeBuilder()
)

func Register(addToScheme func(*runtime.Scheme) error) error {
	localSchemeBuilder = append(localSchemeBuilder, addToScheme)
	return addToScheme(All)
}

func AddToScheme(scheme *runtime.Scheme) error {
	return localSchemeBuilder.AddToScheme(scheme)
}
