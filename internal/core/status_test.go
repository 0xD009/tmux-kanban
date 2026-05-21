package core

import "testing"

func TestApplyPolledStatus(t *testing.T) {
	tests := []struct {
		name       string
		current    SessionStatus
		hasCurrent bool
		polled     SessionStatus
		want       SessionStatus
	}{
		{name: "new idle", polled: StatusIdle, want: StatusIdle},
		{name: "need review wins over working", current: StatusWorking, hasCurrent: true, polled: StatusNeedReview, want: StatusNeedReview},
		{name: "working to idle becomes done", current: StatusWorking, hasCurrent: true, polled: StatusIdle, want: StatusDone},
		{name: "done is sticky over idle", current: StatusDone, hasCurrent: true, polled: StatusIdle, want: StatusDone},
		{name: "working can revive done", current: StatusDone, hasCurrent: true, polled: StatusWorking, want: StatusWorking},
		{name: "need review can revive done", current: StatusDone, hasCurrent: true, polled: StatusNeedReview, want: StatusNeedReview},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ApplyPolledStatus(tt.current, tt.hasCurrent, tt.polled); got != tt.want {
				t.Fatalf("ApplyPolledStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestReviewQueueTracksStableCursorKey(t *testing.T) {
	q := ReviewQueue{}
	keys := []string{"first", "second", "third"}

	var moved bool
	q, moved = q.Move(keys, 1)
	if !moved || q.CursorKey != "second" {
		t.Fatalf("after move q=%#v moved=%v, want second", q, moved)
	}

	q = q.Clamp([]string{"second", "third"})
	if q.CursorKey != "second" || q.Cursor != 0 {
		t.Fatalf("after shrink q=%#v, want cursor on second at 0", q)
	}

	q = q.AdvanceAfter([]string{"second", "third"}, "second")
	if q.CursorKey != "third" {
		t.Fatalf("after advance q=%#v, want third", q)
	}
}
