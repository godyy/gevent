package gevent

import (
	"errors"
	"fmt"
	"strings"
)

// dispatchError 用于封装派发事件时监听者返回的错误
// 记录派发的类别、派发时的事件ID
type dispatchError[EventKind, EventValue comparable] struct {
	dt      string                         // 派发的类别
	eventID EventID[EventKind, EventValue] // 事件ID
	err     error                          // error
}

func (e *dispatchError[EventKind, EventValue]) Error() string {
	return fmt.Sprintf("dispatch %s event of id={kind:%v, value:%v}: %v", e.dt, e.eventID.Value, e.eventID.Value, e.err.Error())
}

func (e *dispatchError[EventKind, EventValue]) Is(o error) bool {
	return errors.Is(e.err, o)
}

func (e *dispatchError[EventKind, EventValue]) As(o interface{}) bool {
	return errors.As(e.err, o)
}

// dispatchErrors 用于统计派发事件时监听者们返回的所有错误
type dispatchErrors struct {
	errors []error
}

func (e *dispatchErrors) Error() string {
	sb := strings.Builder{}
	sb.WriteString("[")
	for i, err := range e.errors {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(err.Error())
	}
	sb.WriteString("]")
	return sb.String()
}

func (e *dispatchErrors) Is(o error) bool {
	for _, err := range e.errors {
		if errors.Is(err, o) {
			return true
		}
	}
	return false
}

func (e *dispatchErrors) As(o interface{}) bool {
	for _, err := range e.errors {
		if errors.As(err, o) {
			return true
		}
	}
	return false
}
