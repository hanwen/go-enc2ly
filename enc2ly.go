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

type Tag [4]byte

func (t Tag) Tag() string {
	return string(t[:])
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

type MeasElem interface {
	GetRaw() []byte
	GetStaff() int
	GetOffset() int
	Sz() int
}

type MeasElemBase struct {
	Raw []byte
	Offset int
	Tick  uint16 `offset:"0"`
	Size  byte `offset:"3"`
	Staff byte `offset:"4"`
}

func (n *MeasElemBase) GetRaw() []byte {
	return n.Raw
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
	
	// type off 5: 204=?, 133=?, 1 = note, 30 = ? , 2=?
	DurationTicks   uint16 `offset:"16"`

	// ledger below staff = 0; top line = 10
	Position        int8 `offset:"12"`

	// 1=sharp, 2=flat, 3=natural, 4=dsharp, 5=dflat
	// used as offset in font. Using 6 gives a longa symbol
	AlterationGlyph byte `offset:"21"`
	SemitonePitch   byte `offset:"15"`
}

type Other struct {
	MeasElemBase
}

type Script struct {
	MeasElemBase
	XOff byte `offset:"10"`
}

type Beam struct {
	MeasElemBase
	LeftPos int8 `offset:"18"` 
	RightPos int8 `offset:"19"` 
}

type Rest struct {
	MeasElemBase
	
	DotControl byte `offset:"14"`
	XOffset    byte `offset:"10"`
	Position int8 `offset:"12"`
	DurationTicks   uint16 `offset:"16"`
}

func readElem(c []byte, off int) (e MeasElem) {
	// ugh - how to determine the type for each element?
	switch len(c) {
	case 28:
		e = &Note{} 
	case 18:
		e = &Rest{}
	case 30:
		e = &Beam{}
	case 16:
		e = &Script{}
	default:
		e = &Other{}
	}
	FillBlock(c, off, e)
	return e
}



func (n *Beam) Sz() int {
	return len(n.Raw)
}

func (n *Beam) GetOffset() int {
	return int(n.Offset)
}

func (n *Beam) GetStaff() int {
	return int(n.Staff)
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

type CGLX struct {
	Offset int
	Raw []byte `want:"CGLX" fixed:"242"`
}

type CGLXTrailer struct {
	Offset int
	Raw []byte `want:"CGLX" fixed:"5"`
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
	f.Cglx = make([]CGLX, f.Header.CglxCount-1)
	for i := 0; i < int(f.Header.CglxCount-1); i++ {
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

	d := &Data{}
	err = readData(content, d)
	if err != nil {
		log.Fatalf("readData %v", err)
	}
	
	fmt.Printf("meas 0\n")
	type Pair struct { A,B int }
	sizes := map[Pair]bool{}
	for _, m:= range d.Measures {
		for _, e := range m.Elems {
			sizes[Pair{e.Sz(), int(e.GetRaw()[5])}] = true
		}
	}
	fmt.Println(sizes)
	//messM(d)
}

func messM(d *Data) {
	raw := make([]byte, len(d.Raw))
	copy(raw, d.Raw)

	d2 := Data{}
	readData(raw, &d2)
	
	for _, m:= range d2.Measures[0].Elems {
		fmt.Printf("%+v\n", m)
	}
	
	err := ioutil.WriteFile("mess.enc", raw, 0644)
	if err != nil {
		log.Fatalf("WriteFile:", err)
	}
	
}

func mess(d *Data) {
	fmt.Printf("mess\n")
	for _, m:= range d.Measures[0].Elems {
		fmt.Printf("%+v\n", m)
	}

	raw := make([]byte, len(d.Raw))
	copy(raw, d.Raw)

	raw[d.Measures[0].Elems[0].GetOffset() + 14] = 0x81

	d2 := Data{}
	readData(raw, &d2)
	

	fmt.Printf("messed\n")
	for _, m:= range d2.Measures[0].Elems {
		fmt.Printf("%+v\n", m)
	}
	
	err := ioutil.WriteFile("mess.enc", raw, 0644)
	if err != nil {
		log.Fatalf("WriteFile:", err)
	}
}
