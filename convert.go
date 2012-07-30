package main
import (
	"enc2ly/lily"
	"fmt"
	"log"
)

func Convert(data *Data) {
	staves := map[int][]MeasElem{}
	for _, m := range data.Measures {
		for _, e := range m.Elems {
			st := e.GetStaff()
			staves[st] = append(staves[st], e)
		}
	}

	for idx, elems := range staves {
		seq := ConvertStaff(elems, data.Cglx[idx].Clef)
		fmt.Printf("staff%d = %v\n", idx, seq)
	}
}

func ConvertRest(n *Rest) (dur lily.Duration) {
	dur.DurationLog = int(n.FaceValue) - 1
	if n.DotControl == 25 || n.DotControl == 29 {
		dur.Dots = 1
	}
	return dur
}

func ConvertNote(n *Note, baseStep lily.Pitch) (pit lily.Pitch, dur lily.Duration) {
	dur.DurationLog = int(n.FaceValue) - 1
	if n.DotControl == 25 || n.DotControl == 29 {
		dur.Dots = 1
	}

	baseStep.Notename += int(n.Position)
	baseStep.Normalize()
	baseStep.Alteration = int(n.SemitonePitch) - (baseStep.SemitonePitch() + 60)
	return baseStep, dur
}


// Returns the pitch for ledger line below staff.
func BasePitch(clefType byte) lily.Pitch {
	switch clefType {
	case 0:
		return lily.Pitch{
			Notename: 0,
			Octave: 0,
		}
	case 1:
		return lily.Pitch{
			Notename: 2,
			Octave: -2,
		}
	case 2:
		return lily.Pitch{
			Notename: 1,
			Octave: -1,
		}
	case 3:
		return lily.Pitch{
			Notename: 6,
			Octave: -2,
		}
	}
	return lily.Pitch{}
}
	

func ConvertStaff(elems []MeasElem, clefType byte) lily.Elem {
	seq := lily.Seq{}
	basePitch := BasePitch(clefType)
	lastTick := -1
	var lastNote *lily.Chord
	for _, e := range elems {
		switch t := e.(type) {
		case *Note:
			p, d := ConvertNote(t, basePitch)
			if e.GetTick() == lastTick {
				if lastNote == nil {
					log.Println("no last note at ", lastTick)
					continue
				}
				lastNote.Pitch = append(lastNote.Pitch, p)
			} else {
				ch := lily.Chord{Duration: d}
				ch.Pitch = append(ch.Pitch, p)
				lastNote = &ch
				seq.Elems = append(seq.Elems, lastNote)
			}
			lastTick = e.GetTick()
		case *Rest:
			d := ConvertRest(t)
			seq.Elems = append(seq.Elems, &lily.Rest{d})
		default:
			continue
		}
	}
	return &seq
}
