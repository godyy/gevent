package gevent

// EventID 事件ID，用于唯一标识一个事
// EventKind 事件类型
// EventValue 事件值
type EventID[EventKind, EventValue comparable] struct {
	Kind  EventKind
	Value EventValue
}

// Event 产生的事件
type Event[EventKind, EventValue comparable] struct {
	eventID   EventID[EventKind, EventValue] // 事件ID
	param     interface{}                    // 参数
	generator interface{}                    // The generator of the event
}

func (e *Event[EventKind, EventValue]) EventID() EventID[EventKind, EventValue] { return e.eventID }

func (e *Event[EventKind, EventValue]) Param() interface{} { return e.param }

func (e *Event[EventKind, EventValue]) Generator() interface{} { return e.generator }
