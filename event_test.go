package gevent

import (
	"math/rand"
	"testing"
)

type testET int
type testEV int64
type testLID int64
type testEventID = EventID[testET, testEV]
type testEvent = Event[testET, testEV]
type testListenerCallback = ListenerCallback[testET, testEV]

func TestTypeHandler(t *testing.T) {
	dispatcher := NewDispatcher[testET, testEV, testLID]()
	eventType := testET(1)
	value := 0

	key1 := testLID(1)
	add1 := 1
	callback1 := func(e testEvent) error {
		value += add1
		return nil
	}

	key2 := testLID(2)
	add2 := 2
	callback2 := func(e testEvent) error {
		value += add2
		return nil
	}

	dispatcher.AddKindListener(eventType, key1, callback1, false)
	dispatcher.AddKindListener(eventType, key2, callback2, false)
	if err := dispatcher.Dispatch(testEventID{eventType, 1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add1+add2 {
		t.Fatal("value must be", add1+add2)
	}

	value = 0
	dispatcher.RemKindListener(eventType, key2)
	if err := dispatcher.Dispatch(testEventID{eventType, 1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add1 {
		t.Fatal("value must be", add1)
	}

	value = 0
	dispatcher.RemKindListener(eventType, key1)
	if err := dispatcher.Dispatch(testEventID{eventType, 1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != 0 {
		t.Fatal("value must be", 0)
	}
	if len(dispatcher.kindListenerContainers) != 0 {
		t.Fatal("handlers must clear")
	}

	value = 0
	dispatcher.AddKindListener(eventType, key2, callback2, true)
	if err := dispatcher.Dispatch(testEventID{eventType, 1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add2 {
		t.Fatal("value must be", add2)
	}
	if len(dispatcher.kindListenerContainers) != 0 {
		t.Fatal("handlers must clear")
	}
}

func TestHandler(t *testing.T) {
	dispatcher := NewDispatcher[testET, testEV, testLID]()
	eventType1 := testET(1)
	eventType2 := testET(2)
	eventVal1 := testEV(1)
	eventVal2 := testEV(2)
	defaultValue := 0
	value := defaultValue

	key1 := testLID(1)
	add1 := rand.Int()
	handler1 := func(e testEvent) error {
		value += add1
		return nil
	}

	key2 := testLID(2)
	add2 := rand.Int()
	handler2 := func(e testEvent) error {
		value += add2
		return nil
	}

	dispatcher.AddValueListener(testEventID{eventType1, eventVal1}, key1, handler1, false)
	dispatcher.AddValueListener(testEventID{eventType2, eventVal2}, key2, handler2, false)
	if err := dispatcher.Dispatch(testEventID{eventType1, eventVal1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if err := dispatcher.Dispatch(testEventID{eventType2, eventVal2}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add1+add2 {
		t.Fatal("value must be", add1+add2)
	}

	value = 0
	if err := dispatcher.Dispatch(testEventID{eventType1, 0}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if err := dispatcher.Dispatch(testEventID{eventType2, 0}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != defaultValue {
		t.Fatal("value must be", defaultValue)
	}

	value = 0
	if err := dispatcher.Dispatch(testEventID{eventType1, eventVal1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add1 {
		t.Fatal("value must be", add1)
	}

	value = 0
	dispatcher.AddKindListener(eventType1, key1, handler1, false)
	if err := dispatcher.Dispatch(testEventID{eventType1, eventVal1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add1+add1 {
		t.Fatal("value must be", add1+add1)
	}

	value = 0
	if err := dispatcher.Dispatch(testEventID{eventType2, eventVal2}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add2 {
		t.Fatal("value must be", add2)
	}

	value = 0
	dispatcher.AddKindListener(eventType2, key2, handler2, false)
	if err := dispatcher.Dispatch(testEventID{eventType2, eventVal2}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add2+add2 {
		t.Fatal("value must be", add2+add2)
	}

	value = 0
	if err := dispatcher.Dispatch(testEventID{eventType1, eventVal1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if err := dispatcher.Dispatch(testEventID{eventType2, eventVal2}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != add1*2+add2*2 {
		t.Fatal("value must be", add1*2+add2*2)
	}

	value = 0
	dispatcher.Clear()
	if err := dispatcher.Dispatch(testEventID{eventType1, eventVal1}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if err := dispatcher.Dispatch(testEventID{eventType2, eventVal2}, nil, nil); err != nil {
		t.Fatal("there must no error")
	}
	if value != defaultValue {
		t.Fatal("value must be", defaultValue)
	}
}

func BenchmarkAddHandler(b *testing.B) {
	maxEventType := 100
	maxEventVal := 1000
	callback := func(event testEvent) error {
		return nil
	}

	b.Run("build", func(b *testing.B) {
		dispatcher := NewDispatcher[testET, testEV, testLID]()

		for i := 0; i < maxEventType; i++ {
			dispatcher.AddKindListener(testET(i), 0, callback, false)
			for j := 0; j < maxEventVal; j++ {
				dispatcher.AddValueListener(testEventID{testET(i), testEV(j)}, 0, callback)
			}
		}

		b.Run("dispatch type", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				dispatcher.Dispatch(
					testEventID{testET(rand.Intn(maxEventType)), testEV(rand.Intn(maxEventVal))},
					nil,
					nil,
				)
			}
		})

		mm := map[testET]testListenerCallback{}
		for i := 0; i < maxEventType; i++ {
			mm[testET(i)] = callback
		}
		b.Run("dispatch type map", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				evt := testEvent{
					eventID: testEventID{testET(rand.Intn(maxEventType)), testEV(rand.Intn(maxEventVal))},
				}
				mm[testET(rand.Intn(maxEventType))](evt)
			}
		})
	})

}
