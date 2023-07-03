package gevent

import (
	"container/list"
	"errors"
)

// ErrRemAfterDispatch 派发后移除
// 监听者专用，返回该 error 即告诉派发器，在本次派发完成后移除它
var ErrRemAfterDispatch = errors.New("remove after dispatch")

// ListenerCallback 监听者回调
type ListenerCallback[EventKind, EventValue comparable] func(Event[EventKind, EventValue]) error

// listener 监听者
// 记录监听者信息
type listener[EventKind, EventValue, ListenerID comparable] struct {
	id         ListenerID                              // 监听者ID
	callback   ListenerCallback[EventKind, EventValue] // 监听者回调
	once       bool                                    // 是否只监听一次
	pendingRem bool                                    // 挂起等待移除
}

func newListener[EventKind, EventValue, ListenerID comparable](id ListenerID, callback ListenerCallback[EventKind, EventValue], once bool) *listener[EventKind, EventValue, ListenerID] {
	if callback == nil {
		panic("callback nil")
	}
	return &listener[EventKind, EventValue, ListenerID]{
		id:         id,
		callback:   callback,
		once:       once,
		pendingRem: false,
	}
}

// dispatch 向监听者派发事件
// 返回监听者产生的错误
func (l *listener[EventKind, EventValue, ListenerID]) dispatch(evt Event[EventKind, EventValue]) error {
	if l.pendingRem {
		// 已处于挂起移除状态，不再接收事件
		return nil
	}
	return l.callback(evt)
}

// reset 重置数据，解除引用
func (l *listener[EventKind, EventValue, ListenerID]) reset() {
	l.callback = nil
}

// listenerContainer 监听者容器
// 每一个独立的事件，都有与之对应的监听者容器来维护相关的监听者
type listenerContainer[EventKind, EventValue, ListenerID comparable] struct {
	listenerList   *list.List                   // 监听者列表
	listenerMap    map[ListenerID]*list.Element // 监听者 Elem Map
	pendingRemList *list.List                   // 挂起移除列表，等待在事件派发完成后被移除的监听者
	dispatching    int                          // 派发状态计数
}

func newListenerContainer[EventKind, EventValue, ListenerID comparable]() *listenerContainer[EventKind, EventValue, ListenerID] {
	return &listenerContainer[EventKind, EventValue, ListenerID]{
		listenerList: list.New(),
		listenerMap:  map[ListenerID]*list.Element{},
	}
}

// addListener 添加监听者
// 不能重复添加相同ID的监听者
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) addListener(l *listener[EventKind, EventValue, ListenerID]) bool {
	if ls.dispatching > 0 {
		// 不能在派发事件状态下，嵌套添加监听者
		panic("add listener on dispatching")
	}
	if elem, ok := ls.listenerMap[l.id]; ok {
		return false
	} else {
		elem = ls.listenerList.PushBack(l)
		ls.listenerMap[l.id] = elem
		return true
	}
}

// remListener 移除监听者
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) remListener(lID ListenerID) bool {
	elem, ok := ls.listenerMap[lID]
	if !ok {
		return false
	}
	l := elem.Value.(*listener[EventKind, EventValue, ListenerID])
	if ls.dispatching > 0 {
		// 正在派发事件，所有需要移除的监听者都需要挂起，等待派发完成后，统一移除
		ls.pendingRemListener(l, elem)
	} else {
		// 未派发事件，直接移除
		ls.directRemListener(l, elem)
	}
	return true
}

// directRemListener 直接移除监听者
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) directRemListener(l *listener[EventKind, EventValue, ListenerID], elem *list.Element) {
	ls.listenerList.Remove(elem)
	delete(ls.listenerMap, l.id)
	l.reset()
}

// pendingRemListener 挂起移除监听者
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) pendingRemListener(l *listener[EventKind, EventValue, ListenerID], elem *list.Element) {
	if l.pendingRem {
		// 无须重复添加已经等待移除的监听者
		return
	}
	l.pendingRem = true
	if ls.pendingRemList == nil {
		ls.pendingRemList = list.New()
	}
	ls.pendingRemList.PushBack(elem)
}

// noListener 返回是否没有监听者
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) noListener() bool {
	return ls.listenerList.Len() == 0
}

// dispatch 向监听者们派发事件
// 返回监听者们产生的错误
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) dispatch(event Event[EventKind, EventValue]) error {
	ls.dispatching++

	var errs []error
	elem := ls.listenerList.Front()
	for elem != nil {
		l := elem.Value.(*listener[EventKind, EventValue, ListenerID])
		if !l.pendingRem {
			err := l.dispatch(event)
			if err != nil && err != ErrRemAfterDispatch {
				errs = append(errs, err)
			}
			if l.once || err == ErrRemAfterDispatch {
				ls.pendingRemListener(l, elem)
			}
		}
		elem = elem.Next()
	}

	ls.dispatching--
	if ls.dispatching < 0 {
		ls.dispatching = 0
	}

	if ls.dispatching == 0 && ls.pendingRemList != nil {
		elem := ls.pendingRemList.Front()
		for elem != nil {
			remElem := elem.Value.(*list.Element)
			ls.directRemListener(remElem.Value.(*listener[EventKind, EventValue, ListenerID]), remElem)
			next := elem.Next()
			ls.pendingRemList.Remove(elem)
			elem = next
		}
		ls.pendingRemList = nil
	}

	if len(errs) > 0 {
		return &dispatchErrors{errors: errs}
	}

	return nil
}

// clear 清理容器，移除所有监听者
func (ls *listenerContainer[EventKind, EventValue, ListenerID]) clear() {
	if ls.dispatching > 0 {
		// 派发过程中无法清理
		return
	}
	ls.listenerMap = nil
	if ls.pendingRemList != nil {
		ls.pendingRemList.Init()
		elem := ls.pendingRemList.Front()
		for elem != nil {
			elem.Value = nil
			rem := elem
			elem = elem.Next()
			ls.pendingRemList.Remove(rem)
		}
		ls.pendingRemList = nil
	}
	if ls.listenerList != nil {
		elem := ls.listenerList.Front()
		for elem != nil {
			elem.Value.(*listener[EventKind, EventValue, ListenerID]).reset()
			elem.Value = nil
			rem := elem
			elem = elem.Next()
			ls.listenerList.Remove(rem)
		}
		ls.listenerList = nil
	}
}

// kindListenerContainer 按事件类型划分的监听者容器
type kindListenerContainer[EventKind, EventValue, ListenerID comparable] struct {
	kindListeners  *listenerContainer[EventKind, EventValue, ListenerID]                // 类型事件监听者
	valueListeners map[EventValue]*listenerContainer[EventKind, EventValue, ListenerID] // 值类事件监听者
	dispatching    int                                                                  // 派发状态计数
}

func newKindListenerContainer[EventKind, EventValue, ListenerID comparable]() *kindListenerContainer[EventKind, EventValue, ListenerID] {
	return &kindListenerContainer[EventKind, EventValue, ListenerID]{}
}

// addKindListener 添加类型事件监听者
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) addKindListener(l *listener[EventKind, EventValue, ListenerID]) bool {
	if kls.dispatching > 0 {
		// 不能在派发事件状态下，嵌套添加监听者
		panic("add kind listener on dispatching")
	}
	if kls.kindListeners == nil {
		kls.kindListeners = newListenerContainer[EventKind, EventValue, ListenerID]()
	}
	return kls.kindListeners.addListener(l)
}

// remKindListener 移除类型事件监听者
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) remKindListener(lID ListenerID) bool {
	if kls.kindListeners == nil {
		return false
	}
	rem := kls.kindListeners.remListener(lID)
	if kls.kindListeners.noListener() {
		kls.kindListeners = nil
	}
	return rem
}

// addValueListener 添加值类型事件监听者
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) addValueListener(value EventValue, l *listener[EventKind, EventValue, ListenerID]) bool {
	if kls.dispatching > 0 {
		// 不能在派发事件状态下，嵌套添加监听者
		panic("add value listener on dispatching")
	}
	if kls.valueListeners == nil {
		kls.valueListeners = map[EventValue]*listenerContainer[EventKind, EventValue, ListenerID]{}
	}
	lc := kls.valueListeners[value]
	if lc == nil {
		lc = newListenerContainer[EventKind, EventValue, ListenerID]()
		kls.valueListeners[value] = lc
	}
	return lc.addListener(l)
}

// remValueListener 移除值类型事件监听者
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) remValueListener(value EventValue, lID ListenerID) bool {
	lc := kls.valueListeners[value]
	if lc != nil {
		rem := lc.remListener(lID)
		if lc.noListener() {
			delete(kls.valueListeners, value)
		}
		if len(kls.valueListeners) == 0 {
			kls.valueListeners = nil
		}
		return rem
	}
	return false
}

// noListener 返回是否没有监听者
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) noListener() bool {
	return (kls.kindListeners == nil || kls.kindListeners.noListener()) && (kls.valueListeners == nil || len(kls.valueListeners) == 0)
}

// dispatch 向监听者们派发事件
// 先派发类型事件，再配发值类事件
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) dispatch(evt Event[EventKind, EventValue]) error {
	kls.dispatching++
	var errs []error

	if kls.kindListeners != nil {
		if err := kls.kindListeners.dispatch(evt); err != nil {
			errs = append(errs, &dispatchError[EventKind, EventValue]{
				dt:      "kind",
				eventID: evt.eventID,
				err:     err,
			})
		}
		if kls.kindListeners.noListener() {
			kls.kindListeners = nil
		}
	}

	if kls.valueListeners != nil {
		lc := kls.valueListeners[evt.eventID.Value]
		if lc != nil {
			if err := lc.dispatch(evt); err != nil {
				errs = append(errs, &dispatchError[EventKind, EventValue]{
					dt:      "value",
					eventID: evt.eventID,
					err:     err,
				})
			}
			if lc.noListener() {
				delete(kls.valueListeners, evt.eventID.Value)
				if len(kls.valueListeners) == 0 {
					kls.valueListeners = nil
				}
			}
		}
	}

	kls.dispatching--
	if kls.dispatching < 0 {
		kls.dispatching = 0
	}

	if len(errs) > 0 {
		return &dispatchErrors{errors: errs}
	}

	return nil
}

// clear 清理容器，移除所有监听者
func (kls *kindListenerContainer[EventKind, EventValue, ListenerID]) clear() {
	if kls.dispatching > 0 {
		// 派发中不能清理
		return
	}
	if kls.kindListeners != nil {
		kls.kindListeners.clear()
		kls.kindListeners = nil
	}

	for _, v := range kls.valueListeners {
		v.clear()
	}
	kls.valueListeners = nil
}
