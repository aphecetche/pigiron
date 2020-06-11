package v2

import (
	"encoding/json"
	"fmt"
	"io"
	"log"

	"github.com/mrrtf/pigiron/mapping"
	"github.com/mrrtf/pigiron/segcontour"
)

type Vertex struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Pad struct {
	DSID int     `json:"DSID"`
	DSCH int     `json:"DSCH"`
	X    float64 `json:"X"`
	Y    float64 `json:"Y"`
	SX   float64 `json:"SX"`
	SY   float64 `json:"SY"`
}

type DualSampaPads struct {
	ID   int `json:"id"`
	Pads []Pad
}

type DualSampa struct {
	ID       int      `json:"id"`
	Vertices []Vertex `json:"vertices"`
}

type DEGeo struct {
	ID       int      `json:"id"`
	Bending  bool     `json:"bending"`
	X        float64  `json:"x"`
	Y        float64  `json:"y"`
	SX       float64  `json:"sx"`
	SY       float64  `json:"sy"`
	Vertices []Vertex `json:"vertices"`
}

type FlipDirection int

const (
	FlipNone FlipDirection = iota + 1
	FlipX
	FlipY
	FlipXY
)

func flipVertex(p Vertex, d FlipDirection, xcenter, ycenter float64) Vertex {
	if d == FlipX || d == FlipXY {
		p.Y = 2*ycenter - p.Y
	}
	if d == FlipY || d == FlipXY {
		p.X = 2*xcenter - p.X
	}
	return p
}

var de2rot = map[int]FlipDirection{}

func initDetectionElementRotations() {
	for i := 1; i <= 4; i++ {
		de2rot[i*100] = FlipX
		de2rot[i*100+1] = FlipXY
		de2rot[i*100+2] = FlipY
		de2rot[i*100+3] = FlipNone
	}
	de2rot[500] = FlipXY
	de2rot[501] = FlipXY
	de2rot[502] = FlipNone
	de2rot[503] = FlipX
	de2rot[504] = FlipNone
	de2rot[505] = FlipY
	de2rot[506] = FlipXY
	de2rot[507] = FlipY
	de2rot[508] = FlipX
	de2rot[509] = FlipX
	de2rot[510] = FlipNone
	de2rot[511] = FlipY
	de2rot[512] = FlipXY
	de2rot[513] = FlipY
	de2rot[514] = FlipNone
	de2rot[515] = FlipX
	de2rot[516] = FlipNone
	de2rot[517] = FlipY
}

func init() {
	initDetectionElementRotations()
}

func flipVertices(vertices []Vertex, deid int, xcenter float64, ycenter float64) []Vertex {
	var flipped []Vertex
	direction, _ := de2rot[deid]
	for _, v := range vertices {
		flipped = append(flipped, flipVertex(v, direction, xcenter, ycenter))
	}
	return flipped
}

func jsonDEGeo(w io.Writer, cseg mapping.CathodeSegmentation, bending bool) {

	bbox := mapping.ComputeBBox(cseg)

	var vertices []Vertex
	contour := segcontour.Contour(cseg)
	deid := int(cseg.DetElemID())

	for _, c := range contour {
		for _, v := range c {
			vertices = append(vertices, Vertex{X: v.X, Y: v.Y})
		}
	}

	degeo := DEGeo{
		ID:      deid,
		Bending: cseg.IsBending(),
		X:       bbox.Xcenter(),
		Y:       bbox.Ycenter(),
		SX:      bbox.Width(),
		SY:      bbox.Height()}

	degeo.Vertices = flipVertices(vertices, deid, bbox.Xcenter(), bbox.Ycenter())
	b, err := json.Marshal(degeo)

	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Fprintf(w, string(b))
}

func jsonDualSampaPads(w io.Writer, cseg mapping.CathodeSegmentation, dsid int) {
	var dualSampas []DualSampa
	n := cseg.NofDualSampas()

	for i := 0; i < n; i++ {
		dsid, err := cseg.DualSampaID(i)
		if err != nil {
			panic(err)
		}

		ds := DualSampa{ID: int(dsid)}

		padContours := segcontour.GetDualSampaPadPolygons(cseg, mapping.DualSampaID(dsid))

		for _, c := range padContours {
			for _, v := range c {
				ds.Vertices = append(ds.Vertices, Vertex{X: v.X, Y: v.Y})
			}
		}

		dualSampas = append(dualSampas, ds)
	}

	b, err := json.Marshal(dualSampas)

	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Fprintf(w, string(b))
}

func jsonDualSampas(w io.Writer, cseg mapping.CathodeSegmentation, bending bool) {

	var dualSampas []DualSampa
	n := cseg.NofDualSampas()

	bbox := mapping.ComputeBBox(cseg)

	deid := int(cseg.DetElemID())

	for i := 0; i < n; i++ {
		dsid, err := cseg.DualSampaID(i)
		if err != nil {
			panic(err)
		}

		ds := DualSampa{ID: int(dsid)}

		dsContour := segcontour.GetDualSampaContour(cseg, dsid)
		for _, c := range dsContour {
			for _, v := range c {
				ds.Vertices = append(ds.Vertices, Vertex{X: v.X, Y: v.Y})
			}
		}

		ds.Vertices = flipVertices(ds.Vertices, deid, bbox.Xcenter(), bbox.Ycenter())

		dualSampas = append(dualSampas, ds)
	}

	b, err := json.Marshal(dualSampas)

	if err != nil {
		log.Fatalf(err.Error())
	}

	fmt.Fprintf(w, string(b))
}