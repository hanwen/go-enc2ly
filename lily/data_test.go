package lily

import (
	"math/big"
	"testing"
)

func TestDuration(t *testing.T) {
	d := Duration{
		DurationLog: 2,
		Dots: 1,
		Factor: big.NewRat(7, 5),
	}
	got := d.String()
	want := "4.*7/5"
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestNote(t *testing.T) {
	n := Chord{
		Pitch: []Pitch{{Octave: 2, Notename: 3, Alteration: -1}},
		Duration: Duration{DurationLog: 2, Dots: 1},
	}
	got := n.String()
	want := "fes'''4."
	if got != want {
		t.Errorf("got %s want %s", got, want)
	}
}
