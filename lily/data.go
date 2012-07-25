package lily
import (
	"fmt"
	"strings"
)

type Elem interface {
	String() string
}


type Duration struct  {
	DurationLog int
	Dots int
	// todo - triplets.
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
		n = fmt.Sprintf("%d", i)
	}

	for i := 0; i < d.Dots; i++ {
		n += "."
	}

	return n
}

type Pitch struct  {
	Octave int
	Notename int
	Alteration int
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

type Note struct  {
	Pitch
	Duration
}

func (p *Note) String() string {
	d := &p.Duration
	fmt.Println(d)
	return p.Pitch.String() + d.String()
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
	return fmt.Sprintf("{ %s }", s.Compound.String())
}

type Par struct {
	Compound
}

func (s *Par) String() string {
	return fmt.Sprintf("<< %s >>", s.Compound.String())
}
