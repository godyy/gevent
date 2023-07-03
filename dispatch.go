package gevent

// Dispatcher 事件派发器
// 用于为特定类型或特定值类型的事件添加监听者，并在产生事件时将事件派发给监听者
type Dispatcher[EventKind, EventValue, ListenerID comparable] struct {
	kindListenerContainers map[EventKind]*kindListenerContainer[EventKind, EventValue, ListenerID] // 按照事件类型划分的监听者容器
	dispatching            int                                                                     // 派发状态计数
}

func NewDispatcher[EventKind, EventValue, ListenerID comparable]() *Dispatcher[EventKind, EventValue, ListenerID] {
	return &Dispatcher[EventKind, EventValue, ListenerID]{
		kindListenerContainers: map[EventKind]*kindListenerContainer[EventKind, EventValue, ListenerID]{},
	}
}

// AddKindListener 添加事件类型监听者
func (d *Dispatcher[EventKind, EventValue, ListenerID]) AddKindListener(evtKind EventKind, lID ListenerID, callback ListenerCallback[EventKind, EventValue], once ...bool) bool {
	if callback == nil {
		panic("listener callback nil")
	}
	once_ := false
	if len(once) > 0 {
		once_ = once[0]
	}
	l := newListener(lID, callback, once_)
	klc := d.addORGetKindListeners(evtKind)
	return klc.addKindListener(l)
}

// AddValueListener 添加值类型监听者
func (d *Dispatcher[EventKind, EventValue, ListenerID]) AddValueListener(evtId EventID[EventKind, EventValue], lID ListenerID, callback ListenerCallback[EventKind, EventValue], once ...bool) bool {
	if callback == nil {
		panic("listener callback nil")
	}
	once_ := false
	if len(once) > 0 {
		once_ = once[0]
	}
	l := newListener(lID, callback, once_)
	klc := d.addORGetKindListeners(evtId.Kind)
	return klc.addValueListener(evtId.Value, l)
}

// RemKindListener 移除事件类型监听者
func (d *Dispatcher[EventKind, EventValue, ListenerID]) RemKindListener(evtKind EventKind, lID ListenerID) bool {
	klc := d.kindListenerContainers[evtKind]
	if klc == nil {
		return false
	}
	rem := klc.remKindListener(lID)
	if klc.noListener() {
		delete(d.kindListenerContainers, evtKind)
	}
	return rem
}

// RemValueListener 移除值类型监听者
func (d *Dispatcher[EventKind, EventValue, ListenerID]) RemValueListener(evtId EventID[EventKind, EventValue], lID ListenerID) bool {
	klc := d.kindListenerContainers[evtId.Kind]
	if klc == nil {
		return false
	}
	rem := klc.remValueListener(evtId.Value, lID)
	if klc.noListener() {
		delete(d.kindListenerContainers, evtId.Kind)
	}
	return rem
}

// Clear 清理状态，移除所有监听者
func (d *Dispatcher[EventKind, EventValue, ListenerID]) Clear() {
	if d.dispatching > 0 {
		// 派发过程中不能清理
		return
	}

	for _, v := range d.kindListenerContainers {
		v.clear()
	}
	d.kindListenerContainers = map[EventKind]*kindListenerContainer[EventKind, EventValue, ListenerID]{}
}

// Dispatch 构造事件，派发给 evtID 指定的监听者们
func (d *Dispatcher[EventKind, EventValue, ListenerID]) Dispatch(evtId EventID[EventKind, EventValue], generator interface{}, param ...interface{}) error {
	klc := d.kindListenerContainers[evtId.Kind]
	if klc == nil {
		return nil
	}
	d.dispatching++
	evt := Event[EventKind, EventValue]{
		eventID:   evtId,
		generator: generator,
	}
	if len(param) > 0 {
		evt.param = param[0]
	}
	err := klc.dispatch(evt)
	if klc.noListener() {
		delete(d.kindListenerContainers, evtId.Kind)
	}
	d.dispatching--
	return err
}

func (d *Dispatcher[EventKind, EventValue, ListenerID]) addORGetKindListeners(evtKind EventKind) *kindListenerContainer[EventKind, EventValue, ListenerID] {
	klc := d.kindListenerContainers[evtKind]
	if klc == nil {
		klc = newKindListenerContainer[EventKind, EventValue, ListenerID]()
		d.kindListenerContainers[evtKind] = klc
	}
	return klc
}
