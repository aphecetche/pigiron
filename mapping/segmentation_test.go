package mapping_test

import (
	"fmt"
	"log"
	"math"
	"os"
	"testing"

	"github.com/mrrtf/pigiron/mapping"
	// must include the specific implementation package of the mapping
	_ "github.com/mrrtf/pigiron/mapping/impl4"
)

var testdeid = []mapping.DEID{100, 300, 500, 501, 502, 503, 504, 600, 601, 602, 700,
	701, 702, 703, 704, 705, 706, 902, 903, 904, 905}

func TestNewSegmentationMustNotErrorIfDetElemIdIsValid(t *testing.T) {
	cseg := mapping.NewSegmentation(100)
	if cseg == nil {
		t.Fatalf("Could not create segmentation")
	}
}

func TestNewSegmentationMustErrorIfDetElemIdIsNotValid(t *testing.T) {
	cseg := mapping.NewSegmentation(-1)
	if cseg != nil {
		t.Fatalf("Should have failed here")
	}
	cseg = mapping.NewSegmentation(121)
	if cseg != nil {
		t.Fatalf("Should have failed here")
	}
}

func TestCreateSegmentation(t *testing.T) {
	for _, de := range testdeid {
		cseg := mapping.NewSegmentation(de)
		if cseg == nil {
			t.Fatalf("could not create segmentation for DE %d", de)
		}
	}
}

func TestNofPads(t *testing.T) {
	var tv = []struct {
		de      mapping.DEID
		nofPads int
	}{
		{100, 14392 + 14280},
		{300, 13947 + 13986},
		{902, 4480 + 3136},
		{702, 4160 + 2912},
		{701, 4096 + 2880},
		{601, 3648 + 2560},
		{501, 3568 + 2496},
		{602, 3200 + 2240},
		{700, 3200 + 2240},
		{502, 3120 + 2176},
		{600, 3008 + 2112},
		{500, 2928 + 2048},
		{903, 2880 + 2016},
		{703, 2560 + 1792},
		{904, 2240 + 1568},
		{503, 1920 + 1344},
		{704, 1920 + 1344},
		{504, 1280 + 896},
		{905, 1280 + 896},
		{705, 960 + 672},
		{706, 640 + 448},
	}

	for _, tt := range tv {
		seg := mapping.NewSegmentation(tt.de)
		if seg.NofPads() != tt.nofPads {
			t.Errorf("DE %d : expected %d pads. Got %d", tt.de, seg.NofPads(), tt.nofPads)
		}
	}
}

func TestNofFEC(t *testing.T) {
	var tv = []struct {
		de            mapping.DEID
		nofDualSampas int
	}{
		{100, 226 + 225},
		{300, 221 + 222},
		{902, 70 + 50},
		{702, 65 + 46},
		{701, 64 + 46},
		{601, 57 + 40},
		{501, 56 + 39},
		{602, 50 + 35},
		{700, 50 + 36},
		{502, 49 + 34},
		{600, 47 + 33},
		{500, 46 + 32},
		{903, 45 + 33},
		{703, 40 + 29},
		{904, 35 + 26},
		{503, 30 + 21},
		{704, 30 + 22},
		{504, 20 + 14},
		{905, 20 + 16},
		{705, 15 + 12},
		{706, 10 + 8},
	}

	for _, tt := range tv {
		seg := mapping.NewSegmentation(tt.de)
		if seg.NofDualSampas() != tt.nofDualSampas {
			t.Errorf("DE %d : expected %d dual sampas. Got %d", tt.de, seg.NofDualSampas(), tt.nofDualSampas)
		}
	}
}

func TestNofPadsInSegmentations(t *testing.T) {
	npads := 0
	mapping.ForOneDetectionElementOfEachSegmentationType(func(deid mapping.DEID) {
		cseg := mapping.NewSegmentation(deid)
		if cseg == nil {
			log.Fatalf("Got nil seg for detElemId %d", deid)
		}
		npads += cseg.NofPads()
	})
	if npads != 143469 {
		t.Errorf("Expected 143469 pads : got %d", npads)
	}
}

func TestMustErrorIfDualSampaChannelIsNotBetween0And63(t *testing.T) {
	cseg := mapping.NewSegmentation(100)
	_, err := cseg.FindPadByFEE(102, -1)
	if err == nil {
		t.Errorf("Should _not_ get a valid pad here")
	}
	_, err = cseg.FindPadByFEE(102, 64)
	if err == nil {
		t.Errorf("Should _not_ get a valid pad here")
	}
}

func TestPositionOfOnePadInDE100Bending(t *testing.T) {
	seg := mapping.NewSegmentation(100)
	p1, err := seg.FindPadByFEE(76, 9)
	if err != nil {
		t.Errorf("Should get a valid pad: %v", err)
	}
	p2, _, err := seg.FindPadPairByPosition(1.575, 18.69)
	if err != nil {
		t.Errorf("Should get a valid pad: %v", err)
	}
	if p1 != p2 {
		t.Errorf("Should get the same pads here p1=%v p2=%v", p1, p2)
		PrintPad(os.Stdout, seg, p1)
		PrintPad(os.Stdout, seg, p2)
	}
}

func TestValidFindPadByFEE(t *testing.T) {
	seg := mapping.NewSegmentation(100)
	_, err := seg.FindPadByFEE(102, 3)
	if err != nil {
		t.Errorf("Should get a valid pad here")
	}
}

func TestInvalidFindPadByFEE(t *testing.T) {
	seg := mapping.NewSegmentation(100)
	_, err := seg.FindPadByFEE(214, 14)
	if err == nil {
		t.Errorf("Should not get a valid pad here")
	}
}

func TestForEachPad(t *testing.T) {
	mapping.ForOneDetectionElementOfEachSegmentationType(func(deid mapping.DEID) {
		seg := mapping.NewSegmentation(deid)
		npads := 0
		seg.ForEachPad(func(paduid mapping.PadUID) {
			npads++
		})
		if npads != seg.NofPads() {
			t.Errorf("DE %v expected %v pads but got %v from ForEachPad loop", deid, seg.NofPads(), npads)
		}
	})
}

func checkSameCathode(seg mapping.Segmentation, paduid mapping.PadUID, nei []int) bool {

	for _, n := range nei {
		if seg.IsBendingPad(mapping.PadUID(n)) != seg.IsBendingPad(paduid) {
			return false
		}
	}
	return true
}

func TestBothSideNeighbours(t *testing.T) {

	seg := mapping.NewSegmentation(100)

	pb, pnb, err := seg.FindPadPairByPosition(24.0, 24.0)
	if err != nil {
		t.Errorf("could not get pad for x=24 y=24")
	}
	nei := make([]int, 13)
	n := seg.GetNeighbourIDs(pb, nei)
	if !checkSameCathode(seg, pb, nei[:n]) {
		t.Errorf("Got NB pads as neighbours of a bending pad")
	}
	n = seg.GetNeighbourIDs(pnb, nei)
	if !checkSameCathode(seg, pnb, nei[:n]) {
		t.Errorf("Got B pads as neighbours of a non-bending pad")
	}
}

func TestCircularTest(t *testing.T) {
	seg := mapping.NewSegmentation(100)
	var tp = []struct {
		dsid mapping.DualSampaID
		dsch mapping.DualSampaChannelID
	}{
		{95, 45}, // both pads @pos 24.0,24.0 cm
		{1119, 45},
	}

	for i, _ := range tp {
		dsid := tp[i].dsid
		dsch := tp[i].dsch
		paduid, _ := seg.FindPadByFEE(dsid, dsch)
		if seg.PadDualSampaID(paduid) != dsid {
			t.Errorf("Wanted DSID=%d. Got %d", dsid, seg.PadDualSampaID(paduid))
		}
		if seg.PadDualSampaChannel(paduid) != dsch {
			t.Errorf("Wanted DSCH=%d. Got %d", dsid, seg.PadDualSampaChannel(paduid))
		}
	}
}

func TestForEachPadAndPadIndexRange(t *testing.T) {
	npads := 0
	mapping.ForOneDetectionElementOfEachSegmentationType(func(deid mapping.DEID) {
		n := 0
		var pmin mapping.PadUID = math.MaxInt32
		var pmax mapping.PadUID
		seg := mapping.NewSegmentation(deid)
		seg.ForEachPad(func(paduid mapping.PadUID) {
			n++
			npads++
			if paduid < pmin {
				pmin = paduid
			}
			if paduid > pmax {
				pmax = paduid
			}
		})
		if n != seg.NofPads() {
			t.Errorf("DE %4d Want %d pads - Got %d", deid, seg.NofPads(), n)
		}
		if pmin != 0 {
			t.Errorf("DE %4d Want pmin=0- Got %d", deid, pmin)
		}
		if pmax != mapping.PadUID(seg.NofPads()-1) {
			t.Errorf("DE %4d Want pmax=%d- Got %d", deid, seg.NofPads()-1, pmax)
		}
	})
	fmt.Println("Looped over", npads, "pads")
}

func TestAllNeighbours(t *testing.T) {
	nei := make([]int, 13)
	nnei := make(map[int]int) // number of pads with a given number of neighbours
	npads := 0
	// mapping.ForOneDetectionElementOfEachSegmentationType(func(deid mapping.DEID) {
	mapping.ForEachDetectionElement(func(deid mapping.DEID) {
		seg := mapping.NewSegmentation(deid)
		seg.ForEachPad(func(paduid mapping.PadUID) {
			npads++
			n := seg.GetNeighbourIDs(paduid, nei)
			nnei[n]++
		})
	})
	fmt.Println("npads=", npads, nnei)
}
