// Mocks for this test are generated with the following command.
//go:generate sh -c "rm -f *Mocks_test.go"
//go:generate mockgen --build_flags=--mod=mod -package generic -destination ./controllerFactoryMocks_test.go github.com/rancher/lasso/pkg/controller SharedControllerFactory,SharedController
//go:generate mockgen -package generic -destination ./clientMocks_test.go -source ./embeddedClient.go

package generic

import (
	context "context"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

const (
	globalTestPodName   = "High-Noon-Harry"
	globalTestNamespace = "rodeo"
	globalTestNodeName  = "cowboy-server"
)

var errExpected = fmt.Errorf("test-error")

func TestController_Get(parentT *testing.T) {
	parentT.Parallel()
	testNamespace := globalTestNamespace
	var testController *Controller[*v1.Pod, *v1.PodList]
	var testNonNamespaceController *NonNamespacedController[*v1.Pod, *v1.PodList]
	test := func(t *testing.T) {
		testOptions := metav1.GetOptions{
			ResourceVersion: "3",
		}
		ctrl := gomock.NewController(t)
		mockClient := NewMockembeddedClient(ctrl)
		pod := &v1.Pod{}
		mockClient.EXPECT().Get(context.TODO(), testNamespace, globalTestPodName, gomock.AssignableToTypeOf(pod), testOptions).DoAndReturn(
			func(ctx context.Context, namespace string, name string, result runtime.Object, options metav1.GetOptions) error {
				resultPod, ok := result.(*v1.Pod)
				require.True(t, ok, "Created result object was the incorrect type.")
				resultPod.Spec.NodeName = globalTestNodeName
				return nil
			})
		var newPod *v1.Pod
		var err error
		if testNamespace == metav1.NamespaceAll {
			testNonNamespaceController = NewTestNonNamespacedController(ctrl, mockClient)
			newPod, err = testNonNamespaceController.Get(globalTestPodName, testOptions)
		} else {
			testController = NewTestController(ctrl, mockClient)
			newPod, err = testController.Get(testNamespace, globalTestPodName, testOptions)
		}
		require.NoError(t, err, "Error when calling get.")
		require.Equal(t, globalTestNodeName, newPod.Spec.NodeName, "Get call did not correctly persist pod changes from the embeddedClient.")

		mockClient.EXPECT().Get(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
		if testNamespace == metav1.NamespaceAll {
			_, err = testNonNamespaceController.Get(globalTestPodName, testOptions)
		} else {
			_, err = testController.Get(testNamespace, globalTestPodName, testOptions)
		}
		require.Error(t, err, "Error from client.Get() was not propagated")
	}
	parentT.Run("Namespaced", test)
	testNamespace = metav1.NamespaceAll
	parentT.Run("NonNamespaced", test)
}

func TestController_List(parentT *testing.T) {
	parentT.Parallel()
	testNamespace := globalTestNamespace
	var testController *Controller[*v1.Pod, *v1.PodList]
	var testNonNamespaceController *NonNamespacedController[*v1.Pod, *v1.PodList]
	test := func(t *testing.T) {
		testOptions := metav1.ListOptions{
			ResourceVersion: "3",
		}
		ctrl := gomock.NewController(t)
		mockClient := NewMockembeddedClient(ctrl)
		pod := &v1.PodList{}
		mockClient.EXPECT().List(context.TODO(), testNamespace, gomock.AssignableToTypeOf(pod), testOptions).DoAndReturn(
			func(ctx context.Context, namespace string, result runtime.Object, options metav1.ListOptions) error {
				pods, ok := result.(*v1.PodList)
				require.True(t, ok, "Created result object was the incorrect type.")
				pods.Items = []v1.Pod{{}}
				pods.Items[0].Spec.NodeName = globalTestNodeName
				return nil
			})
		var newPods *v1.PodList
		var err error
		if testNamespace == metav1.NamespaceAll {
			testNonNamespaceController = NewTestNonNamespacedController(ctrl, mockClient)
			newPods, err = testNonNamespaceController.List(testOptions)
		} else {
			testController = NewTestController(ctrl, mockClient)
			newPods, err = testController.List(testNamespace, testOptions)
		}
		require.NoError(t, err, "Error when calling list.")
		require.Equal(t, globalTestNodeName, newPods.Items[0].Spec.NodeName, "List call did not correctly persist pod changes from the embeddedClient")

		mockClient.EXPECT().List(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
		if testNamespace == metav1.NamespaceAll {
			_, err = testNonNamespaceController.List(testOptions)
		} else {
			_, err = testController.List(testNamespace, testOptions)
		}
		require.Error(t, err, "Error from client.List(...) was not propagated")
	}
	parentT.Run("Namespaced", test)
	testNamespace = metav1.NamespaceAll
	parentT.Run("NonNamespaced", test)
}
func TestController_Watch(parentT *testing.T) {
	parentT.Parallel()
	testNamespace := globalTestNamespace
	var testController *Controller[*v1.Pod, *v1.PodList]
	var testNonNamespaceController *NonNamespacedController[*v1.Pod, *v1.PodList]
	test := func(t *testing.T) {
		testOptions := metav1.ListOptions{
			ResourceVersion: "3",
		}

		ctrl := gomock.NewController(t)
		mockClient := NewMockembeddedClient(ctrl)
		emptyWatch := watch.NewEmptyWatch()
		mockClient.EXPECT().Watch(context.TODO(), testNamespace, testOptions).Return(emptyWatch, nil)

		var watchInterface watch.Interface
		var err error
		if testNamespace == metav1.NamespaceAll {
			testNonNamespaceController = NewTestNonNamespacedController(ctrl, mockClient)
			watchInterface, err = testNonNamespaceController.Watch(testOptions)
		} else {
			testController = NewTestController(ctrl, mockClient)
			watchInterface, err = testController.Watch(testNamespace, testOptions)
		}
		require.NoError(t, err, "Error when calling watch.")
		require.Equal(t, emptyWatch, watchInterface, "Watch call did not send the watch interface from the request")

		mockClient.EXPECT().Watch(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errExpected)
		if testNamespace == metav1.NamespaceAll {
			_, err = testNonNamespaceController.Watch(testOptions)
		} else {
			_, err = testController.Watch(testNamespace, testOptions)
		}
		require.Error(t, err, "Error from client.Watch(...) was not propagated")
	}
	parentT.Run("Namespaced", test)
	testNamespace = metav1.NamespaceAll
	parentT.Run("NonNamespaced", test)

}

func TestController_Patch(parentT *testing.T) {
	parentT.Parallel()
	testNamespace := globalTestNamespace
	var testController *Controller[*v1.Pod, *v1.PodList]
	var testNonNamespaceController *NonNamespacedController[*v1.Pod, *v1.PodList]
	test := func(t *testing.T) {
		testOptions := metav1.PatchOptions{}
		testPT := types.JSONPatchType
		testData := []byte(globalTestNodeName)
		subResources := []string{"sub", "resources"}
		ctrl := gomock.NewController(t)
		mockClient := NewMockembeddedClient(ctrl)
		pod := &v1.Pod{}
		mockClient.EXPECT().Patch(context.TODO(), testNamespace, globalTestPodName, testPT, testData, gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(testOptions), subResources).DoAndReturn(
			func(ctx context.Context, namespace string, name string, pt types.PatchType, data []byte, result runtime.Object, opts metav1.PatchOptions, subresources ...string) error {
				resultPod, ok := result.(*v1.Pod)
				require.True(t, ok, "Created result object was the incorrect type.")
				resultPod.Spec.NodeName = globalTestNodeName
				require.Equal(t, testOptions, opts, "Patch received unexpected patch options.")
				return nil
			})
		var newPod *v1.Pod
		var err error
		if testNamespace == metav1.NamespaceAll {
			testNonNamespaceController = NewTestNonNamespacedController(ctrl, mockClient)
			newPod, err = testNonNamespaceController.Patch(globalTestPodName, testPT, testData, subResources...)
		} else {
			testController = NewTestController(ctrl, mockClient)
			newPod, err = testController.Patch(testNamespace, globalTestPodName, testPT, testData, subResources...)
		}
		require.NoError(t, err, "Error when calling patch.")
		require.Equal(t, globalTestNodeName, newPod.Spec.NodeName, "Patch call did not correctly persist pod changes from the embeddedClient")

		mockClient.EXPECT().Patch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
		if testNamespace == metav1.NamespaceAll {
			_, err = testNonNamespaceController.Patch(globalTestPodName, testPT, testData, subResources...)
		} else {
			_, err = testController.Patch(testNamespace, globalTestPodName, testPT, testData, subResources...)
		}
		require.Error(t, err, "Error from client.Patch(...) was not propagated")
	}
	parentT.Run("Namespaced", test)
	testNamespace = metav1.NamespaceAll
	parentT.Run("NonNamespaced", test)

}

func TestController_Update(t *testing.T) {
	t.Parallel()
	testOptions := metav1.UpdateOptions{}
	ctrl := gomock.NewController(t)
	mockClient := NewMockembeddedClient(ctrl)
	pod := &v1.Pod{}
	mockClient.EXPECT().Update(context.TODO(), globalTestNamespace, gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(testOptions)).DoAndReturn(
		func(ctx context.Context, namespace string, obj runtime.Object, result runtime.Object, opts metav1.UpdateOptions) error {
			updatePod, ok := obj.(*v1.Pod)
			require.True(t, ok, "Obj to update is the incorrect type.")
			require.Equal(t, updatePod, pod, "Incorrect obj to update was sent to the client.")

			resultPod, ok := result.(*v1.Pod)
			require.True(t, ok, "Created result object was the incorrect type.")
			resultPod.Spec.NodeName = globalTestNodeName
			require.Equal(t, testOptions, opts, "Update received unexpected update options.")
			return nil
		})
	testController := NewTestController(ctrl, mockClient)
	pod.Namespace = globalTestNamespace
	newPod, err := testController.Update(pod)
	require.NoError(t, err, "Error when calling update.")
	require.Equal(t, globalTestNodeName, newPod.Spec.NodeName, "Update call did not correctly persist pod changes from the embeddedClient")

	mockClient.EXPECT().Update(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
	_, err = testController.Update(pod)
	require.Error(t, err, "Error from client.Update(...) was not propagated")
}

func TestController_UpdateStatus(t *testing.T) {
	t.Parallel()
	testOptions := metav1.UpdateOptions{}
	ctrl := gomock.NewController(t)
	mockClient := NewMockembeddedClient(ctrl)
	pod := &v1.Pod{}
	mockClient.EXPECT().UpdateStatus(context.TODO(), globalTestNamespace, gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(testOptions)).DoAndReturn(
		func(ctx context.Context, namespace string, obj runtime.Object, result runtime.Object, opts metav1.UpdateOptions) error {
			updatePod, ok := obj.(*v1.Pod)
			require.True(t, ok, "obj to updateStatus is the incorrect type")
			require.Equal(t, updatePod, pod, "incorrect obj to update was sent to the client")
			resultPod, ok := result.(*v1.Pod)
			require.True(t, ok, "Created result object was the incorrect type.")
			resultPod.Status.Reason = globalTestNodeName
			require.Equal(t, testOptions, opts, "UpdateStatus received unexpected update options.")
			return nil
		})
	testController := NewTestController(ctrl, mockClient)
	pod.Namespace = globalTestNamespace
	newPod, err := testController.UpdateStatus(pod)
	require.NoError(t, err, "Error when calling UpdateStatus(...).")
	require.Equal(t, globalTestNodeName, newPod.Status.Reason, "UpdateStatus call did not correctly persist pod changes from the embeddedClient")

	mockClient.EXPECT().UpdateStatus(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
	_, err = testController.UpdateStatus(pod)
	require.Error(t, err, "Error from client.UpdateStatus(...) was not propagated")
}

func TestController_Create(t *testing.T) {
	t.Parallel()
	testOptions := metav1.CreateOptions{}
	ctrl := gomock.NewController(t)
	mockClient := NewMockembeddedClient(ctrl)
	pod := &v1.Pod{}
	mockClient.EXPECT().Create(context.TODO(), globalTestNamespace, gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(pod), gomock.AssignableToTypeOf(testOptions)).DoAndReturn(
		func(ctx context.Context, namespace string, obj runtime.Object, result runtime.Object, opts metav1.CreateOptions) error {
			createPod, ok := obj.(*v1.Pod)
			require.True(t, ok, "obj to create is the incorrect type")
			require.Equal(t, createPod, pod)
			resultPod, ok := result.(*v1.Pod)
			require.True(t, ok, "Created result object was the incorrect type.")
			resultPod.Spec.NodeName = globalTestNodeName
			require.Equal(t, testOptions, opts, "Create received unexpected create options.")
			return nil
		})
	testController := NewTestController(ctrl, mockClient)
	pod.Namespace = globalTestNamespace
	newPod, err := testController.Create(pod)
	require.NoError(t, err, "Error when calling create.")
	require.Equal(t, globalTestNodeName, newPod.Spec.NodeName, "Create call did not correctly persist pod changes from the embeddedClient")

	mockClient.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
	_, err = testController.Create(pod)
	require.Error(t, err, "Error from client.Create(...) was not propagated")
}

func TestController_Delete(parentT *testing.T) {
	parentT.Parallel()
	testNamespace := globalTestNamespace
	var testController *Controller[*v1.Pod, *v1.PodList]
	var testNonNamespaceController *NonNamespacedController[*v1.Pod, *v1.PodList]
	test := func(t *testing.T) {
		testOptions := metav1.DeleteOptions{}
		ctrl := gomock.NewController(t)
		mockClient := NewMockembeddedClient(ctrl)
		mockClient.EXPECT().Delete(context.TODO(), testNamespace, globalTestPodName, testOptions).Return(nil)
		var err error
		if testNamespace == metav1.NamespaceAll {
			testNonNamespaceController = NewTestNonNamespacedController(ctrl, mockClient)
			err = testNonNamespaceController.Delete(globalTestPodName, &testOptions)
		} else {
			testController = NewTestController(ctrl, mockClient)
			err = testController.Delete(testNamespace, globalTestPodName, &testOptions)
		}
		require.NoError(t, err, "Error when calling delete.")

		mockClient.EXPECT().Delete(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errExpected)
		if testNamespace == metav1.NamespaceAll {
			err = testNonNamespaceController.Delete(globalTestPodName, &testOptions)
		} else {
			err = testController.Delete(testNamespace, globalTestPodName, &testOptions)
		}
		require.Error(t, err, "Error from client.Delete(...) was not propagated")
	}

	parentT.Run("Namespaced", test)
	testNamespace = metav1.NamespaceAll
	parentT.Run("NonNamespaced", test)
}

func NewTestController(ctrl *gomock.Controller, testClient embeddedClient) *Controller[*v1.Pod, *v1.PodList] {
	// create mock that allows the new function to run without panic
	mockFactory := NewMockSharedControllerFactory(ctrl)
	mockController := NewMockSharedController(ctrl)
	mockController.EXPECT().Client().Return(nil)
	mockFactory.EXPECT().ForResourceKind(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockController)
	newController := NewController[*v1.Pod, *v1.PodList](schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, "pods", true, mockFactory)
	// override the nil controller client with the test client
	newController.embeddedClient = testClient
	return newController
}

func NewTestNonNamespacedController(ctrl *gomock.Controller, testClient embeddedClient) *NonNamespacedController[*v1.Pod, *v1.PodList] {
	// create mock that allows the new function to run without panic
	mockFactory := NewMockSharedControllerFactory(ctrl)
	mockController := NewMockSharedController(ctrl)
	mockController.EXPECT().Client().Return(nil)
	mockFactory.EXPECT().ForResourceKind(gomock.Any(), gomock.Any(), gomock.Any()).Return(mockController)
	newController := NewNonNamespacedController[*v1.Pod, *v1.PodList](schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"}, "pods", mockFactory)
	// override the nil controller client with the test client
	newController.embeddedClient = testClient
	return newController
}
