package radar

import "unicode/utf8"

// placedLabel is a city label projected into terminal coordinates.
type placedLabel struct {
	col  int    // starting terminal column
	row  int    // terminal row (half-block row, not pixel row)
	text string // "● CityName"
}

// layoutLabels projects visible cities onto the terminal grid and removes
// overlapping labels. Cities earlier in the majorCities list (larger population)
// take priority when labels overlap.
func layoutLabels(bb *BBox, termW, termRows int) []placedLabel {
	if bb == nil || termW < 20 || termRows < 5 {
		return nil
	}
	cities := citiesInBBox(*bb)
	if len(cities) == 0 {
		return nil
	}

	lonSpan := bb.MaxLon - bb.MinLon
	latSpan := bb.MaxLat - bb.MinLat
	if lonSpan == 0 || latSpan == 0 {
		return nil
	}

	var candidates []placedLabel
	for _, c := range cities {
		col := int((c.Lon - bb.MinLon) / lonSpan * float64(termW))
		row := int((bb.MaxLat - c.Lat) / latSpan * float64(termRows))
		if col < 0 || col >= termW || row < 0 || row >= termRows {
			continue
		}
		text := "● " + c.Name
		// Clip label to right edge — leave 1 col margin.
		maxCols := termW - col - 1
		if maxCols < 4 {
			continue // not enough room for even "● X"
		}
		runes := []rune(text)
		if len(runes) > maxCols {
			runes = runes[:maxCols]
			text = string(runes)
		}
		candidates = append(candidates, placedLabel{col: col, row: row, text: text})
	}

	// Greedy overlap removal: per-row column occupancy.
	type span struct{ lo, hi int }
	occupied := make(map[int][]span) // key = row

	var placed []placedLabel
	for _, lbl := range candidates {
		lo := lbl.col
		hi := lbl.col + utf8.RuneCountInString(lbl.text)
		overlap := false
		for _, s := range occupied[lbl.row] {
			if lo < s.hi && hi > s.lo {
				overlap = true
				break
			}
		}
		if overlap {
			continue
		}
		occupied[lbl.row] = append(occupied[lbl.row], span{lo, hi})
		placed = append(placed, lbl)
	}
	return placed
}

// labelIndex builds a lookup from (row, col) → rune for fast access
// during half-block rendering.
type labelIndex struct {
	chars map[[2]int]rune // [row, col] → character
}

func buildLabelIndex(labels []placedLabel) labelIndex {
	idx := labelIndex{chars: make(map[[2]int]rune, len(labels)*12)}
	for _, lbl := range labels {
		runes := []rune(lbl.text)
		for i, ch := range runes {
			idx.chars[[2]int{lbl.row, lbl.col + i}] = ch
		}
	}
	return idx
}

func (idx labelIndex) at(row, col int) (rune, bool) {
	ch, ok := idx.chars[[2]int{row, col}]
	return ch, ok
}
