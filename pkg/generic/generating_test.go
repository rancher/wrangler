package generic_test

import (
	"context"
	"github.com/rancher/wrangler/v2/pkg/generic"
	"testing"

	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"github.com/rancher/wrangler/v2/pkg/apply"
	fakeapply "github.com/rancher/wrangler/v2/pkg/apply/fake"
	v1 "github.com/rancher/wrangler/v2/pkg/generated/controllers/core/v1"
	fake2 "github.com/rancher/wrangler/v2/pkg/generic/fake"
)

func TestUniqueApplyForResourceVersion(t *testing.T) {
	const numOfHandlerCalls = 3
	type args struct {
		opts *generic.GeneratingHandlerOptions
	}
	tests := []struct {
		name               string
		args               args
		expectedApplyCount int
	}{
		{
			name: "disabled",
			args: args{
				opts: &generic.GeneratingHandlerOptions{UniqueApplyForResourceVersion: false},
			},
			expectedApplyCount: numOfHandlerCalls,
		},
		{
			name: "enabled",
			args: args{
				opts: &generic.GeneratingHandlerOptions{UniqueApplyForResourceVersion: true},
			},
			expectedApplyCount: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			applySpy := &fakeapply.FakeApply{}

			h := setupTestHandler(ctrl, applySpy, tt.args.opts)
			if h == nil {
				t.Fatal("error setting up test handler")
			}

			service := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{Name: "testsvc", Namespace: "testns", ResourceVersion: "unchanged"},
				Spec:       corev1.ServiceSpec{Ports: []corev1.ServicePort{{Name: "http", Protocol: "tcp", Port: 80}}},
				Status: corev1.ServiceStatus{
					Conditions: []metav1.Condition{},
				},
			}
			key := service.Namespace + "/" + service.Name
			for i := 0; i < numOfHandlerCalls; i++ {
				if _, err := h(key, service); err != nil {
					t.Error(err)
				}
			}
			if got, want := applySpy.Count, tt.expectedApplyCount; got != want {
				t.Errorf("Apply calls count = %v, want %v", got, want)
			}

			service.ResourceVersion = "changed"
			if _, err := h(key, service); err != nil {
				t.Error(err)
			}
			if got, want := applySpy.Count, tt.expectedApplyCount+1; got != want {
				t.Errorf("Apply calls count = %v, want %v", got, want)
			}
		})
	}
}

func setupTestHandler(ctrl *gomock.Controller, apply apply.Apply, opts *generic.GeneratingHandlerOptions) (handler generic.Handler) {
	const handlerName = "test"
	controller := fake2.NewMockControllerInterface[*corev1.Service, *corev1.ServiceList](ctrl)
	controller.EXPECT().GroupVersionKind()
	controller.EXPECT().OnChange(gomock.Any(), gomock.Any(), gomock.Any())
	controller.EXPECT().
		AddGenericHandler(gomock.Any(), gomock.Eq(handlerName), gomock.Any()).
		Do(func(_ context.Context, _ string, h generic.Handler) {
			handler = h
		}).Times(1)

	v1.RegisterServiceGeneratingHandler(context.Background(), controller, apply, "", handlerName,
		func(svc *corev1.Service, status corev1.ServiceStatus) (objs []runtime.Object, newstatus corev1.ServiceStatus, err error) {
			return []runtime.Object{serviceToEndpoint(svc)}, status, nil
		}, opts)
	return
}

func serviceToEndpoint(svc *corev1.Service) *corev1.Endpoints {
	var ports []corev1.EndpointPort
	for _, port := range svc.Spec.Ports {
		ports = append(ports, corev1.EndpointPort{
			Name: port.Name, Port: port.Port, Protocol: port.Protocol, AppProtocol: port.AppProtocol,
		})
	}
	return &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      svc.Name,
			Namespace: svc.Namespace,
		},
		Subsets: []corev1.EndpointSubset{
			{Ports: ports},
		},
	}
}
