// Code generated by mockery; DO NOT EDIT.
// github.com/vektra/mockery
// template: testify

package mocks

import (
	"context"

	mock "github.com/stretchr/testify/mock"
)

// NewService creates a new instance of Service. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewService(t interface {
	mock.TestingT
	Cleanup(func())
}) *Service {
	mock := &Service{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Service is an autogenerated mock type for the Service type
type Service struct {
	mock.Mock
}

type Service_Expecter struct {
	mock *mock.Mock
}

func (_m *Service) EXPECT() *Service_Expecter {
	return &Service_Expecter{mock: &_m.Mock}
}

// Close provides a mock function for the type Service
func (_mock *Service) Close() {
	_mock.Called()
	return
}

// Service_Close_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Close'
type Service_Close_Call struct {
	*mock.Call
}

// Close is a helper method to define mock.On call
func (_e *Service_Expecter) Close() *Service_Close_Call {
	return &Service_Close_Call{Call: _e.mock.On("Close")}
}

func (_c *Service_Close_Call) Run(run func()) *Service_Close_Call {
	_c.Call.Run(func(args mock.Arguments) {
		run()
	})
	return _c
}

func (_c *Service_Close_Call) Return() *Service_Close_Call {
	_c.Call.Return()
	return _c
}

func (_c *Service_Close_Call) RunAndReturn(run func()) *Service_Close_Call {
	_c.Run(run)
	return _c
}

// Start provides a mock function for the type Service
func (_mock *Service) Start(ctx context.Context) error {
	ret := _mock.Called(ctx)

	if len(ret) == 0 {
		panic("no return value specified for Start")
	}

	var r0 error
	if returnFunc, ok := ret.Get(0).(func(context.Context) error); ok {
		r0 = returnFunc(ctx)
	} else {
		r0 = ret.Error(0)
	}
	return r0
}

// Service_Start_Call is a *mock.Call that shadows Run/Return methods with type explicit version for method 'Start'
type Service_Start_Call struct {
	*mock.Call
}

// Start is a helper method to define mock.On call
//   - ctx context.Context
func (_e *Service_Expecter) Start(ctx interface{}) *Service_Start_Call {
	return &Service_Start_Call{Call: _e.mock.On("Start", ctx)}
}

func (_c *Service_Start_Call) Run(run func(ctx context.Context)) *Service_Start_Call {
	_c.Call.Run(func(args mock.Arguments) {
		var arg0 context.Context
		if args[0] != nil {
			arg0 = args[0].(context.Context)
		}
		run(
			arg0,
		)
	})
	return _c
}

func (_c *Service_Start_Call) Return(err error) *Service_Start_Call {
	_c.Call.Return(err)
	return _c
}

func (_c *Service_Start_Call) RunAndReturn(run func(ctx context.Context) error) *Service_Start_Call {
	_c.Call.Return(run)
	return _c
}
