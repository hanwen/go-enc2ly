package encore

import (
	"fmt"
)


type Data struct {
	Raw      []byte
	Header   Header
	Staff    []*Staff
	Pages    []*Page
	Lines    []*Line
	Measures []*Measure
}

type Header struct {
	Offset int
	Raw    []byte `want:"SCOW" fixed:"194"`

	LineCount      int16 `offset:"0x2e"`
	PageCount      int16 `offset:"0x30"`
	StaffCount     byte  `offset:"0x32"`
	StaffPerSystem byte  `offset:"0x33"`
	MeasureCount   int16 `offset:"0x34"`
}

type Line struct {
	Id      int
	Offset  int
	Raw     []byte `want:"LINE" fixed:"8"`
	VarSize uint32 `offset:"0x4"`
	VarData []byte
	LineData
	Staffs []*LineStaffData

	StaffMap map[int]*LineStaffData
}

type Page struct {
	Id     int
	Offset int
	Raw    []byte `want:"PAGE" fixed:"34"`
}

type LineStaffData struct {
	Id        int
	Clef      byte `offset:"1"`
	Key       byte `offset:"2"`
	PageIdx   byte `offset:"3"`
	StaffType byte `offset:"7"`
	StaffIdx  byte `offset:"8"`

	Line *Line
}

type LineData struct {
	Start        uint16 `offset:"10"`
	MeasureCount byte   `offset:"12"`
}

type Measure struct {
	Id     int
	Offset int
	Raw    []byte `want:"MEAS" fixed:"62"`

	VarSize           int32  `offset:"4"`
	Bpm               uint16 `offset:"8"`
	TimeSigGlyph      byte   `offset:"10"`
	BeatTicks         uint16 `offset:"12"`
	DurTicks          uint16 `offset:"14"`
	TimeSigNum        byte   `offset:"16"`
	TimeSigDen        byte   `offset:"17"`
	BarTypeStart      byte   `offset:"20"`
	BarTypeEnd        byte   `offset:"21"`
	RepeatMarker      byte   `offset:"22"`
	RepeatAlternative byte   `offset:"23"`
	Coda              uint32 `offset:"33"`

	VarData []byte

	Elems   []*MeasElem
	AbsTick int
}

func (m *Measure) TimeSignature() string {
	return fmt.Sprintf("%d/%d", m.TimeSigNum, m.TimeSigDen)
}

type Staff struct {
	Id     int
	Offset int

	// Sometimes TK00, sometimes TK01
	Raw     []byte `fixed:"242"`

	Name [10]byte `offset:"8"`

	// 174, 175, 

	// In semitones; b-flat clar = -2
	Transposition int8 `offset:"165"`

	// 0 = G, 1 = F, 2 = C(middle), 3=C(tenor), 4=G^8, 5=G_8,
	// 6=F_8
	Clef byte `offset:"172"`

	// 181 = 1 for piano staff. ?

	// 180 - 187: MIDI channel (repeated?)
	// 188 - 195: MIDI program (repeated?)
	// 196 - 203: MIDI volume (repeated?)

	// 164 ?

	// 205 ?
}

type MeasElemSpecific interface {
	GetDurationTick() int
	GetTypeName() string
}

type NoDuration struct{}

func (n *NoDuration) GetDurationTick() int {
	return 0
}

type MeasElem struct {
	Raw    []byte
	Offset int
	// Relative to measure start.
	Tick uint16 `offset:"0"`

	// type << 4 | voice
	TypeVoice byte `offset:"2"`
	Size      byte `offset:"3"`
	StaffIdx  byte `offset:"4"`

	TypeSpecific MeasElemSpecific

	Measure       *Measure
	Staff         *Staff
	LineStaffData *LineStaffData
}

const (
	TYPE_NONE      = iota // 0 
	TYPE_CLEF      = 1
	TYPE_KEYCHANGE = 2
	TYPE_TIE       = 3
	TYPE_BEAM      = 4
	TYPE_ORNAMENT  = 5
	TYPE_LYRIC     = 6
	TYPE_CHORD     = 7
	TYPE_REST      = 8
	TYPE_NOTE      = 9
)

func (n *MeasElem) AbsTick() int {
	return int(n.Tick) + n.Measure.AbsTick
}

func (n *MeasElem) GetTypeName() string {
	return n.TypeSpecific.GetTypeName()
}

func (n *MeasElem) GetDurationTick() int {
	return n.TypeSpecific.GetDurationTick()
}

func (n *MeasElem) Voice() int {
	return int(n.TypeVoice & 0xf)
}

func (n *MeasElem) GetRaw() []byte {
	return n.Raw
}

func (n *MeasElem) GetTick() int {
	return int(n.Tick)
}

func (n *MeasElem) Type() int {
	return int(n.TypeVoice) >> 4
}

func (n *MeasElem) Sz() int {
	return len(n.Raw)
}

func (n *MeasElem) GetStaff() int {
	return int(n.StaffIdx & 63)
}

func (n *MeasElem) GetOffset() int {
	return int(n.Offset)
}

type Note struct {
	WithDuration

	// must use masking?
	Grace   byte `offset:"6"`
	XOffset byte `offset:"10"`

	// ledger below staff = 0; top line = 10
	Position int8 `offset:"12"`

	// Does not include staff wide transposition setting; 60 = central C.
	SemitonePitch byte `offset:"15"`

	PlaybackDurationTicks uint16 `offset:"16"`

	// Not sure - but encore defaults to 64; and all have this?
	Velocity byte `offset:"19"`

	// 128 = stem-down bit
	// 7 = unbeamed?
	Options byte `offset:"20"`

	// 1=sharp, 2=flat, 3=natural, 4=dsharp, 5=dflat
	// used as offset in font. Using 6 gives a longa symbol
	// alteration is in low nibble.
	AlterationGlyph byte `offset:"21"`

	ArticulationUp   byte `offset:"24"`
	ArticulationDown byte `offset:"26"`
}

func (n *Note) Alteration() int {
	switch n.AlterationGlyph {
	case 1:
		return 1
	case 2:
		return -1
	case 3:
		return 0
	case 4:
		return 2
	case 5:
		return -2
	}
	return 0
}

func (o *Note) GetTypeName() string {
	return "Note"
}

type Slur struct {
	NoDuration

	// 33 = slur, 16=8va, ... ?
	SlurType       byte `offset:"5"`
	LeftX          byte `offset:"10"`
	LeftPosition   byte `offset:"12"`
	MiddleX        byte `offset:"14"`
	MiddlePosition byte `offset:"16"`
	MeasureDelta   byte `offset:"18"`
	RightX         byte `offset:"20"`
	RightPosition  byte `offset:"22"`
}

func (o *Slur) GetTypeName() string {
	return "Slur"
}

type KeyChange struct {
	NoDuration
	NewKey byte `offset:"5"`
	OldKey byte `offset:"10"`
}

func (o *KeyChange) GetTypeName() string {
	return "KeyChange"
}

type Other struct {
	NoDuration
}

func (o *Other) GetTypeName() string {
	return "Other"
}

type Script struct {
	NoDuration
	XOff byte `offset:"10"`
}

func (o *Script) GetTypeName() string {
	return "Script"
}

type Clef struct {
	NoDuration
	ClefType byte `offset:"5"`
	XOff     byte `offset:"10"`
}

func (o *Clef) GetTypeName() string {
	return "Clef"
}

// Also used for tuplet bracket.
type SubBeam struct {
	// maybe uint16 ?
	StartX byte `offset:"0"`
	EndX   byte `offset:"2"`
}

type Beam struct {
	NoDuration

	// The following fall are only populated in the first subbeam
	LeftPos      int8   `offset:"18"`
	RightPos     int8   `offset:"19"`
	EndNoteTick  uint16 `offset:"20"`
	TupletNumber byte   `offset:"23"`

	// base size: 14, 16 bytes per beam (16th: 46 bytes)
	SubBeams []SubBeam
}

func (o *Beam) GetTypeName() string {
	return "Beam"
}

type WithDuration struct {
	// 4 = 8th, 3=quarter, 2=half, etc.
	//
	// hi nibble has notehead type.
	FaceValue byte `offset:"5"`
	// 50 = (3 << 4) | 2 => 2/3 for triplet.
	Tuplet byte `offset:"13"`
	// & 0x3: dotcount; &0x4: vertical dot position.
	DotControl            byte   `offset:"14"`
	PlaybackDurationTicks uint16 `offset:"16"`
}

func (w *WithDuration) TupletDen() int {
	return int(w.Tuplet >> 4)
}

func (w *WithDuration) TupletNum() int {
	return int(w.Tuplet & 0xf)
}

func (w *WithDuration) GetDurationTick() int {
	num := 1
	den := 1
	diff := w.DurationLog()
	if diff >= 0 {
		den <<= uint(diff)
	} else {
		num <<= uint(-diff)
	}

	if w.DotControl&0x3 == 1 {
		// todo double dot
		num *= 3
		den *= 2
	}
	if w.Tuplet != 0 {
		num *= int(w.Tuplet & 0xf)
		den *= int(w.Tuplet >> 4)
	}

	// 16 16ths to the wholes, 60 ticks per 16th.
	num *= 60 * 16

	return num / den
}

func (w *WithDuration) DurationLog() int {
	return int(w.FaceValue&0xf) - 1
}

type Rest struct {
	WithDuration
	XOffset  byte `offset:"10"`
	Position int8 `offset:"12"`
}

func (o *Rest) GetTypeName() string {
	return "Rest"
}

type Tie struct {
	NoDuration
	// offset: 5 - vertical, staff ?
	// 4=>whole, 7 => 8th -> ? 
	LeftDurationType byte `offset:"5"`

	// offset: 6 - to left/to right ? Bitfield?
	XOffset byte `offset:"10"`
	// 11: visibility?  Affects left note too.

	// position/pitch of left note?
	NotePosition byte `offset:"12"`

	// 13: causes 0 symbol to be printed.
	// position/pitch of curve
	TiePosition byte `offset:"14"`
}

func (o *Tie) GetTypeName() string {
	return "Tie"
}
