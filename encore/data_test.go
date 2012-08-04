package encore

import (
	"testing"
)

func TestWithDuration(t *testing.T) {
	w := WithDuration{
		FaceValue:  4,
		DotControl: 1,
		Tuplet:     50,
	}
	got := w.GetDurationTick()
	want := 120
	if got != want {
		t.Errorf("GetDurationTick(%v) = %d want %d",
			w, got, want)
	}
}
