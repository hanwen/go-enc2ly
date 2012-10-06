package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"

	"go-enc2ly/encore"
)

func analyze(d *encore.Data) {
	//	analyzeTags(content)
	//	Convert(d)
	//	analyzeStaff(d)
	//	messM(d)
	//mess(d)
	//	analyzeKeyCh(d)
	//	analyzeAll(d)
	//	analyzeStaff(d)
	//	analyzeMeasStaff(d)
	analyzeMeas(d)
	//		analyzeStaffdata(d)
	//		analyzeStaffHeader(d)	
	//	analyzeLine(d)
	//	analyzeBeam(d)	
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
			if idx := bytes.Index(sectionContent, want); idx > 0 {
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

func analyzeLine(d *encore.Data) {
	for i, l := range d.Lines {
		fmt.Printf("%v\n", l)
		fmt.Printf("linesize %d %v\n", i, l.VarSize)
		fmt.Printf(" %+v, %+v\n", l.LineData, l.Staffs)
	}
}

func analyzeAll(d *encore.Data) {
	for i, m := range d.Measures[:2] {
		fmt.Printf("meas %d\n", i)
		for _, e := range m.Elems {
			fmt.Printf("%+v\n", e)
		}
	}
}

func analyzeStaff(d *encore.Data) {
	for _, m := range d.Measures {
		for _, e := range m.Elems {
			if e.GetStaff() == 0 && e.GetTypeName() == "Note" {
				fmt.Printf("%+v\n", e)
			}
		}
	}
}

func analyzeMeas(d *encore.Data) {
	for i, m := range d.Measures {
		fmt.Printf("meas %d: rep %d rep %d volta %x\n", i, m.RepeatMarker, m.RepeatAlternative, m.Coda)
	}
}

func analyzeMeasStaff(d *encore.Data) {
	for _, e := range d.Measures[3].Elems {
		if e.GetStaff() == 0 {
			fmt.Printf("%+v\n", e)
		}
	}
}

func analyzeKeyCh(d *encore.Data) {
	for i, m := range d.Measures {
		for j, e := range m.Elems {
			if e.Type() == encore.TYPE_KEYCHANGE {
				log.Printf("meas %d elt %d staff %d", i, j,
					e.GetStaff())
			}
		}
	}
}

func analyzeStaffdata(d *encore.Data) {
	for i, s := range d.Staff {
		fmt.Printf("%d %+v\n", i, s)
	}
}

func analyzeBeam(d *encore.Data) {
	for _, m := range d.Measures {
		for _, e := range m.Elems {
			if e.Type() == encore.TYPE_BEAM {
				fmt.Printf("meas %d %+v %+v\n", m.Id, e, e.TypeSpecific)

			}
		}
	}
}

func messM(d *encore.Data) {
	raw := make([]byte, len(d.Raw))
	copy(raw, d.Raw)

	encore.ReadData(raw)

	err := ioutil.WriteFile("mess.enc", raw, 0644)
	if err != nil {
		log.Fatalf("WriteFile:", err)
	}

}

func mess(d *encore.Data) {
	fmt.Printf("mess\n")
	for i := 0; i < 13; i++ {
		raw := make([]byte, len(d.Raw))
		copy(raw, d.Raw)

		for _, m := range d.Measures[5:6] {
			for _, e := range m.Elems {
				if e.GetStaff() == 6 {
					raw[e.GetOffset()+5+i] += 3
				}
			}
		}

		encore.ReadData(raw)
		fmt.Printf("messed\n")

		err := ioutil.WriteFile(fmt.Sprintf("mess%d.enc", i), raw, 0644)
		if err != nil {
			log.Fatalf("WriteFile:", err)
		}
	}
}

func isH(x []byte) bool {
	for i := 0; i < 4; i++ {
		if !(('0' <= x[i] && x[i] <= '9') ||
			('A' <= x[i] && x[i] <= 'Z')) {
			return false
		}
	}
	return true
}

func dumpBytes(d []byte) {
	for i, c := range d {
		fmt.Printf("%5d: %3d", i, c)
		if i%4 == 3 && i > 0 {
			fmt.Printf("\n")
		}
	}
	fmt.Printf("\n")
}
