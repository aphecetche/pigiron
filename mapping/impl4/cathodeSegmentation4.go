package impl4

import (
	"fmt"
	"io"
	"log"
	"math"
	"sort"
	"strconv"

	"github.com/mrrtf/pigiron/geo"
	"github.com/mrrtf/pigiron/mapping"
)

// InvalidPadCID is a special integer that denotes a non existing pad
const invalidPadCID mapping.PadCID = -1

type cathodeSegmentation4 struct {
	segType                      int
	isBendingPlane               bool
	padGroups                    []padGroup
	padGroupTypes                []padGroupType
	padSizes                     []padSize
	dsids                        []mapping.DualSampaID
	dsidmap                      map[mapping.DualSampaID]int
	dualSampaPadCIDs             [][]mapping.PadCID
	padGroupIndex2PadCIDIndex    []int
	padcid2PadGroupTypeFastIndex []int
	padcid2PadGroupIndex         []int
	grid                         padGroupGrid
	deid                         mapping.DEID
}

func (seg *cathodeSegmentation4) DetElemID() mapping.DEID {
	return seg.deid
}

func (seg *cathodeSegmentation4) NofDualSampas() int {
	return len(seg.dsids)
}

func (cseg *cathodeSegmentation4) String(padcid mapping.PadCID) string {
	return fmt.Sprintf("FEC %4d CH %2d X %7.3f Y %7.3f SX %7.3f SY %7.3f",
		cseg.PadDualSampaID(padcid), cseg.PadDualSampaChannel(padcid),
		cseg.PadPositionX(padcid), cseg.PadPositionY(padcid),
		cseg.PadSizeX(padcid), cseg.PadSizeY(padcid))
}

func (seg *cathodeSegmentation4) Print(out io.Writer) {
	fmt.Fprintf(out, "segmentation3 has %v dual sampa ids = %v\n", len(seg.dsids), seg.dsids)
	fmt.Fprintf(out, "cells=%v\n", seg.grid.cells)
	for c := range seg.grid.cells {
		fmt.Fprintf(out, "%v ", seg.padGroups[c].fecID)
	}
	fmt.Fprintf(out, "\n")
	seg.grid.Print(out)
}

func newCathodeSegmentation(deid mapping.DEID, segType int, isBendingPlane bool, padGroups []padGroup,
	padGroupTypes []padGroupType, padSizes []padSize) *cathodeSegmentation4 {
	seg := &cathodeSegmentation4{
		segType:        segType,
		isBendingPlane: isBendingPlane,
		padGroups:      padGroups,
		padGroupTypes:  padGroupTypes,
		padSizes:       padSizes,
		deid:           deid}
	uniq := make(map[mapping.DualSampaID]struct{})
	var empty struct{}
	for i := range padGroups {
		uniq[padGroups[i].fecID] = empty
	}
	seg.init()
	seg.dualSampaPadCIDs = make([][]mapping.PadCID, len(uniq))
	seg.dsidmap = make(map[mapping.DualSampaID]int, len(uniq))
	seg.dsids = make([]mapping.DualSampaID, len(uniq))
	i := 0
	// sort the dual sampa ids to get them always ordered the
	// same way (not strictly necessary but helps comparisons
	// sometimes)
	keys := make([]int, len(uniq))
	for k := range uniq {
		keys[i] = int(k)
		i++
	}
	i = 0
	sort.Ints(keys)
	for _, k := range keys {
		dsid := mapping.DualSampaID(k)
		seg.dsids[i] = dsid
		seg.dsidmap[dsid] = i
		seg.dualSampaPadCIDs[i] = append(seg.dualSampaPadCIDs[i], seg.createPadCIDs(dsid)...)
		i++
	}
	return seg
}

func (seg *cathodeSegmentation4) init() {
	// here must make two loops
	// first one to fill in the 3 index slices
	// - padGroupIndex2PadCIDIndex
	// - padUID2PadGroupIndex
	// - padUID2PadGroupTypeFastIndex
	// then compute the global x,y ranges to be able to compute a grid
	// then loop over to put each cell (a cell is a pair (box,paduid))
	// within the correct grid cellSlice

	seg.fillIndexSlices()
	bbox := mapping.ComputeBBox(seg)
	seg.fillGrid(bbox)
}

func (seg *cathodeSegmentation4) fillIndexSlices() {
	padcid := 0
	for padGroupIndex := range seg.padGroups {
		seg.padGroupIndex2PadCIDIndex = append(seg.padGroupIndex2PadCIDIndex, padcid)
		pg := seg.padGroups[padGroupIndex]
		pgt := seg.padGroupTypes[pg.padGroupTypeID]
		for ix := 0; ix < pgt.NofPadsX; ix++ {
			for iy := 0; iy < pgt.NofPadsY; iy++ {
				if pgt.idByIndices(ix, iy) >= 0 {
					seg.padcid2PadGroupIndex = append(seg.padcid2PadGroupIndex, padGroupIndex)
					seg.padcid2PadGroupTypeFastIndex = append(seg.padcid2PadGroupTypeFastIndex, pgt.fastIndex(ix, iy))
					padcid++
				}
			}
		}
	}
}

func (seg *cathodeSegmentation4) padGroupBox(i int) geo.BBox {
	pg := seg.padGroups[i]
	pgt := seg.padGroupTypes[pg.padGroupTypeID]
	dx := seg.padSizes[pg.padSizeID].x * float64(pgt.NofPadsX)
	dy := seg.padSizes[pg.padSizeID].y * float64(pgt.NofPadsY)
	x := pg.x
	y := pg.y
	box, err := geo.NewBBox(x, y, x+dx, y+dy)
	if err != nil {
		panic(err)
	}
	return box
}

func (seg *cathodeSegmentation4) fillGrid(bbox geo.BBox) {
	// for each cell in the grid we find out which
	// padgroups have their bounding box intersecting with
	// the cell bounding box and insert them in that cell
	// if the intersect is not nil

	seg.grid = *(newPadGroupGrid(bbox))
	for index := range seg.grid.cells {
		cbox := seg.grid.cellBox(index)
		for i := range seg.padGroups {
			pbox := seg.padGroupBox(i)
			_, err := geo.Intersect(cbox, pbox)
			if err == nil {
				seg.grid.cells[index] = append(seg.grid.cells[index], i)
			}
		}
	}
}

func (seg *cathodeSegmentation4) createPadCIDs(dsid mapping.DualSampaID) []mapping.PadCID {
	pi := make([]mapping.PadCID, 0, 64)
	for pgi, pg := range seg.padGroups {
		if pg.fecID == dsid {
			pgt := seg.padGroupTypes[pg.padGroupTypeID]
			i1 := seg.padGroupIndex2PadCIDIndex[pgi]
			for i := i1; i < i1+pgt.NofPads; i++ {
				pi = append(pi, mapping.PadCID(i))
			}
		}
	}
	return pi
}

func (seg *cathodeSegmentation4) getDualSampaIndex(dsid mapping.DualSampaID) int {
	i, ok := seg.dsidmap[dsid]
	if ok == false {
		panic("should always find our ids within this internal code! dsid " + strconv.Itoa(int(dsid)) + " not found")
	}
	return i
}

func (seg *cathodeSegmentation4) getPadCIDs(dsid mapping.DualSampaID) []mapping.PadCID {
	dsIndex := seg.getDualSampaIndex(dsid)
	return seg.dualSampaPadCIDs[dsIndex]
}

func (seg *cathodeSegmentation4) DualSampaID(dualSampaIndex int) (mapping.DualSampaID, error) {
	if dualSampaIndex >= len(seg.dsids) {
		return -1, fmt.Errorf("Incorrect dualSampaIndex %d (should be within 0-%d range", dualSampaIndex,
			len(seg.dsids))
	}
	return seg.dsids[dualSampaIndex], nil
}

func (seg *cathodeSegmentation4) NofPads() int {
	n := 0
	for i := 0; i < seg.NofDualSampas(); i++ {
		dsid, err := seg.DualSampaID(i)
		if err != nil {
			log.Fatalf("Could not get mapping.DualSampaID for i=%d", i)
		}
		n += len(seg.getPadCIDs(dsid))
	}
	return n
}

func (seg *cathodeSegmentation4) ForEachPadInDualSampa(dsid mapping.DualSampaID, padHandler func(padcid mapping.PadCID)) {
	for _, padcid := range seg.getPadCIDs(dsid) {
		padHandler(padcid)
	}
}

func (seg *cathodeSegmentation4) PadDualSampaChannel(padcid mapping.PadCID) mapping.DualSampaChannelID {
	return seg.padGroupType(padcid).idByFastIndex(seg.padcid2PadGroupTypeFastIndex[padcid])
}

func (seg *cathodeSegmentation4) PadDualSampaID(padcid mapping.PadCID) mapping.DualSampaID {
	return seg.padGroup(padcid).fecID
}

func (seg *cathodeSegmentation4) PadSizeX(padcid mapping.PadCID) float64 {
	return seg.padSizes[seg.padGroup(padcid).padSizeID].x
}
func (seg *cathodeSegmentation4) PadSizeY(padcid mapping.PadCID) float64 {
	return seg.padSizes[seg.padGroup(padcid).padSizeID].y
}

func (seg *cathodeSegmentation4) IsValid(padcid mapping.PadCID) bool {
	return padcid != invalidPadCID
}

func (seg *cathodeSegmentation4) FindPadByFEE(dsid mapping.DualSampaID, dualSampaChannel mapping.DualSampaChannelID) (mapping.PadCID, error) {
	for _, padcid := range seg.getPadCIDs(dsid) {
		if seg.padGroupType(padcid).idByFastIndex(seg.padcid2PadGroupTypeFastIndex[padcid]) == dualSampaChannel {
			return padcid, nil
		}
	}
	return invalidPadCID, mapping.ErrInvalidPadCID
}

func (seg *cathodeSegmentation4) padGroup(padcid mapping.PadCID) *padGroup {
	return &seg.padGroups[seg.padcid2PadGroupIndex[padcid]]
}

func (seg *cathodeSegmentation4) padGroupType(padcid mapping.PadCID) *padGroupType {
	return &seg.padGroupTypes[seg.padGroup(padcid).padGroupTypeID]
}

func (seg *cathodeSegmentation4) FindPadByPosition(x, y float64) (mapping.PadCID, error) {
	pgti := make([]int, 0, 2)
	pgis := seg.grid.padGroupIndex(x, y)
	for pgi := range pgis {
		pgIndex := pgis[pgi]
		pg := seg.padGroups[pgIndex]
		pgt := seg.padGroupTypes[pg.padGroupTypeID]
		lx := x - pg.x
		ly := y - pg.y
		ix := int(math.Trunc(lx / seg.padSizes[pg.padSizeID].x))
		iy := int(math.Trunc(ly / seg.padSizes[pg.padSizeID].y))
		valid := pgt.areIndicesPossible(ix, iy) && lx >= 0.0 && ly >= 0.0
		if valid {
			// find in padcid2PadGroupTypeFastIndex array, starting at seg.padGroupIndex2PadCIDIndex[pgis[pgi]]
			// the padcid corresponding to pgt.fastIndex(ix,iy)
			// FIXME : that is wrong.
			a := seg.padGroupIndex2PadCIDIndex[pgIndex]
			asize := len(seg.padGroupIndex2PadCIDIndex) - 1
			var b int
			if pgIndex >= asize-1 {
				b = len(seg.padcid2PadGroupIndex) - 1
			} else {
				b = seg.padGroupIndex2PadCIDIndex[pgIndex+1]
			}
			for j := a; j <= b; j++ {
				if pgt.fastIndex(ix, iy) == seg.padcid2PadGroupTypeFastIndex[j] {
					pgti = append(pgti, j)
					break
				}
			}
		}
	}
	if len(pgti) > 1 {
		var imin int
		var dmin = math.MaxFloat64
		for i := 0; i < len(pgti); i++ {
			px := seg.PadPositionX(mapping.PadCID(pgti[i]))
			py := seg.PadPositionY(mapping.PadCID(pgti[i]))
			d := (x-px)*(x-px) + (y-py)*(y-py)
			if d < dmin {
				imin = i
				dmin = d
			}
		}
		return mapping.PadCID(pgti[imin]), nil
	}
	if len(pgti) > 0 {
		return mapping.PadCID(pgti[0]), nil
	}
	return invalidPadCID, mapping.ErrInvalidPadCID
}

func (seg *cathodeSegmentation4) PadPositionX(padcid mapping.PadCID) float64 {
	pg := seg.padGroup(padcid)
	pgt := seg.padGroupType(padcid)
	return pg.x + (float64(pgt.ix(seg.padcid2PadGroupTypeFastIndex[padcid]))+0.5)*
		seg.padSizes[pg.padSizeID].x
}

func (seg *cathodeSegmentation4) PadPositionY(padcid mapping.PadCID) float64 {
	pg := seg.padGroup(padcid)
	pgt := seg.padGroupType(padcid)
	return pg.y + (float64(pgt.iy(seg.padcid2PadGroupTypeFastIndex[padcid]))+0.5)*
		seg.padSizes[pg.padSizeID].y
}

func (seg *cathodeSegmentation4) ForEachPad(padHandler func(padcid mapping.PadCID)) {
	for p := 0; p < len(seg.padcid2PadGroupIndex); p++ {
		padHandler(mapping.PadCID(p))
	}
}

// GetNeighbours returns the list of neighbours of given pad.
// padcid is not checked here so it is assumed to be correct.
//
// neighbours slice must be of len >= 12
// The returned value indicates which range in the neighbours slice
// is useable (i.e. if 3 is returned, it means there are 3 neighbours,
// so only neighbours[0..2] is valid)
//
// Algorithm asserts pads at test positions (Left,Top,Right,Bottom)
// relative to pad's center (O) where we'll try to get a neighbouring pad,
// by getting a little bit outside the pad itself.
// The pad density can only decrease when going from left to right except
// for round slates where it is the opposite.
// The pad density can only decrease when going from bottom to top but
// to be symmetric we also consider the opposite.
// 4- 5- 6-7
// |       |
// 3       8
// |   0   |
// 2       9
// |       |
// 1-12-11-10
//
// TODO: investigate some (obvious ?) optimizations, e.g. not using
// all 12 test positions for some regions where we know it's not needed
//
var (
	tp []struct{ x, y float64 }
)

const eps float64 = 2 * 1E-5

func init() {
	tp = []struct{ x, y float64 }{
		{-1, -1},
		{-1, -1 / 3.0},
		{-1, 1 / 3.0},
		{-1, 1},
		{-1 / 3.0, 1},
		{1 / 3.0, 1},
		{1, 1},
		{1, 1 / 3.0},
		{1, -1 / 3.0},
		{1, -1},
		{1 / 3.0, -1},
		{-1 / 3.0, -1}}
}

func (seg *cathodeSegmentation4) GetNeighbourIDs(padcid mapping.PadCID, neighbours []int) int {
	px := seg.PadPositionX(padcid)
	py := seg.PadPositionY(padcid)
	dx := seg.PadSizeX(padcid) / 2.0
	dy := seg.PadSizeY(padcid) / 2.0
	var previous mapping.PadCID = -1
	i := 0
	for _, shift := range tp {
		xtest := px + (dx+eps)*shift.x
		ytest := py + (dy+eps)*shift.y
		uid, err := seg.FindPadByPosition(xtest, ytest)
		if err == nil && uid != previous {
			previous = uid
			neighbours[i] = int(previous)
			i++
		}
	}
	return i
}

func (seg *cathodeSegmentation4) IsBending() bool {
	return seg.isBendingPlane
}
