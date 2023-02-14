// Package fake is used to create github.com/golang/mock compatible mocks for generic client, caches, and controller interfaces.
package fake

import (
	"context"
	"reflect"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/rancher/lasso/pkg/client"
	"github.com/rancher/wrangler/pkg/generic"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
)

// MockControllerInterface is a mock of ControllerInterface interface.
type MockControllerInterface[T runtime.Object, TList runtime.Object] struct {
	ctrl     *gomock.Controller
	recorder *MockControllerInterfaceMockRecorder[T, TList]
}

// MockControllerInterfaceMockRecorder is the mock recorder for MockControllerInterface.
type MockControllerInterfaceMockRecorder[T runtime.Object, TList runtime.Object] struct {
	mock *MockControllerInterface[T, TList]
}

// NewMockControllerInterface creates a new mock instance.
func NewMockControllerInterface[T runtime.Object, TList runtime.Object](ctrl *gomock.Controller) *MockControllerInterface[T, TList] {
	mock := &MockControllerInterface[T, TList]{ctrl: ctrl}
	mock.recorder = &MockControllerInterfaceMockRecorder[T, TList]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockControllerInterface[T, TList]) EXPECT() *MockControllerInterfaceMockRecorder[T, TList] {
	return m.recorder
}

// AddGenericHandler mocks base method.
func (m *MockControllerInterface[T, TList]) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddGenericHandler", ctx, name, handler)
}

// AddGenericHandler indicates an expected call of AddGenericHandler.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) AddGenericHandler(ctx, name, handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGenericHandler", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).AddGenericHandler), ctx, name, handler)
}

// AddGenericRemoveHandler mocks base method.
func (m *MockControllerInterface[T, TList]) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddGenericRemoveHandler", ctx, name, handler)
}

// AddGenericRemoveHandler indicates an expected call of AddGenericRemoveHandler.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) AddGenericRemoveHandler(ctx, name, handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGenericRemoveHandler", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).AddGenericRemoveHandler), ctx, name, handler)
}

// Cache mocks base method.
func (m *MockControllerInterface[T, TList]) Cache() generic.CacheInterface[T] {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cache")
	ret0, _ := ret[0].(generic.CacheInterface[T])
	return ret0
}

// Cache indicates an expected call of Cache.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Cache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cache", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Cache))
}

// Create mocks base method.
func (m *MockControllerInterface[T, TList]) Create(arg0 T, arg1 client.CreateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Create), arg0, arg1)
}

// Delete mocks base method.
func (m *MockControllerInterface[T, TList]) Delete(namespace, name string, options client.DeleteOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", namespace, name, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Delete(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Delete), namespace, name, options)
}

// Enqueue mocks base method.
func (m *MockControllerInterface[T, TList]) Enqueue(namespace, name string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Enqueue", namespace, name)
}

// Enqueue indicates an expected call of Enqueue.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Enqueue(namespace, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enqueue", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Enqueue), namespace, name)
}

// EnqueueAfter mocks base method.
func (m *MockControllerInterface[T, TList]) EnqueueAfter(namespace, name string, duration time.Duration) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EnqueueAfter", namespace, name, duration)
}

// EnqueueAfter indicates an expected call of EnqueueAfter.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) EnqueueAfter(namespace, name, duration interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnqueueAfter", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).EnqueueAfter), namespace, name, duration)
}

// Get mocks base method.
func (m *MockControllerInterface[T, TList]) Get(namespace, name string, options client.GetOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", namespace, name, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Get(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Get), namespace, name, options)
}

// GroupVersionKind mocks base method.
func (m *MockControllerInterface[T, TList]) GroupVersionKind() schema.GroupVersionKind {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GroupVersionKind")
	ret0, _ := ret[0].(schema.GroupVersionKind)
	return ret0
}

// GroupVersionKind indicates an expected call of GroupVersionKind.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) GroupVersionKind() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GroupVersionKind", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).GroupVersionKind))
}

// Informer mocks base method.
func (m *MockControllerInterface[T, TList]) Informer() cache.SharedIndexInformer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Informer")
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	return ret0
}

// Informer indicates an expected call of Informer.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Informer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Informer", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Informer))
}

// List mocks base method.
func (m *MockControllerInterface[T, TList]) List(namespace string, opts client.ListOptions) (TList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", namespace, opts)
	ret0, _ := ret[0].(TList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) List(namespace, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).List), namespace, opts)
}

// OnChange mocks base method.
func (m *MockControllerInterface[T, TList]) OnChange(ctx context.Context, name string, sync generic.ObjectHandler[T]) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnChange", ctx, name, sync)
}

// OnChange indicates an expected call of OnChange.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) OnChange(ctx, name, sync interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnChange", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).OnChange), ctx, name, sync)
}

// OnRemove mocks base method.
func (m *MockControllerInterface[T, TList]) OnRemove(ctx context.Context, name string, sync generic.ObjectHandler[T]) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnRemove", ctx, name, sync)
}

// OnRemove indicates an expected call of OnRemove.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) OnRemove(ctx, name, sync interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnRemove", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).OnRemove), ctx, name, sync)
}

// Patch mocks base method.
func (m *MockControllerInterface[T, TList]) Patch(namespace, name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (T, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{namespace, name, pt, data, options}
	for _, a := range subresources {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Patch", varargs...)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Patch indicates an expected call of Patch.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Patch(namespace, name, pt, data, options interface{}, subresources ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{namespace, name, pt, data, options}, subresources...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Patch", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Patch), varargs...)
}

// Update mocks base method.
func (m *MockControllerInterface[T, TList]) Update(arg0 T, arg1 client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Update(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Update), arg0, arg1)
}

// UpdateStatus mocks base method.
func (m *MockControllerInterface[T, TList]) UpdateStatus(arg0 T, arg1 client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) UpdateStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).UpdateStatus), arg0, arg1)
}

// Updater mocks base method.
func (m *MockControllerInterface[T, TList]) Updater() generic.Updater {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Updater")
	ret0, _ := ret[0].(generic.Updater)
	return ret0
}

// Updater indicates an expected call of Updater.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Updater() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Updater", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Updater))
}

// Watch mocks base method.
func (m *MockControllerInterface[T, TList]) Watch(namespace string, opts client.ListOptions) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", namespace, opts)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockControllerInterfaceMockRecorder[T, TList]) Watch(namespace, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockControllerInterface[T, TList])(nil).Watch), namespace, opts)
}

// MockNonNamespacedControllerInterface is a mock of NonNamespacedControllerInterface interface.
type MockNonNamespacedControllerInterface[T runtime.Object, TList runtime.Object] struct {
	ctrl     *gomock.Controller
	recorder *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]
}

// MockNonNamespacedControllerInterfaceMockRecorder is the mock recorder for MockNonNamespacedControllerInterface.
type MockNonNamespacedControllerInterfaceMockRecorder[T runtime.Object, TList runtime.Object] struct {
	mock *MockNonNamespacedControllerInterface[T, TList]
}

// NewMockNonNamespacedControllerInterface creates a new mock instance.
func NewMockNonNamespacedControllerInterface[T runtime.Object, TList runtime.Object](ctrl *gomock.Controller) *MockNonNamespacedControllerInterface[T, TList] {
	mock := &MockNonNamespacedControllerInterface[T, TList]{ctrl: ctrl}
	mock.recorder = &MockNonNamespacedControllerInterfaceMockRecorder[T, TList]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNonNamespacedControllerInterface[T, TList]) EXPECT() *MockNonNamespacedControllerInterfaceMockRecorder[T, TList] {
	return m.recorder
}

// AddGenericHandler mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) AddGenericHandler(ctx context.Context, name string, handler generic.Handler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddGenericHandler", ctx, name, handler)
}

// AddGenericHandler indicates an expected call of AddGenericHandler.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) AddGenericHandler(ctx, name, handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGenericHandler", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).AddGenericHandler), ctx, name, handler)
}

// AddGenericRemoveHandler mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) AddGenericRemoveHandler(ctx context.Context, name string, handler generic.Handler) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddGenericRemoveHandler", ctx, name, handler)
}

// AddGenericRemoveHandler indicates an expected call of AddGenericRemoveHandler.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) AddGenericRemoveHandler(ctx, name, handler interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddGenericRemoveHandler", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).AddGenericRemoveHandler), ctx, name, handler)
}

// Cache mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Cache() generic.NonNamespacedCacheInterface[T] {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Cache")
	ret0, _ := ret[0].(generic.NonNamespacedCacheInterface[T])
	return ret0
}

// Cache indicates an expected call of Cache.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Cache() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Cache", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Cache))
}

// Create mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Create(arg0 T, arg1 client.CreateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Create), arg0, arg1)
}

// Delete mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Delete(name string, options client.DeleteOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", name, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Delete(name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Delete), name, options)
}

// Enqueue mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Enqueue(name string) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Enqueue", name)
}

// Enqueue indicates an expected call of Enqueue.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Enqueue(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Enqueue", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Enqueue), name)
}

// EnqueueAfter mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) EnqueueAfter(name string, duration time.Duration) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "EnqueueAfter", name, duration)
}

// EnqueueAfter indicates an expected call of EnqueueAfter.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) EnqueueAfter(name, duration interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EnqueueAfter", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).EnqueueAfter), name, duration)
}

// Get mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Get(name string, options client.GetOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", name, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Get(name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Get), name, options)
}

// GroupVersionKind mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) GroupVersionKind() schema.GroupVersionKind {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GroupVersionKind")
	ret0, _ := ret[0].(schema.GroupVersionKind)
	return ret0
}

// GroupVersionKind indicates an expected call of GroupVersionKind.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) GroupVersionKind() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GroupVersionKind", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).GroupVersionKind))
}

// Informer mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Informer() cache.SharedIndexInformer {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Informer")
	ret0, _ := ret[0].(cache.SharedIndexInformer)
	return ret0
}

// Informer indicates an expected call of Informer.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Informer() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Informer", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Informer))
}

// List mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) List(opts client.ListOptions) (TList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", opts)
	ret0, _ := ret[0].(TList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) List(opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).List), opts)
}

// OnChange mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) OnChange(ctx context.Context, name string, sync generic.ObjectHandler[T]) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnChange", ctx, name, sync)
}

// OnChange indicates an expected call of OnChange.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) OnChange(ctx, name, sync interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnChange", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).OnChange), ctx, name, sync)
}

// OnRemove mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) OnRemove(ctx context.Context, name string, sync generic.ObjectHandler[T]) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "OnRemove", ctx, name, sync)
}

// OnRemove indicates an expected call of OnRemove.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) OnRemove(ctx, name, sync interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "OnRemove", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).OnRemove), ctx, name, sync)
}

// Patch mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Patch(name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (T, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{name, pt, data, options}
	for _, a := range subresources {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Patch", varargs...)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Patch indicates an expected call of Patch.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Patch(name, pt, data, options interface{}, subresources ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{name, pt, data, options}, subresources...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Patch", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Patch), varargs...)
}

// Update mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Update(arg0 T, arg1 client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Update(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Update), arg0, arg1)
}

// UpdateStatus mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) UpdateStatus(arg0 T, arg1 client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) UpdateStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).UpdateStatus), arg0, arg1)
}

// Updater mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Updater() generic.Updater {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Updater")
	ret0, _ := ret[0].(generic.Updater)
	return ret0
}

// Updater indicates an expected call of Updater.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Updater() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Updater", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Updater))
}

// Watch mocks base method.
func (m *MockNonNamespacedControllerInterface[T, TList]) Watch(opts client.ListOptions) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", opts)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockNonNamespacedControllerInterfaceMockRecorder[T, TList]) Watch(opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockNonNamespacedControllerInterface[T, TList])(nil).Watch), opts)
}

// MockClientInterface is a mock of ClientInterface interface.
type MockClientInterface[T runtime.Object, TList runtime.Object] struct {
	ctrl     *gomock.Controller
	recorder *MockClientInterfaceMockRecorder[T, TList]
}

// MockClientInterfaceMockRecorder is the mock recorder for MockClientInterface.
type MockClientInterfaceMockRecorder[T runtime.Object, TList runtime.Object] struct {
	mock *MockClientInterface[T, TList]
}

// NewMockClientInterface creates a new mock instance.
func NewMockClientInterface[T runtime.Object, TList runtime.Object](ctrl *gomock.Controller) *MockClientInterface[T, TList] {
	mock := &MockClientInterface[T, TList]{ctrl: ctrl}
	mock.recorder = &MockClientInterfaceMockRecorder[T, TList]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockClientInterface[T, TList]) EXPECT() *MockClientInterfaceMockRecorder[T, TList] {
	return m.recorder
}

// Create mocks base method.
func (m *MockClientInterface[T, TList]) Create(arg0 T, arg1 client.CreateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, arg1)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockClientInterfaceMockRecorder[T, TList]) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockClientInterface[T, TList])(nil).Create), arg0, arg1)
}

// Delete mocks base method.
func (m *MockClientInterface[T, TList]) Delete(namespace, name string, options client.DeleteOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", namespace, name, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockClientInterfaceMockRecorder[T, TList]) Delete(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockClientInterface[T, TList])(nil).Delete), namespace, name, options)
}

// Get mocks base method.
func (m *MockClientInterface[T, TList]) Get(namespace, name string, options client.GetOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", namespace, name, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockClientInterfaceMockRecorder[T, TList]) Get(namespace, name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockClientInterface[T, TList])(nil).Get), namespace, name, options)
}

// List mocks base method.
func (m *MockClientInterface[T, TList]) List(namespace string, opts client.ListOptions) (TList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", namespace, opts)
	ret0, _ := ret[0].(TList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockClientInterfaceMockRecorder[T, TList]) List(namespace, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockClientInterface[T, TList])(nil).List), namespace, opts)
}

// Patch mocks base method.
func (m *MockClientInterface[T, TList]) Patch(namespace, name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (T, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{namespace, name, pt, data, options}
	for _, a := range subresources {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Patch", varargs...)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Patch indicates an expected call of Patch.
func (mr *MockClientInterfaceMockRecorder[T, TList]) Patch(namespace, name, pt, data, options interface{}, subresources ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{namespace, name, pt, data, options}, subresources...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Patch", reflect.TypeOf((*MockClientInterface[T, TList])(nil).Patch), varargs...)
}

// Update mocks base method.
func (m *MockClientInterface[T, TList]) Update(arg0 T, options client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockClientInterfaceMockRecorder[T, TList]) Update(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockClientInterface[T, TList])(nil).Update), arg0, arg1)
}

// UpdateStatus mocks base method.
func (m *MockClientInterface[T, TList]) UpdateStatus(arg0 T, options client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", arg0, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockClientInterfaceMockRecorder[T, TList]) UpdateStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockClientInterface[T, TList])(nil).UpdateStatus), arg0, arg1)
}

// Watch mocks base method.
func (m *MockClientInterface[T, TList]) Watch(namespace string, opts client.ListOptions) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", namespace, opts)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockClientInterfaceMockRecorder[T, TList]) Watch(namespace, opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockClientInterface[T, TList])(nil).Watch), namespace, opts)
}

// MockNonNamespacedClientInterface is a mock of NonNamespacedClientInterface interface.
type MockNonNamespacedClientInterface[T runtime.Object, TList runtime.Object] struct {
	ctrl     *gomock.Controller
	recorder *MockNonNamespacedClientInterfaceMockRecorder[T, TList]
}

// MockNonNamespacedClientInterfaceMockRecorder is the mock recorder for MockNonNamespacedClientInterface.
type MockNonNamespacedClientInterfaceMockRecorder[T runtime.Object, TList runtime.Object] struct {
	mock *MockNonNamespacedClientInterface[T, TList]
}

// NewMockNonNamespacedClientInterface creates a new mock instance.
func NewMockNonNamespacedClientInterface[T runtime.Object, TList runtime.Object](ctrl *gomock.Controller) *MockNonNamespacedClientInterface[T, TList] {
	mock := &MockNonNamespacedClientInterface[T, TList]{ctrl: ctrl}
	mock.recorder = &MockNonNamespacedClientInterfaceMockRecorder[T, TList]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNonNamespacedClientInterface[T, TList]) EXPECT() *MockNonNamespacedClientInterfaceMockRecorder[T, TList] {
	return m.recorder
}

// Create mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) Create(arg0 T, options client.CreateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", arg0, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) Create(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).Create), arg0, arg1)
}

// Delete mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) Delete(name string, options client.DeleteOptions) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", name, options)
	ret0, _ := ret[0].(error)
	return ret0
}

// Delete indicates an expected call of Delete.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) Delete(name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).Delete), name, options)
}

// Get mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) Get(name string, options client.GetOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", name, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) Get(name, options interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).Get), name, options)
}

// List mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) List(opts client.ListOptions) (TList, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", opts)
	ret0, _ := ret[0].(TList)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) List(opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).List), opts)
}

// Patch mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) Patch(name string, pt types.PatchType, data []byte, options client.PatchOptions, subresources ...string) (T, error) {
	m.ctrl.T.Helper()
	varargs := []interface{}{name, pt, data, options}
	for _, a := range subresources {
		varargs = append(varargs, a)
	}
	ret := m.ctrl.Call(m, "Patch", varargs...)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Patch indicates an expected call of Patch.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) Patch(name, pt, data, options interface{}, subresources ...interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	varargs := append([]interface{}{name, pt, data, options}, subresources...)
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Patch", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).Patch), varargs...)
}

// Update mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) Update(arg0 T, options client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Update", arg0, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Update indicates an expected call of Update.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) Update(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Update", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).Update), arg0, arg1)
}

// UpdateStatus mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) UpdateStatus(arg0 T, options client.UpdateOptions) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateStatus", arg0, options)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateStatus indicates an expected call of UpdateStatus.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) UpdateStatus(arg0, arg1 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateStatus", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).UpdateStatus), arg0, arg1)
}

// Watch mocks base method.
func (m *MockNonNamespacedClientInterface[T, TList]) Watch(opts client.ListOptions) (watch.Interface, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Watch", opts)
	ret0, _ := ret[0].(watch.Interface)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Watch indicates an expected call of Watch.
func (mr *MockNonNamespacedClientInterfaceMockRecorder[T, TList]) Watch(opts interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Watch", reflect.TypeOf((*MockNonNamespacedClientInterface[T, TList])(nil).Watch), opts)
}

// MockCacheInterface is a mock of CacheInterface interface.
type MockCacheInterface[T runtime.Object] struct {
	ctrl     *gomock.Controller
	recorder *MockCacheInterfaceMockRecorder[T]
}

// MockCacheInterfaceMockRecorder is the mock recorder for MockCacheInterface.
type MockCacheInterfaceMockRecorder[T runtime.Object] struct {
	mock *MockCacheInterface[T]
}

// NewMockCacheInterface creates a new mock instance.
func NewMockCacheInterface[T runtime.Object](ctrl *gomock.Controller) *MockCacheInterface[T] {
	mock := &MockCacheInterface[T]{ctrl: ctrl}
	mock.recorder = &MockCacheInterfaceMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCacheInterface[T]) EXPECT() *MockCacheInterfaceMockRecorder[T] {
	return m.recorder
}

// AddIndexer mocks base method.
func (m *MockCacheInterface[T]) AddIndexer(indexName string, indexer generic.Indexer[T]) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddIndexer", indexName, indexer)
}

// AddIndexer indicates an expected call of AddIndexer.
func (mr *MockCacheInterfaceMockRecorder[T]) AddIndexer(indexName, indexer interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddIndexer", reflect.TypeOf((*MockCacheInterface[T])(nil).AddIndexer), indexName, indexer)
}

// Get mocks base method.
func (m *MockCacheInterface[T]) Get(namespace, name string) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", namespace, name)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockCacheInterfaceMockRecorder[T]) Get(namespace, name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockCacheInterface[T])(nil).Get), namespace, name)
}

// GetByIndex mocks base method.
func (m *MockCacheInterface[T]) GetByIndex(indexName, key string) ([]T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIndex", indexName, key)
	ret0, _ := ret[0].([]T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIndex indicates an expected call of GetByIndex.
func (mr *MockCacheInterfaceMockRecorder[T]) GetByIndex(indexName, key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIndex", reflect.TypeOf((*MockCacheInterface[T])(nil).GetByIndex), indexName, key)
}

// List mocks base method.
func (m *MockCacheInterface[T]) List(namespace string, selector labels.Selector) ([]T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", namespace, selector)
	ret0, _ := ret[0].([]T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockCacheInterfaceMockRecorder[T]) List(namespace, selector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockCacheInterface[T])(nil).List), namespace, selector)
}

// MockNonNamespacedCacheInterface is a mock of NonNamespacedCacheInterface interface.
type MockNonNamespacedCacheInterface[T runtime.Object] struct {
	ctrl     *gomock.Controller
	recorder *MockNonNamespacedCacheInterfaceMockRecorder[T]
}

// MockNonNamespacedCacheInterfaceMockRecorder is the mock recorder for MockNonNamespacedCacheInterface.
type MockNonNamespacedCacheInterfaceMockRecorder[T runtime.Object] struct {
	mock *MockNonNamespacedCacheInterface[T]
}

// NewMockNonNamespacedCacheInterface creates a new mock instance.
func NewMockNonNamespacedCacheInterface[T runtime.Object](ctrl *gomock.Controller) *MockNonNamespacedCacheInterface[T] {
	mock := &MockNonNamespacedCacheInterface[T]{ctrl: ctrl}
	mock.recorder = &MockNonNamespacedCacheInterfaceMockRecorder[T]{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockNonNamespacedCacheInterface[T]) EXPECT() *MockNonNamespacedCacheInterfaceMockRecorder[T] {
	return m.recorder
}

// AddIndexer mocks base method.
func (m *MockNonNamespacedCacheInterface[T]) AddIndexer(indexName string, indexer generic.Indexer[T]) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "AddIndexer", indexName, indexer)
}

// AddIndexer indicates an expected call of AddIndexer.
func (mr *MockNonNamespacedCacheInterfaceMockRecorder[T]) AddIndexer(indexName, indexer interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddIndexer", reflect.TypeOf((*MockNonNamespacedCacheInterface[T])(nil).AddIndexer), indexName, indexer)
}

// Get mocks base method.
func (m *MockNonNamespacedCacheInterface[T]) Get(name string) (T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", name)
	ret0, _ := ret[0].(T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockNonNamespacedCacheInterfaceMockRecorder[T]) Get(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockNonNamespacedCacheInterface[T])(nil).Get), name)
}

// GetByIndex mocks base method.
func (m *MockNonNamespacedCacheInterface[T]) GetByIndex(indexName, key string) ([]T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetByIndex", indexName, key)
	ret0, _ := ret[0].([]T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetByIndex indicates an expected call of GetByIndex.
func (mr *MockNonNamespacedCacheInterfaceMockRecorder[T]) GetByIndex(indexName, key interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetByIndex", reflect.TypeOf((*MockNonNamespacedCacheInterface[T])(nil).GetByIndex), indexName, key)
}

// List mocks base method.
func (m *MockNonNamespacedCacheInterface[T]) List(selector labels.Selector) ([]T, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", selector)
	ret0, _ := ret[0].([]T)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockNonNamespacedCacheInterfaceMockRecorder[T]) List(selector interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockNonNamespacedCacheInterface[T])(nil).List), selector)
}
