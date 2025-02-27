// Code generated by mockery v2.52.2. DO NOT EDIT.

package domain

import (
	context "context"

	mock "github.com/stretchr/testify/mock"

	time "time"
)

// MockReadStorage is an autogenerated mock type for the ReadStorage type
type MockReadStorage struct {
	mock.Mock
}

// Get provides a mock function with given fields: ctx, key
func (_m *MockReadStorage) Get(ctx context.Context, key string) ([]byte, error) {
	ret := _m.Called(ctx, key)

	if len(ret) == 0 {
		panic("no return value specified for Get")
	}

	var r0 []byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string) ([]byte, error)); ok {
		return rf(ctx, key)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string) []byte); ok {
		r0 = rf(ctx, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).([]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string) error); ok {
		r1 = rf(ctx, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetAll provides a mock function with given fields: ctx
func (_m *MockReadStorage) GetAll(ctx context.Context) (map[string][]byte, error) {
	ret := _m.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for GetAll")
	}

	var r0 map[string][]byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context) (map[string][]byte, error)); ok {
		return rf(ctx)
	}
	if rf, ok := ret.Get(0).(func(context.Context) map[string][]byte); ok {
		r0 = rf(ctx)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context) error); ok {
		r1 = rf(ctx)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetRange provides a mock function with given fields: ctx, startKey, endKey
func (_m *MockReadStorage) GetRange(ctx context.Context, startKey string, endKey string) (map[string][]byte, error) {
	ret := _m.Called(ctx, startKey, endKey)

	if len(ret) == 0 {
		panic("no return value specified for GetRange")
	}

	var r0 map[string][]byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) (map[string][]byte, error)); ok {
		return rf(ctx, startKey, endKey)
	}
	if rf, ok := ret.Get(0).(func(context.Context, string, string) map[string][]byte); ok {
		r0 = rf(ctx, startKey, endKey)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, startKey, endKey)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// GetSince provides a mock function with given fields: ctx, since
func (_m *MockReadStorage) GetSince(ctx context.Context, since time.Time) (map[string][]byte, error) {
	ret := _m.Called(ctx, since)

	if len(ret) == 0 {
		panic("no return value specified for GetSince")
	}

	var r0 map[string][]byte
	var r1 error
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) (map[string][]byte, error)); ok {
		return rf(ctx, since)
	}
	if rf, ok := ret.Get(0).(func(context.Context, time.Time) map[string][]byte); ok {
		r0 = rf(ctx, since)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(map[string][]byte)
		}
	}

	if rf, ok := ret.Get(1).(func(context.Context, time.Time) error); ok {
		r1 = rf(ctx, since)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// NewMockReadStorage creates a new instance of MockReadStorage. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewMockReadStorage(t interface {
	mock.TestingT
	Cleanup(func())
}) *MockReadStorage {
	mock := &MockReadStorage{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}
