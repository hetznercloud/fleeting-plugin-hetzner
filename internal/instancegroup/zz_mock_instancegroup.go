// Code generated by MockGen. DO NOT EDIT.
// Source: gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup (interfaces: InstanceGroup)
//
// Generated by this command:
//
//	mockgen -package instancegroup -destination zz_mock_instancegroup.go gitlab.com/hetznercloud/fleeting-plugin-hetzner/internal/instancegroup InstanceGroup
//

// Package instancegroup is a generated GoMock package.
package instancegroup

import (
	context "context"
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockInstanceGroup is a mock of InstanceGroup interface.
type MockInstanceGroup struct {
	ctrl     *gomock.Controller
	recorder *MockInstanceGroupMockRecorder
	isgomock struct{}
}

// MockInstanceGroupMockRecorder is the mock recorder for MockInstanceGroup.
type MockInstanceGroupMockRecorder struct {
	mock *MockInstanceGroup
}

// NewMockInstanceGroup creates a new mock instance.
func NewMockInstanceGroup(ctrl *gomock.Controller) *MockInstanceGroup {
	mock := &MockInstanceGroup{ctrl: ctrl}
	mock.recorder = &MockInstanceGroupMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockInstanceGroup) EXPECT() *MockInstanceGroupMockRecorder {
	return m.recorder
}

// Decrease mocks base method.
func (m *MockInstanceGroup) Decrease(ctx context.Context, iids []string) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Decrease", ctx, iids)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Decrease indicates an expected call of Decrease.
func (mr *MockInstanceGroupMockRecorder) Decrease(ctx, iids any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Decrease", reflect.TypeOf((*MockInstanceGroup)(nil).Decrease), ctx, iids)
}

// Get mocks base method.
func (m *MockInstanceGroup) Get(ctx context.Context, iid string) (*Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, iid)
	ret0, _ := ret[0].(*Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockInstanceGroupMockRecorder) Get(ctx, iid any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockInstanceGroup)(nil).Get), ctx, iid)
}

// Increase mocks base method.
func (m *MockInstanceGroup) Increase(ctx context.Context, delta int) ([]string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Increase", ctx, delta)
	ret0, _ := ret[0].([]string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Increase indicates an expected call of Increase.
func (mr *MockInstanceGroupMockRecorder) Increase(ctx, delta any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Increase", reflect.TypeOf((*MockInstanceGroup)(nil).Increase), ctx, delta)
}

// Init mocks base method.
func (m *MockInstanceGroup) Init(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Init", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Init indicates an expected call of Init.
func (mr *MockInstanceGroupMockRecorder) Init(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Init", reflect.TypeOf((*MockInstanceGroup)(nil).Init), ctx)
}

// List mocks base method.
func (m *MockInstanceGroup) List(ctx context.Context) ([]*Instance, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "List", ctx)
	ret0, _ := ret[0].([]*Instance)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// List indicates an expected call of List.
func (mr *MockInstanceGroupMockRecorder) List(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "List", reflect.TypeOf((*MockInstanceGroup)(nil).List), ctx)
}

// Sanity mocks base method.
func (m *MockInstanceGroup) Sanity(ctx context.Context) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Sanity", ctx)
	ret0, _ := ret[0].(error)
	return ret0
}

// Sanity indicates an expected call of Sanity.
func (mr *MockInstanceGroupMockRecorder) Sanity(ctx any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Sanity", reflect.TypeOf((*MockInstanceGroup)(nil).Sanity), ctx)
}
