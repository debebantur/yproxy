// Code generated by MockGen. DO NOT EDIT.
// Source: pkg/backups/backups.go

// Package mock is a generated GoMock package.
package mock

import (
	reflect "reflect"

	gomock "github.com/golang/mock/gomock"
)

// MockBackupInterractor is a mock of BackupInterractor interface.
type MockBackupInterractor struct {
	ctrl     *gomock.Controller
	recorder *MockBackupInterractorMockRecorder
}

// MockBackupInterractorMockRecorder is the mock recorder for MockBackupInterractor.
type MockBackupInterractorMockRecorder struct {
	mock *MockBackupInterractor
}

// NewMockBackupInterractor creates a new mock instance.
func NewMockBackupInterractor(ctrl *gomock.Controller) *MockBackupInterractor {
	mock := &MockBackupInterractor{ctrl: ctrl}
	mock.recorder = &MockBackupInterractorMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockBackupInterractor) EXPECT() *MockBackupInterractorMockRecorder {
	return m.recorder
}

// GetFirstLSN mocks base method.
func (m *MockBackupInterractor) GetFirstLSN(arg0 int) (uint64, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetFirstLSN", arg0)
	ret0, _ := ret[0].(uint64)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetFirstLSN indicates an expected call of GetFirstLSN.
func (mr *MockBackupInterractorMockRecorder) GetFirstLSN(arg0 interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetFirstLSN", reflect.TypeOf((*MockBackupInterractor)(nil).GetFirstLSN), arg0)
}
