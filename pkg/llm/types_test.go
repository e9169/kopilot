package llm

import "testing"

func TestEventEmitter(t *testing.T) {
	var emitter EventEmitter
	var got []EventType
	emitter.On(func(e Event) { got = append(got, e.Type) })
	emitter.On(func(e Event) { got = append(got, e.Type) })

	emitter.Emit(Event{Type: EventDelta})
	if len(got) != 2 {
		t.Fatalf("handler invocations = %d, want 2", len(got))
	}
	if got[0] != EventDelta || got[1] != EventDelta {
		t.Fatalf("events = %#v, want both delta", got)
	}
}
