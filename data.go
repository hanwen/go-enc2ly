package main

import (
	"fmt"
)

type Line struct {
	Offset int
	Raw     []byte `want:"LINE" fixed:"8"`
	VarSize uint32  `offset:"0x4"`
	VarData []byte
	LineData
	Staffs  []*LineStaffData
}

type Data struct {
	Raw      []byte
	Header   Header
	Staff    []*Staff
	Pages    []*Page
	Lines    []*Line
	Measures []*Measure
}

type Header struct {
	Offset     int
	Raw []byte `want:"SCOW" fixed:"436"`

	LineCount      int16 `offset:"0x2e"`
	PageCount      int16 `offset:"0x30"`
	StaffCount     byte  `offset:"0x32"`
	StaffPerSystem byte  `offset:"0x33"`
	MeasureCount   int16 `offset:"0x34"`
}

type Page struct {
	Offset int
	Raw []byte `want:"PAGE" fixed:"34"`
}

type LineStaffData struct {
	Clef byte `offset:"1"`
	Key  byte `offset:"2"`
	PageIdx byte `offset:"3"`
	StaffType byte `offset:"7"`
	StaffIdx byte `offset:"8"`
}

type LineData struct {
	MeasStart uint16 `offset:"10"` 
	MeasureCount byte `offset:"12"`
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
	BarTypeStart byte `offset:"20"`
	BarTypeEnd byte `offset:"21"`
	VarData []byte
	
	Elems   []MeasElem
}

func (m *Measure) TimeSignature() string {
	return fmt.Sprintf("%d/%d", m.TimeSigNum, m.TimeSigDen)
}

type Staff struct {
	Offset int

	// Sometimes TK00, sometimes TK01
	Raw []byte `want:"TK0" fixed:"8"`
	VarSize uint32 `offset:"4"`
	VarData []byte

	StaffData
}

type StaffData struct {
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

type MeasElem interface {
	GetTick() int
	GetDurationTick() int
	GetRaw() []byte
	GetStaff() int
	GetOffset() int
	Sz() int
	GetType() int
	GetTypeName() string
	Voice() int
}

// Voice (1-8) should be somewhere too.
type MeasElemBase struct {
	Raw []byte
	Offset int
	Tick  uint16 `offset:"0"`

	// type << 4 | voice
	Type  byte `offset:"2"`
	Size  byte `offset:"3"`
	Staff byte `offset:"4"`
}

func (n *MeasElemBase) Voice() int {
	return int(n.Type & 0xf)
}


func (n *MeasElemBase) GetDurationTick() int {
	return 0
}

func (n *MeasElemBase) GetRaw() []byte {
	return n.Raw
}

func (n *MeasElemBase) GetTick() int {
	return int(n.Tick)
}

func (n *MeasElemBase) GetType() int {
	return int(n.Type) >> 4
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
	// 4 = 8th, 3=quarter, 2=half, etc.
	//
	// hi nibble has notehead type.
	FaceValue     byte `offset:"5"`

	// must use masking?
	Grace  byte `offset:"6"`
	XOffset       byte `offset:"10"`

	// ledger below staff = 0; top line = 10
	Position        int8 `offset:"12"`

	// 50 = (3 << 4) | 2 => 2/3 for triplet.
	Tuplet  byte  `offset:"13"`

	// & 0x3: dotcount; &0x4: vertical dot position.
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
	// alteration is in low nibble.
	AlterationGlyph byte `offset:"21"`

	ArticulationUp byte `offset:"24"`
	ArticulationDown byte `offset:"26"`
}

func (n *Note) GetDurationTick() int {
	return int(n.DurationTicks)
}

func (n *Note) DurationLog() int {
	return int(n.FaceValue & 0xf) - 1
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

type Clef struct {
	MeasElemBase
	ClefType byte `offset:"5"`
	XOff byte `offset:"10"`
}

func (o *Clef) GetTypeName() string {
	return "Clef"
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

	// see Note for more explanation. 
	FaceValue     byte `offset:"5"`
	XOffset    byte `offset:"10"`
	Position int8 `offset:"12"`
	Tuplet   byte `offset:"13"`
	DotControl byte `offset:"14"`
	DurationTicks   uint16 `offset:"16"`
}

func (n *Rest) GetDurationTick() int {
	return int(n.DurationTicks)
}

func (o *Rest) GetTypeName() string {
	return "Rest"
}

type Tie struct {
	MeasElemBase
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
