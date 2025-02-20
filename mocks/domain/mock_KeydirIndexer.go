// Code generated by mockery v2.52.2. DO NOT EDIT.

package domain

import (
	context "context"

	domain "github.com/Kryniol/go-moniepoint-storage-engine/domain"
	mock "github.com/stretchr/testify/mock"
)

// MockKeydirIndexer is an autogenerated mock type for the KeydirIndexer type
type MockKeydirIndexer struct {
	mock.Mock
}

// Reindex provides a mock function with given fields: ctx
func (_m *MockKeydirIndexer) Reindex(ctx context.Context) ([]domain.IndexEntry, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Reindex")
	}

	var r0 []domain.IndexEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) ([]domain.IndexEntry, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) []domain.IndexEntry); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]domain.IndexEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockKeydirIndexer creates a new instance of MockKeydirIndexer. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockKeydirIndexer(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockKeydirIndexer {
	mock := &MockKeydirIndexer{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
