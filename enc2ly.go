package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
	"strconv"
)

func (h *Header) FillFirstCgxl(cglx *CGLX) {
	raw := h.Raw[189:]
	cglx.Raw = raw
	cglx.Offset = 189
		
	FillFields(raw, cglx)
}

type Header struct {
	Offset     int
	Raw []byte `want:"SCOW" fixed:"431"`

	LineCount      int16 `offset:"0x2e"`
	PageCount      int16 `offset:"0x30"`
	CglxCount      byte  `offset:"0x32"`
	StaffPerSystem byte  `offset:"0x33"`
	MeasureCount   int16 `offset:"0x34"`
}

type Page struct {
	Offset int
	Raw []byte `want:"PAGE" fixed:"34"`
}

type Line struct {
	Offset int
	Raw     []byte `want:"LINE" fixed:"8"`
	VarSize int32  `offset:"0x4"`
	VarData []byte
}

type Measure struct {
	Offset int
	Raw     []byte `want:"MEAS" fixed:"62"`

	VarSize int32  `offset:"4"`
	Bpm     uint16 `offset:"8"`
	TimeSigGlyph byte `offset:"10"`
	TimeSigNumTicks uint16 `offset:"12"`
	TimeSigDenTicks uint16 `offset:"14"`
	TimeSigNum byte `offset:"16"`
	TimeSigDen byte `offset:"17"`
	
	VarData []byte
	Elems []MeasElem
}

type CGLX struct {
	Offset int
	Raw []byte `want:"CGLX" fixed:"242"`
	Name [10]byte `offset:"13"`

	// 174, 175, 
	
	// In semitones; b-flat clar = -2
	Transposition int8 `offset:"178"`

	Unknown byte `offset:"181"`
	
	// 0 = G, 1 = F, 2 = C(middle), 3=C(tenor), 4=G^8, 5=G_8,
	// 6=F_8
	Clef byte `offset:"185"`

	// 186 = 1 for piano staff. ?
	
	// 193 - 200: MIDI channel (repeated?)
	// 201 - 208: MIDI program (repeated?)
	// 209 - 216: MIDI volume (repeated?)

	// 218 ?

}

type CGLXTrailer struct {
	Offset int
	Raw []byte `want:"CGLX" fixed:"5"`
}

type MeasElem interface {
	GetTick() int
	GetRaw() []byte
	GetStaff() int
	GetOffset() int
	Sz() int
	GetType() int
	GetTypeName() string
}

// Voice (1-8) should be somewhere too.
type MeasElemBase struct {
	Raw []byte
	Offset int
	Tick  uint16 `offset:"0"`
	Type  byte `offset:"2"`
	Size  byte `offset:"3"`
	Staff byte `offset:"4"`
}

func (n *MeasElemBase) GetRaw() []byte {
	return n.Raw
}

func (n *MeasElemBase) GetTick() int {
	return int(n.Tick)
}

func (n *MeasElemBase) GetType() int {
	return int(n.Type)
}

func (n *MeasElemBase) Sz() int {
	return len(n.Raw)
}

func (n *MeasElemBase) GetStaff() int {
	return int(n.Staff)
}

func (n *MeasElemBase) GetOffset() int {
	return int(n.Offset)
}

type Note struct {
	MeasElemBase
	XOffset       byte `offset:"10"`

	// 4 = 8th, 3=quarter, 2=half, etc.
	FaceValue     byte `offset:"5"`
	
	// ledger below staff = 0; top line = 10
	Position        int8 `offset:"12"`

	// 25 = same pos as head, 29 for dot 1 position above head
	DotControl byte `offset:"14"`

	// Does not include staff wide transposition setting; 60 = central C.
	SemitonePitch   byte `offset:"15"`
	
	DurationTicks   uint16 `offset:"16"`

	// Not sure - but encore defaults to 64; and all have this?
	Velocity byte `offset:"19"`
	
	// 128 = stem-down bit
	// 7 = unbeamed?
	Options byte `offset:"20"`
	
	// 1=sharp, 2=flat, 3=natural, 4=dsharp, 5=dflat
	// used as offset in font. Using 6 gives a longa symbol
	AlterationGlyph byte `offset:"21"`
}

func (n *Note) Alteration() int {
	switch n.AlterationGlyph {
	case 1: return 1
	case 2: return -1
	case 3: return 0
	case 4: return 2
	case 5: return -2
	}
	return 0
}

func (o *Note) GetTypeName() string {
	return "Note"
}

type Slur struct {
	MeasElemBase
	
	// 33 = slur, 16=8va, ... ?
	SlurType  byte `offset:"5"`
	LeftX byte `offset:"10"`
	LeftPosition byte `offset:"12"`
	MiddleX byte `offset:"14"`
	MiddlePosition byte `offset:"16"`
	RightX byte `offset: "20"`
	RightPosition byte `offset:"22"`
}


func (o *Slur) GetTypeName() string {
	return "Slur"
}

type KeyChange struct {
	MeasElemBase
	NewKey byte  `offset:"5"`
	OldKey byte  `offset:"10"`
}

func (o *KeyChange) GetTypeName() string {
	return "KeyChange"
}

type Other struct {
	MeasElemBase
}

func (o *Other) GetTypeName() string {
	return "Other"
}

type Script struct {
	MeasElemBase
	XOff byte `offset:"10"`
}

func (o *Script) GetTypeName() string {
	return "Script"
}

// Also used for tuplet bracket.
type Beam struct {
	MeasElemBase
	LeftPos int8 `offset:"18"` 
	RightPos int8 `offset:"19"` 
}

func (o *Beam) GetTypeName() string {
	return "Beam"
}

type Rest struct {
	MeasElemBase
	
	FaceValue     byte `offset:"5"`
	DotControl byte `offset:"14"`
	XOffset    byte `offset:"10"`
	Position int8 `offset:"12"`
	DurationTicks   uint16 `offset:"16"`
}

func (o *Rest) GetTypeName() string {
	return "Rest"
}

func readElem(c []byte, off int) (e MeasElem) {
	switch c[2] {
	case 144:
		fallthrough
	case 145:
		e = &Note{}
	case 32:
		e = &KeyChange{}
	case 48:
		fallthrough
	case 128:
		e = &Rest{}
	case 64:
		e = &Beam {}
	case 80:
		switch c[3] {
		case 16:
			e = &Script{}
		case 86:
		case 28:
			e = &Slur{}
		}
	}
	if e == nil {
		e = &Other{}
	}
	
	FillBlock(c, off, e)
	return e
}


var endMarker = string([]byte{255,255})

func (m *Measure) ReadElems() {
	r := m.VarData
	off := m.Offset + 62  // todo - extract.
	for len(r) >= 3	{
		sz := int(r[3])

		m.Elems = append(m.Elems, readElem(r[:sz], off))
		r = r[sz:]
		off += sz
	}

	if string(r) != endMarker {
		log.Fatalf("end marker not found")
	}
}

func FillBlock(raw []byte, offset int, dest interface{}) {
	v := reflect.ValueOf(dest).Elem()
	byteOffAddr := v.FieldByName("Offset").Addr().Interface().(*int)
	*byteOffAddr = offset

	rawAddr := v.FieldByName("Raw").Addr().Interface().(*[]byte)
	*rawAddr = raw
	
	FillFields(raw, dest)
}

func FillFields(raw []byte, dest interface{}) {
	v := reflect.ValueOf(dest).Elem()
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Anonymous {
			FillFields(raw, v.Field(i).Addr().Interface())
			continue
		}
		f := v.Field(i)
		offStr := v.Type().Field(i).Tag.Get("offset")
		if offStr == "" {
			continue
		}

		off, _ := strconv.ParseInt(offStr, 0, 64)

		z := f.Addr().Interface()
		binary.Read(bytes.NewBuffer(raw[off:]), binary.LittleEndian, z)
	}
}

func ReadTaggedBlock(c []byte, off int, dest interface{}) int {
	v := reflect.ValueOf(dest).Elem()
	
	tagField, ok := v.Type().FieldByName("Raw")
	if !ok {
		log.Fatalf("missing Raw in %T", dest)
	}

	want := tagField.Tag.Get("want")
	if want == "" {
		log.Fatal("missing want", dest)
	}

	fixed := tagField.Tag.Get("fixed")
	sz, _ := strconv.ParseInt(fixed, 0, 64)
	rawAddr := v.FieldByName("Raw").Addr().Interface().(*[]byte)
	raw := c[off:off+int(sz)]
	*rawAddr = raw
	if string(raw[:4]) != want {
		log.Fatalf("Got tag %q want %q - %q", raw[:4], want, raw)
	}
	FillBlock(raw, off, dest)
	return int(sz)
}

func (h *Header) String() string {
	return fmt.Sprintf("Systems %d PAGE %d CGLX %d staffpersys %d MEAS %d",
		h.LineCount, h.PageCount, h.CglxCount, h.StaffPerSystem,
		h.MeasureCount)
}

type Data struct {
	Raw      []byte
	Header   Header
	Cglx     []CGLX
	Pages    []Page
	Lines    []Line
	Measures []Measure
}

func readData(c []byte, f *Data) error {
	f.Raw = c
	off := 0
	off += ReadTaggedBlock(c, off, &f.Header)
	f.Cglx = make([]CGLX, f.Header.CglxCount)
	f.Header.FillFirstCgxl(&f.Cglx[0])
	for i := 1; i < int(f.Header.CglxCount); i++ {
		off += ReadTaggedBlock(c, off, &f.Cglx[i])
	}
	trailer := CGLXTrailer{}
	off += ReadTaggedBlock(c, off, &trailer)
	f.Pages = make([]Page, f.Header.PageCount)
	for i := 0; i < int(f.Header.PageCount); i++ {
		off += ReadTaggedBlock(c, off, &f.Pages[i])
	}

	f.Lines = make([]Line, f.Header.LineCount)
	for i := 0; i < int(f.Header.LineCount); i++ {
		l := &f.Lines[i]
		off += ReadTaggedBlock(c, off, l)
		l.VarData = c[off:off+int(l.VarSize)]
		off += int(l.VarSize)
	}

	f.Measures = make([]Measure, f.Header.MeasureCount)
	for i := 0; i < int(f.Header.MeasureCount); i++ {
		m := &f.Measures[i]
		off += ReadTaggedBlock(c, off, m)
		m.VarData = c[off:off+int(m.VarSize)]
		off += int(m.VarSize)
		m.ReadElems()
	}

	return nil
}

func isH(x []byte) bool {
	for i := 0; i < 4; i++ {
		if !('A' <= x[i] && x[i] <= 'Z') {
			return false
		}
	}
	return true
}

func dumpBytes(d []byte) {
	for i, c := range d {
		fmt.Printf("%5d: %3d", i, c)
		if i % 4 == 3 && i > 0 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n")
}

func analyzeTags(content []byte) {
	tags := map[string]int{}
	lastHI := 0
	lastHName := ""
	for i, _ := range content {
		if isH(content[i:]) && i-lastHI > 4 {
			log.Printf("Header %q, delta %d", lastHName, i-lastHI)
			sectionContent := content[lastHI:i]
			want := []byte("Flaut")
			if idx:= bytes.Index(sectionContent, want); idx > 0 {
				log.Println("found first staff", idx)

				log.Printf("content %q", content[200:432])
			}
			lastHI = i
			lastHName = string(content[i : i+4])
			tags[lastHName]++
		}
	}

	if false {
		// find size counter in header.
		log.Println(tags)
		head := content[:341]
		for t, cnt := range tags {
			offsets := []int{}
			for i, c := range head {
				if cnt == int(c) {
					offsets = append(offsets, i)
				}
			}

			log.Printf("tag %q can be at %v", t, offsets)
		}
	}
}

func main() {
	flag.Parse()
	content, err := ioutil.ReadFile(flag.Arg(0))
	if err != nil {
		log.Fatal("ReadFile", err)
	}

	//	analyzeTags(content)
	d := &Data{}
	err = readData(content, d)
	if err != nil {
		log.Fatalf("readData %v", err)
	}
	//	analyzeCglx(d)
	//	messM(d)
	//mess(d)
	//	analyzeKeyCh(d)
	//analyzeAll(d)
	//	analyzeStaff(d)
	//	analyzeMeasStaff(d)
	//	analyzeLine(d)	
}

func analyzeLine(d *Data) {
	for i, l  := range d.Lines {
		fmt.Printf("linesize %d %v\n", i, l.VarSize)
		fmt.Printf(" %v\n", l.VarData)
	}
}

func analyzeAll(d *Data) {
	for i, m := range d.Measures[:] {
		fmt.Printf("meas %d\n", i)
		for _, e  := range m.Elems {
			fmt.Printf("%+v\n", e)
		}
	}
}

func analyzeStaff(d *Data) {
	for _, m := range  d.Measures {
		for _, e  := range m.Elems {
			if e.GetStaff() == 0 && e.GetTypeName() == "Note"{
				fmt.Printf("%+v\n", e)
			}
		}
	}
}

func analyzeMeasStaff(d *Data) {
	for _, e  := range d.Measures[0].Elems {
		if e.GetStaff() == 0{
			fmt.Printf("%+v\n", e)
		}
	}
}

func analyzeKeyCh(d *Data) {
	for i, m  := range d.Measures {
		for j, e  := range m.Elems {
			if e.GetType() == 32 {
				log.Printf("meas %d elt %d staff %d", i, j,
					e.GetStaff())
			}
		}
	}
}

func analyzeCglx(d *Data) {
	occs := make([]map[int]int, 242)
	for i := 0; i < 242; i++ {
		occs[i] = make(map[int]int)
	}
	
	for _, c := range d.Cglx {
		for i := range c.Raw {
			occs[i][int(c.Raw[i])]++
		}
	}
	log.Printf("looking for key")
	for j := 0; j < 242; j++ {
		if len(occs[j]) == 1 {
			continue
		}
		log.Println("values", j, len(occs[j]))
		for _, c := range d.Cglx {
			fmt.Printf("%d ", c.Raw[j])
		}
		fmt.Printf("\n")
	}
	
	for i := 0; i < 242; i++ {
		if len(occs[i]) == 3 {
			fmt.Printf("%d: %d diff %v\n", i, len(occs[i]), occs[i])
		}
	}

}

func messM(d *Data) {
	raw := make([]byte, len(d.Raw))
	copy(raw, d.Raw)

	d2 := Data{}
	readData(raw, &d2)
	
	err := ioutil.WriteFile("mess.enc", raw, 0644)
	if err != nil {
		log.Fatalf("WriteFile:", err)
	}
	
}

func mess(d *Data) {
	fmt.Printf("mess\n")
	raw := make([]byte, len(d.Raw))
	copy(raw, d.Raw)

	for _, m := range d.Measures[:1] {
		for _, e  := range m.Elems {
			if e.GetTypeName() == "Slur" {
				raw[e.GetOffset() + 5] /= 2
			}
		}
	}

	d2 := Data{}
	readData(raw, &d2)
	fmt.Printf("messed\n")
	
	err := ioutil.WriteFile("mess.enc", raw, 0644)
	if err != nil {
		log.Fatalf("WriteFile:", err)
	}
}
