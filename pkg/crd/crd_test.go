package crd

//go:generate mockgen --build_flags=--mod=mod -package crd -destination ./mockCRDClient_test.go "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/typed/apiextensions/v1" CustomResourceDefinitionInterface

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/validation"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/validation/field"
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

func TestCreateCRDWithColumns(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
		columns []apiextv1.CustomResourceColumnDefinition
		crd     func() CRD
	}{
		{
			name:    "Basic CRD with no printer column",
			wantErr: false,
			crd: func() CRD {
				type ExampleSpec struct {
					Source   string `json:"source,omitempty"`
					Checksum string `json:"checksum,omitempty"`
				}

				type Example struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata,omitempty"`

					Spec ExampleSpec `json:"spec,omitempty"`
				}

				example := Example{}
				return NamespacedType("Example.example.com/v1").WithSchemaFromStruct(example).WithColumnsFromStruct(example)
			},
		},
		{
			name:    "Basic CRD with single printer column",
			wantErr: false,
			columns: []apiextv1.CustomResourceColumnDefinition{
				{Name: "Source", Type: "string", Format: "", Description: "", Priority: 0, JSONPath: ".spec.source"},
			},
			crd: func() CRD {
				type ExampleSpec struct {
					Source   string `json:"source,omitempty" column:""`
					Checksum string `json:"checksum,omitempty"`
				}

				type Example struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata,omitempty"`

					Spec ExampleSpec `json:"spec,omitempty"`
				}

				example := Example{}
				return NamespacedType("Example.example.com/v1").WithSchemaFromStruct(example).WithColumnsFromStruct(example)
			},
		},
		{
			name:    "Basic CRD with single printer column and custom name",
			wantErr: false,
			columns: []apiextv1.CustomResourceColumnDefinition{
				{Name: "ExampleSource", Type: "string", Format: "", Description: "", Priority: 0, JSONPath: ".spec.source"},
			},
			crd: func() CRD {
				type ExampleSpec struct {
					Source   string `json:"source,omitempty" column:"name=ExampleSource"`
					Checksum string `json:"checksum,omitempty"`
				}

				type Example struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata,omitempty"`

					Spec ExampleSpec `json:"spec,omitempty"`
				}

				example := Example{}
				return NamespacedType("Example.example.com/v1").WithSchemaFromStruct(example).WithColumnsFromStruct(example)
			},
		},
		{
			name:    "Basic CRD with struct field columns",
			wantErr: false,
			columns: []apiextv1.CustomResourceColumnDefinition{
				{Name: "Time", Type: "string", Format: "date-time", Description: "", Priority: 0, JSONPath: ".spec.time"},
				{Name: "Quantity", Type: "string", Format: "", Description: "", Priority: 0, JSONPath: ".spec.quantity"},
			},
			crd: func() CRD {
				type ExampleSpec struct {
					Time     *metav1.Time       `json:"time,omitempty" column:""`
					Quantity *resource.Quantity `json:"quantity,omitempty" column:""`
					Checksum string             `json:"checksum,omitempty"`
				}

				type Example struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata,omitempty"`

					Spec ExampleSpec `json:"spec,omitempty"`
				}

				example := Example{}
				return NamespacedType("Example.example.com/v1").WithSchemaFromStruct(example).WithColumnsFromStruct(example)
			},
		},
		{
			name:    "Complex CRD with mix of struct and basic field columns",
			wantErr: false,
			columns: []apiextv1.CustomResourceColumnDefinition{
				{Name: "Time", Type: "string", Format: "date-time", Description: "", Priority: 0, JSONPath: ".spec.time"},
				{Name: "Quantity", Type: "string", Format: "", Description: "", Priority: 0, JSONPath: ".spec.quantity"},
				{Name: "Byte", Type: "string", Format: "byte", Description: "", Priority: 0, JSONPath: ".status.checksum"},
				{Name: "Password", Type: "string", Format: "password", Description: "", Priority: 0, JSONPath: ".status.password"},
				{Name: "Boolean", Type: "boolean", Format: "", Description: "", Priority: 0, JSONPath: ".status.boolean"},
				{Name: "Float", Type: "number", Format: "", Description: "", Priority: 0, JSONPath: ".status.float"},
				{Name: "Integer", Type: "integer", Format: "", Description: "", Priority: 0, JSONPath: ".status.integer"},
				{Name: "IntOrString", Type: "string", Format: "", Description: "", Priority: 0, JSONPath: ".status.intOrString"},
			},
			crd: func() CRD {
				type ExampleSpec struct {
					Time     *metav1.Time       `json:"time,omitempty" column:""`
					Quantity *resource.Quantity `json:"quantity,omitempty" column:""`
				}

				type ExampleStatus struct {
					Byte        string              `json:"checksum,omitempty" column:"format=byte"`
					Password    string              `json:"password,omitempty" column:"format=password"`
					Boolean     *bool               `json:"boolean,omitempty" column:""`
					Float       *float32            `json:"float,omitempty" column:""`
					Integer     *int32              `json:"integer,omitempty" column:""`
					IntOrString *intstr.IntOrString `json:"intOrString,omitempty" column:""`
				}

				type Example struct {
					metav1.TypeMeta   `json:",inline"`
					metav1.ObjectMeta `json:"metadata,omitempty"`

					Spec   ExampleSpec   `json:"spec,omitempty"`
					Status ExampleStatus `json:"status,omitempty"`
				}

				example := Example{}
				return NamespacedType("Example.example.com/v1").WithSchemaFromStruct(example).WithColumnsFromStruct(example)
			},
		},
	}

	for i := range tests {
		tt := &tests[i]
		t.Run(tt.name, func(t *testing.T) {
			o, err := tt.crd().ToCustomResourceDefinition()
			if (err != nil) != tt.wantErr {
				t.Fatalf("ToCustomResourceDefinition() error = %v, wantErr %v", err, tt.wantErr)
			}
			unstructuredCRD, ok := o.(*unstructured.Unstructured)
			if !ok {
				t.Fatal("could not convert CRD runtime.Object to *unstructured.Unstructured")
			}
			var v1CRD *apiextv1.CustomResourceDefinition
			if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredCRD.UnstructuredContent(), &v1CRD); err != nil {
				t.Fatalf("Failed to convert CRD *unstructured.Unstructured to *apiextv1.CustomResourceDefinition: %v", err)
			}

			if len(v1CRD.Spec.Versions) == 0 {
				t.Errorf("CRD has no schema versions")
			}

			fldPath := field.NewPath("spec")
			for _, version := range v1CRD.Spec.Versions {
				for i := range version.AdditionalPrinterColumns {
					apc := &apiextensions.CustomResourceColumnDefinition{}
					if err := apiextv1.Convert_v1_CustomResourceColumnDefinition_To_apiextensions_CustomResourceColumnDefinition(&version.AdditionalPrinterColumns[i], apc, nil); err != nil {
						t.Errorf("Failed to convert apiextv1.CustomResourceColumnDefinition to apiextensions.CustomResourceColumnDefinition for validation: %v", err)
					}
					if errs := validation.ValidateCustomResourceColumnDefinition(apc, fldPath.Child("additionalPrinterColumns").Index(i)); len(errs) > 0 {
						t.Errorf("AdditionalPrinterColumn definition validation failed: %s", errs.ToAggregate().Error())
					}
				}

				if !reflect.DeepEqual(tt.columns, version.AdditionalPrinterColumns) {
					t.Errorf("AdditionalPrinterColumns = %#v,\n\t\twanted columns = %#v", version.AdditionalPrinterColumns, tt.columns)
				}
			}
		})
	}
}
