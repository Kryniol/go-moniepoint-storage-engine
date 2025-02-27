// Code generated by mockery v2.52.2. DO NOT EDIT.

package domain

import (
	context "context"

	domain "github.com/Kryniol/go-moniepoint-storage-engine/domain"
	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MockIndex is an autogenerated mock type for the Index type
type MockIndex struct {
	mock.Mock
}

// Add provides a mock function with given fields: ctx, entry
func (_m *MockIndex) Add(ctx context.Context, entry domain.IndexEntry) error {
	ret := _m.Called(ctx, entry)

	if len(ret) == 0 {
		panic("no return value specified for Add")
	}

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, domain.IndexEntry) error); ok {
		r0 = rf(ctx, entry)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Find provides a mock function with given fields: ctx, key
func (_m *MockIndex) Find(ctx context.Context, key string) (*domain.IndexEntry, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for Find")
	}

	var r0 *domain.IndexEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) (*domain.IndexEntry, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) *domain.IndexEntry); ok {
		r0 = rf(ctx, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.IndexEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindFirst provides a mock function with given fields: ctx
func (_m *MockIndex) FindFirst(ctx context.Context) (*domain.IndexEntry, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for FindFirst")
	}

	var r0 *domain.IndexEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*domain.IndexEntry, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *domain.IndexEntry); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.IndexEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindLast provides a mock function with given fields: ctx
func (_m *MockIndex) FindLast(ctx context.Context) (*domain.IndexEntry, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for FindLast")
	}

	var r0 *domain.IndexEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (*domain.IndexEntry, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) *domain.IndexEntry); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(*domain.IndexEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindRange provides a mock function with given fields: ctx, startKey, endKey
func (_m *MockIndex) FindRange(ctx context.Context, startKey string, endKey string) (map[string]domain.IndexEntry, error) {
	ret := _m.Called(ctx, startKey, endKey)

	if len(ret) == 0 {
		panic("no return value specified for FindRange")
	}

	var r0 map[string]domain.IndexEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (map[string]domain.IndexEntry, error)); ok {
		return rf(ctx, startKey, endKey)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) map[string]domain.IndexEntry); ok {
		r0 = rf(ctx, startKey, endKey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]domain.IndexEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, startKey, endKey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// FindSince provides a mock function with given fields: ctx, since
func (_m *MockIndex) FindSince(ctx context.Context, since time.Time) (map[string]domain.IndexEntry, error) {
	ret := _m.Called(ctx, since)

	if len(ret) == 0 {
		panic("no return value specified for FindSince")
	}

	var r0 map[string]domain.IndexEntry
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) (map[string]domain.IndexEntry, error)); ok {
		return rf(ctx, since)
	}
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) map[string]domain.IndexEntry); ok {
		r0 = rf(ctx, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string]domain.IndexEntry)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, time.Time) error); ok {
		r1 = rf(ctx, since)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockIndex creates a new instance of MockIndex. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockIndex(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockIndex {
	mock := &MockIndex{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
