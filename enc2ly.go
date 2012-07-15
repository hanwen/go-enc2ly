package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
//	"io"
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
	Symbol  byte  `offset:"10"`
	TimeSigCode uint32 `offset:"8"`
	TimeSigNum byte `offset:"0xc"`
	TimeSigDen byte `offset:"0xd"`
	
	VarSize int32  `offset:"0x4"`
	VarData []byte
}

type CGLX struct {
	Offset int
	Raw []byte `want:"CGLX" fixed:"242"`
}

type CGLXTrailer struct {
	Offset int
	Raw []byte `want:"CGLX" fixed:"5"`
}

func ReadBlock(c []byte, off int, dest interface{}) int {
	v := reflect.ValueOf(dest).Elem()
	byteOffAddr := v.FieldByName("Offset").Addr().Interface().(*int)
	*byteOffAddr = off
	
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
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)

		offStr := v.Type().Field(i).Tag.Get("offset")
		if offStr == "" {
			continue
		}

		off, _ := strconv.ParseInt(offStr, 0, 64)

		z := f.Addr().Interface()
		binary.Read(bytes.NewBuffer(raw[off:]), binary.LittleEndian, z)
	}

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
	off += ReadBlock(c, off, &f.Header)
	f.Cglx = make([]CGLX, f.Header.CglxCount-1)
	for i := 0; i < int(f.Header.CglxCount-1); i++ {
		off += ReadBlock(c, off, &f.Cglx[i])
	}
	trailer := CGLXTrailer{}
	off += ReadBlock(c, off, &trailer)
	f.Pages = make([]Page, f.Header.PageCount)
	for i := 0; i < int(f.Header.PageCount); i++ {
		off += ReadBlock(c, off, &f.Pages[i])
	}

	f.Lines = make([]Line, f.Header.LineCount)
	for i := 0; i < int(f.Header.LineCount); i++ {
		l := &f.Lines[i]
		off += ReadBlock(c, off, l)
		l.VarData = c[off:off+int(l.VarSize)]
		off += int(l.VarSize)
	}

	f.Measures = make([]Measure, f.Header.MeasureCount)
	for i := 0; i < int(f.Header.MeasureCount); i++ {
		m := &f.Measures[i]
		off += ReadBlock(c, off, m)
		m.VarData = c[off:off+int(m.VarSize)]
		off += int(m.VarSize)
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

	analyzeTags(content)

	d := &Data{}
	err = readData(content, d)
	if err != nil {
		log.Fatalf("readData %v", err)
	}
	log.Println("HEAD", &d.Header)
	analyzeMeas(d)
	mess(d)
}

func mess(d *Data) {
	raw := make([]byte, len(d.Raw))
	copy(raw, d.Raw)

	fmt.Println("meas", d.Measures[2].Raw[4:])
	fmt.Println(":58", d.Measures[2].VarData[:58])
	fmt.Println("last", d.Measures[2].VarData[309:])
	meas := d.Measures[2].VarData[58:]
	meas = meas[:28]
	for i, c := range meas {
		if i % 4 == 0 {
			fmt.Printf("\n")
		}
		fmt.Printf("%2d: %3d ", i, c)
	}
	fmt.Printf("\n")

	// 0: xoff, relative to measure start. 128 = full meas?
	// 1: 57 -> 255 = appears in first bar.
	//raw[1] = 255

	//raw[2] = 254

	// 3: step - 0 = central C.
	// 4 

	raw[3] = 17
	// 5: MIDI ? 
	// 6
	// 7
	// 8
	// 9
	// 10 
	// 11
	// chromatic step 1=sharp, 2=flat, 3=natural, 4=dsharp,
	// 5=dflat - used as offset in font. Using 6 gives a longa symbol
	//raw[11] = 0

	err := ioutil.WriteFile("mess.enc", raw, 0644)
	if err != nil {
		log.Fatalf("WriteFile:", err)
	}
}

func analyzeMeas(d *Data) {
	m1 := d.Measures[2].VarData
	m2 := d.Measures[10].VarData
	if len(m1) != len(m2) {
		log.Println("bah")
		return
	}

	n := 0
	for i, c := range m1 {
		if c != m2[i] {
			fmt.Println("diff", i, c, m2[i])
			n++
		}
	}
	fmt.Println("diffcnt", n)
	return
}
