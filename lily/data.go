package lily
import (
	"fmt"
	"math/big"
	"log"
	"strings"
)
var _ = log.Printf
type Elem interface {
	String() string
}

type Duration struct  {
	DurationLog int
	Dots int
	
	// If not set, assume 1/1: 
	Factor *big.Rat
}

func (d *Duration) String() string {
	names := map[int]string{
		-1: "\\breve",
		-2: "\\maxima",
	}
	n := names[d.DurationLog]
	if n == "" {
		i := uint(1)
		i <<= uint(d.DurationLog)
		if i == 0 {
			panic(d.DurationLog)
		}
			
		n = fmt.Sprintf("%d", i)
	}

	for i := 0; i < d.Dots; i++ {
		n += "."
	}
	if d.Factor != nil {
		n += "*" + d.Factor.RatString()
	}
	return n
}

type BarCheck struct {}

func (b *BarCheck) String() string {
	return "|\n"
}

type TimeSignature struct {
	Num, Den int
}

func (t *TimeSignature) String() string {
	return fmt.Sprintf("\\time %d/%d", t.Num, t.Den)
}

type Pitch struct  {
	Octave int
	Notename int
	Alteration int
}

func (p *Pitch) SemitonePitch() int {
	p.Normalize()
	scale := []int{0, 2, 4, 5, 7, 9, 11}
	return p.Octave * 12 + scale[p.Notename] + p.Alteration
}

func (p *Pitch) Normalize() {
	for p.Notename < 0 {
		p.Notename += 7
		p.Octave--
	}
	for p.Notename >= 7 {
		p.Notename -= 7
		p.Octave++
	}
}

func (p *Pitch) String() string {
	names := []string{"c", "d", "e", "f", "g", "a", "b"}
	altsuffix := []string{"eses", "es", "", "is", "isis"}

	n := names[p.Notename]
	n += altsuffix[p.Alteration + 2]
	if p.Octave < 0 {
		for i := -1; i > p.Octave; i-- {
			n += ","
		}
	} else {
		for i := 0; i <= p.Octave; i++ {
			n += "'"
		}
	}
	return n
}

type Chord struct  {
	Pitch []Pitch
	Duration
	PostEvents []string
}

func (p *Chord) String() string {
	d := &p.Duration
	pstr := "s"
	if len(p.Pitch) == 1 {
		pstr = p.Pitch[0].String()
	} else if len(p.Pitch) > 1 {
		pitches := []string{}
		for _, p := range p.Pitch {
			pitches = append(pitches, p.String())
		}
		pstr = "<" + strings.Join(pitches, " ") + ">"
	}
	
	pstr += d.String()
	for _, e := range p.PostEvents {
		pstr += "-" + e
	}
	return pstr
}

type Skip struct  {
	Duration
}

func (r *Skip) String() string  {
	return "s" + r.Duration.String()
}

type Rest struct  {
	Duration
}

func (r *Rest) String() string  {
	return "r" + r.Duration.String()
}

type Compound struct {
	Elems []Elem
}

func (s *Compound) String() string {
	elts := []string{}
	for _, e := range s.Elems {
		elts = append(elts, e.String())
	}
	return strings.Join(elts, " ")
}

type Seq struct {
	Compound
}

func (s *Seq) String() string {
	return fmt.Sprintf("{\n%s\n}\n", s.Compound.String())
}

type Par struct {
	Compound
}

func (s *Par) String() string {
	return fmt.Sprintf("<< %s >>", s.Compound.String())
}

