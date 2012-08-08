package encore

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"reflect"
	"strconv"
)

func (l *Line) lineReadStaffs() {
	d := l.VarData[26:]
	if len(d)%30 != 0 {
		log.Fatalf("must be multiple of 30: %d", len(d))
	}
	i := 0
	for len(d) > 0 {
		staffRaw := d[:30]
		d = d[30:]
		i++
		lsd := &LineStaffData{}
		fillFields(staffRaw, lsd)
		l.Staffs = append(l.Staffs, lsd)
	}
}

func readElem(c []byte, off int) (result *MeasElem) {
	result = new(MeasElem)
	fillBlock(c, off, result)

	var e MeasElemSpecific
	switch result.Type() {
	case TYPE_CLEF:
		e = &Clef{}
	case TYPE_KEYCHANGE:
		e = &KeyChange{}
	case TYPE_TIE:
		e = &Tie{}
	case TYPE_BEAM:
		e = &Beam{}
	case TYPE_ORNAMENT:
		switch c[3] {
		case 16:
			e = &Script{}
		case 86:
			e = &Other{}
		case 28:
			e = &Slur{}
		}
	case TYPE_REST:
		e = &Rest{}
	case TYPE_NOTE:
		e = &Note{}
	default:
		e = &Other{}
	}

	result.TypeSpecific = e
	fillFields(c, e)
	if result.Type() == TYPE_BEAM {
		b := e.(*Beam)
		b.SubBeams = make([]SubBeam, (result.Size-14)/16)
		for i := range b.SubBeams {
			fillFields(result.Raw[14+16*i:], &b.SubBeams[i])
		}
	}
	return result
}

var endMarker = string([]byte{255, 255})

func (m *Measure) readElems() {
	r := m.VarData
	off := m.Offset + 62 // todo - extract.
	for len(r) >= 3 {
		if string(r[:2]) == endMarker {
			break
		}
		sz := int(r[3])
		if sz < 3 {
			log.Fatalf("got sz %d: %q, left %d bytes", sz, r[:10], len(r))
		}

		m.Elems = append(m.Elems, readElem(r[:sz], off))
		r = r[sz:]
		off += sz
	}

	if string(r) != endMarker {
		log.Printf("end marker not found: have %q", r)
	}
}

// Like fillFields, but fill Raw/Offset too.
func fillBlock(raw []byte, offset int, dest interface{}) {
	v := reflect.ValueOf(dest).Elem()
	byteOffAddr := v.FieldByName("Offset").Addr().Interface().(*int)
	*byteOffAddr = offset

	rawAddr := v.FieldByName("Raw").Addr().Interface().(*[]byte)
	*rawAddr = raw

	fillFields(raw, dest)
}

func fillFields(raw []byte, dest interface{}) {
	v := reflect.ValueOf(dest).Elem()
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).Anonymous {
			fillFields(raw, v.Field(i).Addr().Interface())
			continue
		}
		f := v.Field(i)
		offStr := v.Type().Field(i).Tag.Get("offset")
		if offStr == "" {
			continue
		}

		off, _ := strconv.ParseInt(offStr, 0, 64)
		if off >= int64(len(raw)) {
			continue
		}

		z := f.Addr().Interface()
		binary.Read(bytes.NewBuffer(raw[off:]), binary.LittleEndian, z)
	}
}

func readTaggedBlock(c []byte, off int, dest interface{}) int {
	v := reflect.ValueOf(dest).Elem()

	tagField, ok := v.Type().FieldByName("Raw")
	if !ok {
		log.Fatalf("missing Raw in %T", dest)
	}

	fixed := tagField.Tag.Get("fixed")
	sz, _ := strconv.ParseInt(fixed, 0, 64)
	rawAddr := v.FieldByName("Raw").Addr().Interface().(*[]byte)
	raw := c[off : off+int(sz)]
	*rawAddr = raw
	
	want := tagField.Tag.Get("want")
	if want != "" && string(raw[:len(want)]) != want {
		log.Fatalf("Got tag %q want %q - %q", raw[:len(want)], want, raw)
	}
	fillBlock(raw, off, dest)
	return int(sz)
}

func (h *Header) String() string {
	return fmt.Sprintf("Systems %d PAGE %d Staff %d staffpersys %d MEAS %d",
		h.LineCount, h.PageCount, h.StaffCount, h.StaffPerSystem,
		h.MeasureCount)
}

func ReadData(c []byte) (*Data, error) {
	f := new(Data)
	f.Raw = c
	off := 0
	off += readTaggedBlock(c, off, &f.Header)
	f.Staff = make([]*Staff, f.Header.StaffCount)
	for i := 0; i < len(f.Staff); i++ {
		s := new(Staff)
		f.Staff[i] = s
		s.Id = i
		off += readTaggedBlock(c, off, s)
	}

	f.Pages = make([]*Page, f.Header.PageCount)
	for i := 0; i < int(f.Header.PageCount); i++ {
		p := new(Page)
		p.Id = i
		off += readTaggedBlock(c, off, p)
		f.Pages[i] = p
	}

	f.Lines = make([]*Line, f.Header.LineCount)
	for i := 0; i < int(f.Header.LineCount); i++ {
		l := new(Line)
		l.Id = i
		f.Lines[i] = l
		off += readTaggedBlock(c, off, l)
		l.VarData = c[off : off+int(l.VarSize)]
		off += int(l.VarSize)
		fillFields(l.VarData, &l.LineData)
		l.lineReadStaffs()
	}

	f.Measures = make([]*Measure, f.Header.MeasureCount)
	for i := 0; i < int(f.Header.MeasureCount); i++ {
		m := new(Measure)
		m.Id = i
		f.Measures[i] = m
		off += readTaggedBlock(c, off, m)
		m.VarData = c[off : off+int(m.VarSize)]
		off += int(m.VarSize)
		m.readElems()
	}

	setLinks(f)
	return f, nil
}

func setLinks(d *Data) {
	systemIdx := 0
	for _, l := range d.Lines {
		for _, s := range l.Staffs {
			s.Line = l
		}
	}
	var abs int
	for i, m := range d.Measures {
		for int(d.Lines[systemIdx].LineData.Start)+int(d.Lines[systemIdx].LineData.MeasureCount) < i {
			systemIdx++
		}
		for _, e := range m.Elems {
			e.Measure = m
			e.Staff = d.Staff[e.GetStaff()]
			e.LineStaffData = d.Lines[systemIdx].Staffs[e.GetStaff()]
		}
		m.AbsTick = abs
		abs += int(m.DurTicks)
	}
}
