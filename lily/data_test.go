package lily

import (
	"testing"
)

func TestDuration(t *testing.T) {
	d := Duration{DurationLog: 2, Dots: 1}
	got := d.String()
	want := "4."
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestNote(t *testing.T) {
	n := Note{
		Pitch{Octave: 2, Notename: 3, Alteration: -1},
		Duration{DurationLog: 2, Dots: 1},
	}
	got := n.String()
	want := "fes'''4."
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}
