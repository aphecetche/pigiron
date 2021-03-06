package mapping

// SegCache is a simple cache for the detection element segmentations.
type SegCache struct {
	seg map[DEID]Segmentation
}

// Segmentation returns the segmentation for given detection element id
// and given plane (true for bending plane).
// The segmentation for both planes of that detection element is created
// and cached if not already cached
func (sc *SegCache) CathodeSegmentation(deid DEID, bending bool) CathodeSegmentation {
	seg := sc.Segmentation(deid)
	if bending {
		return seg.Bending()
	}
	return seg.NonBending()
}

func (sc *SegCache) Segmentation(deid DEID) Segmentation {
	if sc.seg == nil {
		sc.seg = make(map[DEID]Segmentation)
	}
	seg := sc.seg[deid]
	if seg == nil {
		sc.seg[deid] = NewSegmentation(deid)
		seg = sc.seg[deid]
	}
	return seg
}
