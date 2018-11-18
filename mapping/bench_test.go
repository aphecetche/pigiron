package mapping_test

import (
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/aphecetche/pigiron/geo"
	"github.com/aphecetche/pigiron/mapping"
)

func BenchmarkSegmentationCreationPerDE(b *testing.B) {

	mapping.ForOneDetectionElementOfEachSegmentationType(func(detElemID int) {
		b.Run(strconv.Itoa(detElemID), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_ = mapping.NewSegmentation(detElemID, true)
				_ = mapping.NewSegmentation(detElemID, false)
			}
		})
	})
}

func BenchmarkSegmentationCreation(b *testing.B) {

	for i := 0; i < b.N; i++ {
		mapping.ForOneDetectionElementOfEachSegmentationType(func(detElemID int) {
			_ = mapping.NewSegmentation(detElemID, true)
			_ = mapping.NewSegmentation(detElemID, false)
		})
	}
}

type SegPair map[bool]mapping.Segmentation

type TestPoint struct {
	x, y float64
}

func generateUniformTestPoints(n int, box geo.BBox) []TestPoint {
	var testpoints = make([]TestPoint, n)
	for i := 0; i < n; i++ {
		x := box.Xmin() + rand.Float64()*box.Width()
		y := box.Ymin() + rand.Float64()*box.Height()
		testpoints[i] = TestPoint{x, y}
	}
	return testpoints
}

func BenchmarkPositions(b *testing.B) {
	mapping.ForOneDetectionElementOfEachSegmentationType(func(detElemID int) {
		const n = 100000
		for _, isBendingPlane := range []bool{true, false} {
			b.Run(fmt.Sprintf("findPadByPositions(%d,%v)", detElemID, isBendingPlane), func(b *testing.B) {
				seg := mapping.NewSegmentation(detElemID, isBendingPlane)
				bbox := mapping.ComputeBbox(seg)
				testpoints := generateUniformTestPoints(n, bbox)
				for i := 0; i < b.N; i++ {
					for _, tp := range testpoints {
						seg.FindPadByPosition(tp.x, tp.y)
					}
				}
			})
		}
	})
}

type DC struct {
	D int
	C int
}

var (
	detElemIds []int
)

func init() {

	detElemIds = []int{100, 300, 501, 1025}
}

func BenchmarkByFEE(b *testing.B) {
	for _, deid := range detElemIds {
		for _, isBendingPlane := range []bool{true, false} {
			planeName := "B"
			if isBendingPlane == false {
				planeName = "NB"
			}
			seg := mapping.NewSegmentation(deid, isBendingPlane)
			var dcs []DC
			seg.ForEachPad(func(paduid int) {
				dcs = append(dcs, DC{D: seg.PadDualSampaID(paduid), C: seg.PadDualSampaChannel(paduid)})
			})
			b.Run(strconv.Itoa(deid)+planeName, func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					for _, pad := range dcs {
						seg.FindPadByFEE(pad.D, pad.C)
					}
				}
			})
		}
	}
}