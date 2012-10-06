package main

import (
	"fmt"
	"log"
	"math/big"
	"sort"

	"github.com/hanwen/go-enc2ly/lily"
	"github.com/hanwen/go-enc2ly/encore"
)

// TODO - text elements

// TODO - slur elements
// slurs are tricky: they are not explicitly linked to their encompassing notes.

// TODO - clef changes,

type elemSequence []*encore.MeasElem

func (e elemSequence) Len() int {
	return len(e)
}

func (e elemSequence) Less(i, j int) bool {
	return priority(e[i]) < priority(e[j])
}

func (e elemSequence) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func priority(e *encore.MeasElem) int {
	prio := e.AbsTick() << 10
	switch e.Type() {
	case encore.TYPE_REST:
		fallthrough
	case encore.TYPE_NOTE:
		prio += 10

	case encore.TYPE_BEAM:
		// Before notes, so we catch tuplets.
		prio += 5
	// pref matter:
	case encore.TYPE_CLEF:
		fallthrough
	case encore.TYPE_KEYCHANGE:
		prio += 0

	default:
		prio += 20
	}

	return prio
}

type idKey struct {
	staff int
	voice int
}

type idKeys []idKey

func (e idKeys) Len() int {
	return len(e)
}

func (e idKeys) Less(i, j int) bool {
	if e[i].staff == e[j].staff {
		return e[i].voice == e[j].voice
	}
	return e[i].staff < e[j].staff
}

func (e idKeys) Swap(i, j int) {
	e[i], e[j] = e[j], e[i]
}

func (i *idKey) String() string {
	return fmt.Sprintf("staff%svoice%s", Int2Letter(i.staff), Int2Letter(i.voice))
}

func Convert(data *encore.Data) {
	staves := map[idKey][]*encore.MeasElem{}
	for _, m := range data.Measures {
		for _, e := range m.Elems {
			key := idKey{
				staff: int(e.StaffIdx),
				voice: e.Voice(),
			}
			staves[key] = append(staves[key], e)
		}
	}
	sortedKeys := idKeys{}
	for k := range staves {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(sortedKeys)
	staffVoiceMap := make(map[int][]idKey, len(data.Staff))
	for _, k := range sortedKeys {
		elems := staves[k]
		sort.Sort(elemSequence(elems))
		seq := convertStaff(elems)
		fmt.Printf("%v = %v\n", k.String(), seq)
		staffVoiceMap[k.staff] = append(staffVoiceMap[k.staff], k)
	}

	sortedStaves := []int{}
	for k := range staffVoiceMap {
		sortedStaves = append(sortedStaves, k)
	}
	sort.Ints(sortedStaves)
	fmt.Printf("<<\n")
	for _, k := range sortedStaves {
		voices := staffVoiceMap[k]
		fmt.Printf("  \\new Staff << \n")
		for _, voice := range voices {
			fmt.Printf("  \\new Voice \\%s\n", voice.String())
		}
		fmt.Printf(">>\n")
	}
	fmt.Printf(">>\n")
}

func convertClef(key byte) *lily.Clef {
	s := ""
	switch key {
	case 0:
		s = "G"
	case 1:
		s = "F"
	case 2:
		s = "C"
	case 3:
		s = "tenor"
	case 4:
		s = "G^8"
	case 5:
		s = "G_8"
	}

	return &lily.Clef{Name: s}
}

func convertKey(key byte) *lily.KeySignature {
	names := []string{
		"c", "f", "bes",
		"es", "as", "des", "ges", "ces", "g", "d", "a", "e", "b",
		"fis", "cis"}

	return &lily.KeySignature{
		Name:      names[key],
		ScaleType: "major",
	}
}

func convertBarType(end byte, start byte) string {
	switch start {
	case 2:
		if end == 4 {
			return ":|:"
		} else {
			return "|:"
		}
	case 1:
		return "|."
	case 3:
		return "||"
	case 8:
		return ":"
	case 0:
		if end == 4 {
			return ":|"
		} else {
			return "|"
		}
	}
	return ""
}

func convertRest(n *encore.Rest) (dur lily.Duration) {
	dur.DurationLog = int(n.FaceValue) - 1
	if n.DotControl == 25 || n.DotControl == 29 {
		dur.Dots = 1
	}
	return dur
}

func Int2Letter(a int) string {
	k := ""
	for a != 0 {
		k = k + string(byte(a&0xf) + 'A')
		a >>= 4
	}
	if k == "" {
		k = "A"
	}
	return k
}

func convertNote(n *encore.Note, baseStep lily.Pitch) (pit lily.Pitch, dur lily.Duration) {
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
func basePitch(clefType byte) lily.Pitch {
	switch clefType {
	case 0:
		return lily.Pitch{
			Notename: 0,
			Octave:   0,
		}
	case 1:
		return lily.Pitch{
			Notename: 2,
			Octave:   -2,
		}
	case 2:
		return lily.Pitch{
			Notename: 1,
			Octave:   -1,
		}
	case 3:
		return lily.Pitch{
			Notename: 6,
			Octave:   -2,
		}
	}
	return lily.Pitch{}
}

func skipTicks(ticks int) *lily.Skip {
	return &lily.Skip{
		Duration: lily.Duration{
			DurationLog: 4,
			Factor:      big.NewRat(int64(ticks), 60),
		},
	}
}

func setTuplet(t *lily.Tuplet, w *encore.WithDuration) {
	if t != nil && t.Num == 0 {
		t.Num = w.TupletNum()
		t.Den = w.TupletDen()
	}
}

func convertStaff(elems []*encore.MeasElem) lily.Elem {
	baseSeq := &lily.Seq{}
	seq := baseSeq

	lastTick := -1
	var lastNote *lily.Chord
	var articulations []string
	var nextTick int
	var endTupletTick int
	var currentTuplet *lily.Tuplet
	var currentVolta int
	for i, e := range elems {
		if e.AbsTick() != lastTick && lastNote != nil {
			lastNote.PostEvents = append(lastNote.PostEvents, articulations...)
			articulations = nil
		}

		if currentTuplet != nil && e.AbsTick() > endTupletTick {
			seq = baseSeq
			currentTuplet = nil
			endTupletTick = 0
		}

		if e.GetTick() == 0 && e.AbsTick() > lastTick && e.GetDurationTick() > 0 {
			seq.Append(&lily.BarCheck{})
		}

		if i == 0 || e.Measure != elems[i-1].Measure && e.GetTick() == 0 {
			var last byte
			if i > 0 {
				last = elems[i-1].Measure.BarTypeEnd
			}
			barType := convertBarType(last, e.Measure.BarTypeStart)
			if barType != "|" {
				seq.Append(&lily.Bar{Name: barType})
			}

			if volta := int(e.Measure.RepeatAlternative); volta != currentVolta {
				val := "'((volta  #f))"
				if volta > 0 {
					val = fmt.Sprintf("'((volta \"%d\"))", volta)
				}
				r := &lily.PropertySet{
					Context: "Score",
					Name:    "repeatCommands",
					Value:   val,
				}
				seq.Append(r)
				currentVolta = volta
			}
		}
		if i == 0 || (e.GetTick() == 0 && elems[i-1].Measure.TimeSignature() != e.Measure.TimeSignature()) {
			seq.Append(&lily.TimeSignature{
				Num: int(e.Measure.TimeSigNum),
				Den: int(e.Measure.TimeSigDen),
			})
		}
		if i == 0 {
			seq.Append(convertKey(e.LineStaffData.Key))
			seq.Append(convertClef(e.LineStaffData.Clef))
		}

		if nextTick < e.AbsTick() {
			seq.Append(skipTicks(e.AbsTick() - nextTick))
			nextTick = e.AbsTick()
		}

		end := e.AbsTick() + e.GetDurationTick()
		switch t := e.TypeSpecific.(type) {
		case *encore.Beam:
			if t.TupletNumber != 0 {
				if currentTuplet != nil {
					log.Panic("already have tuplet")
				}

				endTupletTick = e.Measure.AbsTick + int(t.EndNoteTick)
				seq = new(lily.Seq)
				currentTuplet = &lily.Tuplet{Elem: seq}
				baseSeq.Append(currentTuplet)
			}
		case *encore.Tie:
			if lastNote == nil {
				log.Println("no last for tie ", lastTick)
			} else {
				articulations = append(articulations, "~")
			}
		case *encore.Note:
			setTuplet(currentTuplet, &t.WithDuration)
			p, d := convertNote(t, basePitch(e.LineStaffData.Clef))
			if e.AbsTick() == lastTick {
				if lastNote == nil {
					log.Println("no last note at ", lastTick)
					continue
				}
				lastNote.Pitch = append(lastNote.Pitch, p)
			} else {
				ch := lily.Chord{Duration: d}
				ch.Pitch = append(ch.Pitch, p)
				lastNote = &ch
				seq.Append(lastNote)
			}
			lastTick = e.AbsTick()
			if end > nextTick {
				nextTick = end
			}
		case *encore.Rest:
			setTuplet(currentTuplet, &t.WithDuration)
			d := convertRest(t)
			seq.Append(&lily.Rest{d})
			if end > nextTick {
				nextTick = end
			}
		case *encore.KeyChange:
			seq.Append(convertKey(t.NewKey))
		}
	}
	if lastNote != nil {
		lastNote.PostEvents = append(lastNote.PostEvents, articulations...)
	}
	return baseSeq
}
