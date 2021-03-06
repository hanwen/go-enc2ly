package lily

import (
	"fmt"
	"log"
	"math/big"
	"strings"
)

var _ = log.Printf

type Elem interface {
	String() string
}

type Duration struct {
	DurationLog int
	Dots        int

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

type BarCheck struct{}

func (b *BarCheck) String() string {
	return "|\n"
}

type TimeSignature struct {
	Num, Den int
}

func (t *TimeSignature) String() string {
	return fmt.Sprintf("\\time %d/%d", t.Num, t.Den)
}

type Pitch struct {
	Octave     int
	Notename   int
	Alteration int
}

func (p *Pitch) SemitonePitch() int {
	p.Normalize()
	scale := []int{0, 2, 4, 5, 7, 9, 11}
	return p.Octave*12 + scale[p.Notename] + p.Alteration
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
	alt := p.Alteration
	if alt < -2 || alt > 2 {
		log.Printf("illegal alteration %d", alt)
		alt = 0
	}
	n := names[p.Notename]
	
	n += altsuffix[alt+2]
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

type Chord struct {
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

type Skip struct {
	Duration
}

func (r *Skip) String() string {
	return "s" + r.Duration.String()
}

type Rest struct {
	Duration
}

func (r *Rest) String() string {
	return "r" + r.Duration.String()
}

type Tuplet struct {
	Num int
	Den int
	Elem
}

func (t *Tuplet) String() string {
	return fmt.Sprintf("\\times %d/%d %v",
		t.Num, t.Den, t.Elem)
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

func (c *Compound) Append(e Elem) {
	c.Elems = append(c.Elems, e)
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

type KeySignature struct {
	// TODO - should use pitch instead.
	Name      string
	ScaleType string
}

func (k *KeySignature) String() string {
	return fmt.Sprintf("\\key %s \\%s", k.Name, k.ScaleType)
}

type Clef struct {
	Name string
}

func (c *Clef) String() string {
	return fmt.Sprintf("\\clef \"%s\"", c.Name)
}

type Bar struct {
	Name string
}

func (b *Bar) String() string {
	return fmt.Sprintf("\\bar \"%s\"", b.Name)
}

type PropertySet struct {
	Context string
	Name    string

	// TODO - something more lispy? 
	Value string
}

func (p *PropertySet) String() string {
	return fmt.Sprintf("\\set %s.%s = #%s", p.Context, p.Name, p.Value)
}
