package fake_test

import (
	"testing"

	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/generic/fake"
	v1 "k8s.io/api/core/v1"
)

// TestInterfaceImplementation is a simple test to verify the fake package is kept up to date.
// if this compiles it is valid.
func TestInterfaceImplementation(t *testing.T) {
	var (
		_ generic.ControllerInterface[*v1.Secret, *v1.SecretList]              = fake.NewMockControllerInterface[*v1.Secret, *v1.SecretList](nil)
		_ generic.NonNamespacedControllerInterface[*v1.Secret, *v1.SecretList] = fake.NewMockNonNamespacedControllerInterface[*v1.Secret, *v1.SecretList](nil)
		_ generic.ClientInterface[*v1.Secret, *v1.SecretList]                  = fake.NewMockClientInterface[*v1.Secret, *v1.SecretList](nil)
		_ generic.NonNamespacedClientInterface[*v1.Secret, *v1.SecretList]     = fake.NewMockNonNamespacedClientInterface[*v1.Secret, *v1.SecretList](nil)
		_ generic.CacheInterface[*v1.Secret]                                   = fake.NewMockCacheInterface[*v1.Secret](nil)
		_ generic.NonNamespacedCacheInterface[*v1.Secret]                      = fake.NewMockNonNamespacedCacheInterface[*v1.Secret](nil)
	)
}
