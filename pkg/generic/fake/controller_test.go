package fake_test

import (
	"testing"

	"github.com/rancher/wrangler/pkg/generic"
	"github.com/rancher/wrangler/pkg/generic/fake"
	v1 "k8s.io/api/apps/v1"
)

// Currently if this test compiles we satisfy the interface
func TestInterfaceSatisfaction(t *testing.T) {
	mock := &fake.MockControllerInterface[*v1.Deployment, *v1.DeploymentList]{}
	_ = generic.ControllerInterface[*v1.Deployment, *v1.DeploymentList](mock)

	mock2 := &fake.MockNonNamespacedControllerInterface[*v1.Deployment, *v1.DeploymentList]{}
	_ = generic.NonNamespacedControllerInterface[*v1.Deployment, *v1.DeploymentList](mock2)

	cache := &fake.MockCacheInterface[*v1.Deployment]{}
	_ = generic.CacheInterface[*v1.Deployment](cache)

	cache2 := &fake.MockNonNamespacedCacheInterface[*v1.Deployment]{}
	_ = generic.NonNamespacedCacheInterface[*v1.Deployment](cache2)

	client := &fake.MockClientInterface[*v1.Deployment, *v1.DeploymentList]{}
	_ = generic.ClientInterface[*v1.Deployment, *v1.DeploymentList](client)

	client2 := &fake.MockNonNamespacedClientInterface[*v1.Deployment, *v1.DeploymentList]{}
	_ = generic.NonNamespacedClientInterface[*v1.Deployment, *v1.DeploymentList](client2)
}
