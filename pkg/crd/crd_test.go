package crd

// Mocks generated with
// mockgen --build_flags=--mod=mod -package crd -destination ./mockCRDClient_test.go "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1" CustomResourceDefinitionInterface

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

var errTest = errors.New("test Error")

func TestBatchCreateCRDs(t *testing.T) {
	t.Parallel()
	// decrease the ready wait duration for tests
	waitDuration := time.Second * 5

	// create 3 CRDs no status clone and update status in setup
	crd1 := &apiextv1.CustomResourceDefinition{}
	crd1.Name = "crd1s.testGroup"
	crd1.Spec.Group = "testGroup"
	crd1.Spec.Names.Plural = "crd1s"
	crd1.Spec.Names.Kind = "CRD1"
	crd1.Spec.Scope = apiextv1.ClusterScoped
	crd1.Spec.Versions = []apiextv1.CustomResourceDefinitionVersion{{Name: "v1"}}

	crd2 := &apiextv1.CustomResourceDefinition{}
	crd2.Name = "crd2s.testGroup"
	crd2.Spec.Group = "testGroup"
	crd2.Spec.Names.Plural = "crd2s"
	crd2.Spec.Names.Kind = "CRD2"
	crd2.Spec.Scope = apiextv1.ClusterScoped
	crd2.Spec.Versions = []apiextv1.CustomResourceDefinitionVersion{{Name: "v1"}}

	crd3 := &apiextv1.CustomResourceDefinition{}
	crd3.Name = "crd3s.testGroup"
	crd3.Spec.Group = "testGroup"
	crd3.Spec.Names.Plural = "crd3s"
	crd3.Spec.Names.Kind = "CRD3"
	crd3.Spec.Scope = apiextv1.ClusterScoped
	crd3.Spec.Versions = []apiextv1.CustomResourceDefinitionVersion{{Name: "v1"}}

	tests := []struct {
		name         string
		toCreateCRDs []*apiextv1.CustomResourceDefinition
		selector     labels.Selector
		wantErr      bool
		setupMock    func(*MockCustomResourceDefinitionInterface)
	}{
		{
			name:         "create single CRD",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1},
			selector:     labels.Nothing(),
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{LabelSelector: labels.Nothing().String()}).Return(list, nil)

				mock.EXPECT().Create(gomock.Any(), crd1, gomock.Any())

				readyCRD := crd1.DeepCopy()
				readyCRD.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				list2 := &apiextv1.CustomResourceDefinitionList{Items: []apiextv1.CustomResourceDefinition{*readyCRD}}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
			},
		},
		{
			name:         "create multiple CRDs",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1, crd2, crd3},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				// initial list should be allowed to fail.
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(nil, errTest)

				mock.EXPECT().Create(gomock.Any(), crd1, gomock.Any())
				mock.EXPECT().Create(gomock.Any(), crd2, gomock.Any())
				mock.EXPECT().Create(gomock.Any(), crd3, gomock.Any())

				readyCRD1 := crd1.DeepCopy()
				readyCRD1.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD2 := crd2.DeepCopy()
				readyCRD2.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD3 := crd3.DeepCopy()
				readyCRD3.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}

				list2 := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*readyCRD1, *readyCRD2, *readyCRD3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
			},
		},
		{
			name:         "create already exist CRD",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Create(gomock.Any(), crd1, gomock.Any()).Return(nil, apierrors.NewAlreadyExists(apiextv1.Resource("customeresourcedefinitions"), crd1.Name))
				mock.EXPECT().Get(gomock.Any(), crd1.Name, gomock.Any()).Return(crd1, nil)
				mock.EXPECT().Update(gomock.Any(), crd1, gomock.Any())

				readyCRD := crd1.DeepCopy()
				readyCRD.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				list2 := &apiextv1.CustomResourceDefinitionList{Items: []apiextv1.CustomResourceDefinition{*readyCRD}}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
			},
		},
		{
			name:         "create failed",
			wantErr:      true,
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Create(gomock.Any(), crd1, gomock.Any()).Return(nil, errTest)
			},
		},
		{
			name:         "update CRD",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{Items: []apiextv1.CustomResourceDefinition{*crd1}}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Update(gomock.Any(), crd1, gomock.Any())

				readyCRD := crd1.DeepCopy()
				readyCRD.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				list2 := &apiextv1.CustomResourceDefinitionList{Items: []apiextv1.CustomResourceDefinition{*readyCRD}}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
			},
		},
		{
			name:         "update Failed",
			wantErr:      true,
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{Items: []apiextv1.CustomResourceDefinition{*crd1}}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Update(gomock.Any(), crd1, gomock.Any()).Return(nil, errTest)
			},
		},
		{
			name:         "update multiple CRDs",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1, crd2, crd3},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*crd1, *crd2, *crd3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Update(gomock.Any(), crd1, gomock.Any())
				mock.EXPECT().Update(gomock.Any(), crd2, gomock.Any())
				mock.EXPECT().Update(gomock.Any(), crd3, gomock.Any())

				readyCRD1 := crd1.DeepCopy()
				readyCRD1.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD2 := crd2.DeepCopy()
				readyCRD2.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD3 := crd3.DeepCopy()
				readyCRD3.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}

				list2 := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*readyCRD1, *readyCRD2, *readyCRD3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
			},
		},
		{
			name:         "create and update Multiple CRDs",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1, crd2, crd3},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*crd2, *crd3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Create(gomock.Any(), crd1, gomock.Any())
				mock.EXPECT().Update(gomock.Any(), crd2, gomock.Any())
				mock.EXPECT().Update(gomock.Any(), crd3, gomock.Any())

				readyCRD1 := crd1.DeepCopy()
				readyCRD1.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD2 := crd2.DeepCopy()
				readyCRD2.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD3 := crd3.DeepCopy()
				readyCRD3.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}

				list2 := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*readyCRD1, *readyCRD2, *readyCRD3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
			},
		},
		{
			name:         "wait for Multiple CRDs",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1, crd2, crd3},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				readyCRD1 := crd1.DeepCopy()
				readyCRD1.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD2 := crd2.DeepCopy()
				readyCRD2.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				readyCRD3 := crd3.DeepCopy()
				readyCRD3.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionTrue},
				}
				list := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*readyCRD1, *readyCRD2, *readyCRD3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)

				mock.EXPECT().Update(gomock.Any(), crd1, gomock.Any())
				mock.EXPECT().Update(gomock.Any(), crd2, gomock.Any())
				mock.EXPECT().Update(gomock.Any(), crd3, gomock.Any())

				notReadyCRD1 := crd1.DeepCopy()
				notReadyCRD1.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionFalse},
				}
				notReadyCRD2 := crd2.DeepCopy()
				notReadyCRD2.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionFalse},
				}
				notReadyCRD3 := crd3.DeepCopy()
				notReadyCRD3.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionFalse},
				}

				list2 := &apiextv1.CustomResourceDefinitionList{
					Items: []apiextv1.CustomResourceDefinition{*readyCRD1, *readyCRD2, *readyCRD3},
				}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)

				mock.EXPECT().Get(gomock.Any(), crd1.Name, gomock.Any()).Return(readyCRD1, nil).AnyTimes()
				mock.EXPECT().Get(gomock.Any(), crd1.Name, gomock.Any()).Return(readyCRD2, nil).AnyTimes()
				mock.EXPECT().Get(gomock.Any(), crd1.Name, gomock.Any()).Return(readyCRD3, nil).AnyTimes()
			},
		},
		{
			name:         "wait for CRD that doesn't resolve",
			toCreateCRDs: []*apiextv1.CustomResourceDefinition{crd1},
			setupMock: func(mock *MockCustomResourceDefinitionInterface) {
				list := &apiextv1.CustomResourceDefinitionList{}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list, nil)
				mock.EXPECT().Create(gomock.Any(), crd1, gomock.Any())
				notReadyCRD := crd1.DeepCopy()
				notReadyCRD.Status.Conditions = []apiextv1.CustomResourceDefinitionCondition{
					{Type: apiextv1.Established, Status: apiextv1.ConditionFalse},
				}
				list2 := &apiextv1.CustomResourceDefinitionList{Items: []apiextv1.CustomResourceDefinition{*notReadyCRD}}
				mock.EXPECT().List(gomock.Any(), metav1.ListOptions{}).Return(list2, nil)
				mock.EXPECT().Get(gomock.Any(), crd1.Name, gomock.Any()).Return(notReadyCRD, nil).AnyTimes()
			},
			wantErr: true,
		},
	}
	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			mock := NewMockCustomResourceDefinitionInterface(ctrl)
			if tt.setupMock != nil {
				tt.setupMock(mock)
			}
			if err := BatchCreateCRDs(context.Background(), mock, tt.selector, waitDuration, tt.toCreateCRDs); (err != nil) != tt.wantErr {
				t.Errorf("BatchCreateCRDs() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
