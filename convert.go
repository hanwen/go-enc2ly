package main
import (
	"go-enc2ly/lily"
	"fmt"
	"log"
	"sort"
)

type ElemSequence []linkedMeasElem
func (e ElemSequence) Len() int {
	return len(e)
}

func (e ElemSequence) Less(i, j int) bool {
	return priority(e[i]) < priority(e[j])
}

func (e ElemSequence) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func priority(e linkedMeasElem) int {
	prio := int(e.GetTick()) << 10
	switch e.GetType() {
	case 8: fallthrough
	case 9:
		prio += 10

	// pref matter:
	case 1: fallthrough
	case 2: 
		prio += 0

	default: prio += 20
	}
	
	return prio
}

type linkedMeasElem struct {
	MeasElem

	measure *Measure
	staff   *Staff
}

func Convert(data *Data) {
	staves := map[int][]linkedMeasElem{}
	for _, m := range data.Measures {
		measStaves := map[int][]linkedMeasElem{}
		for _, e := range m.Elems {
			key := e.Voice() + e.GetStaff() << 4
			l := linkedMeasElem{
				MeasElem: e,
				measure: &m,
				staff: &data.Staff[e.GetStaff()],
			}
			measStaves[key] = append(measStaves[key], l)
		}

		for k, v := range measStaves {
			sort.Sort(ElemSequence(v))
			staves[k] = append(staves[k], v...)
		}
	}

	for idx, elems := range staves {
		seq := ConvertStaff(elems, data.Staff[idx].Clef)
		staff := idx >> 4
		voice := idx & 0xf
 		fmt.Printf("staff%svoice%s = %v\n", Int2Letter(staff), Int2Letter(voice), seq)
	}
}

func ConvertRest(n *Rest) (dur lily.Duration) {
	dur.DurationLog = int(n.FaceValue) - 1
	if n.DotControl == 25 || n.DotControl == 29 {
		dur.Dots = 1
	}
	return dur
}

func Int2Letter(a int) string {
	return string(byte(a) + 'A')
}


func ConvertNote(n *Note, baseStep lily.Pitch) (pit lily.Pitch, dur lily.Duration) {
	dur.DurationLog = n.DurationLog()
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
	

func ConvertStaff(elems []linkedMeasElem, clefType byte) lily.Elem {
	seq := lily.Seq{}
	basePitch := BasePitch(clefType)
	lastTick := -1
	var lastNote *lily.Chord
	var articulations []string 
	for i, e := range elems {
		if e.GetTick() != lastTick && lastNote != nil {
			lastNote.PostEvents = articulations
			articulations = nil
		}
		
		if e.GetTick() == 0 && lastTick > 0 && e.GetDurationTick() > 0 {
			seq.Elems = append(seq.Elems, &lily.BarCheck{})
		}

		if i == 0 || (e.GetTick() == 0 && elems[i-1].measure.TimeSignature() != e.measure.TimeSignature()) {
			seq.Elems = append(seq.Elems, &lily.TimeSignature{
				Num: int(e.measure.TimeSigNum),
				Den: int(e.measure.TimeSigDen),
			})
		}
		
		switch t := e.MeasElem.(type) {
		case *Tie:
			if lastNote == nil {
				log.Println("no last for tie ", lastTick)
			} else {
				articulations = append(articulations, "~")
			}
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
