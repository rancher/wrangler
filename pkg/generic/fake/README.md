# Generic Controller, Client, and Cache Mocks

This package leverages https://github.com/golang/mock for using generic mocks in tests.
[gomock](https://pkg.go.dev/github.com/golang/mock/gomock) holds more specific information on different ways to use gomock in test.

## Usage
This package has four entry points for creating a mock controller, client, or cache interface.<br>
 Note: A mock controller will implement both a generic.ControllerInterface and a generic.ClientInterface.
- `NewMockControllerInterface[T runtime.Object, TList runtime.Object](*gomock.Controller)`
- `NewMockNonNamespacedControllerInterface[T runtime.Object, TList runtime.Object](*gomock.Controller)`
- `NewCacheInterface[T runtime.Object](*gomock.Controller)`
- `NewNonNamespaceCacheInterface[T runtime.Object](*gomock.Controller)`

## Examples

Example use of generic/fake with a generated Deployment Controller.
``` golang
// Generated controller interface to mock.
type DeploymentController interface {
	generic.ControllerInterface[*v1.Deployment, *v1.DeploymentList]
}
```
``` golang
// Example Test Function 
import (
	"testing"
    
	"github.com/golang/mock/gomock"
	wranglerv1 "github.com/rancher/wrangler/v3/pkg/generated/controllers/rbac/v1"
	"github.com/rancher/wrangler/v3/pkg/generic/fake"
	v1 "k8s.io/api/apps/v1"
)

func TestController(t *testing.T){
    // Create gomock controller. This is used by the gomock library.
	gomockCtrl := gomock.NewController(t)

    // Create a new Generic Controller Mock with type apps1.Deployment.
	deployMock := fake.NewMockControllerInterface[*v1.Deployment, *v1.DeploymentList](ctrl)

    // Define expected calls to our mock controller using gomock.
    deployMock.EXPECT().Enqueue("test-namespace", "test-name").AnyTimes()

    // Start Test Code.
    // .
    // . 
    // .

    // Test calls Enqueue with expected parameters nothing happens.
    deployMock.Enqueue("test-namespace", "test-name")

    // Test calls Enqueue with unexpected parameters.
    // gomock will fail the test because it did not expect the call.
    deployMock.Enqueue("unexpected-namespace", "unexpected-name")
}
```

### NonNamespacedController
```golang
ctrl := gomock.NewController(t)

mock := fake.NewMockNonNamespacedControllerInterface[*v3.RoleTemplate, *v3.RoleTemplateList](ctrl)

mock.EXPECT().List(gomock.Any()).Return(nil, nil)
```

## Fake Generation
This package was generated with `mockgen`, see [`generate.go`](./generate.go).

Due to incomplete support for generics, some modifications over the original file need to be applied for the generation
to succeeed.
#### `controller.go`
1. Comment out the `comparable` in RuntimeMetaObject