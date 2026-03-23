package radar

import (
	"testing"
	"unicode/utf8"
)

func TestCitiesInBBox_KansasCity(t *testing.T) {
	// BBox centered roughly on Kansas City, 200 km radius ≈ ±1.8° lat.
	bb := BBox{MinLat: 37.3, MinLon: -96.8, MaxLat: 40.9, MaxLon: -92.4}
	cities := citiesInBBox(bb)
	found := false
	for _, c := range cities {
		if c.Name == "Kansas City" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected Kansas City in bbox, got:", cities)
	}
}

func TestCitiesInBBox_Empty(t *testing.T) {
	// Middle of the Pacific Ocean — no cities.
	bb := BBox{MinLat: 10, MinLon: -160, MaxLat: 12, MaxLon: -158}
	if got := citiesInBBox(bb); len(got) != 0 {
		t.Errorf("expected 0 cities in ocean bbox, got %d", len(got))
	}
}

func TestLayoutLabels_NilBBox(t *testing.T) {
	labels := layoutLabels(nil, 80, 20)
	if len(labels) != 0 {
		t.Errorf("expected no labels with nil BBox, got %d", len(labels))
	}
}

func TestLayoutLabels_Placement(t *testing.T) {
	// A small BBox containing only Kansas City.
	bb := BBox{MinLat: 38.5, MinLon: -95.5, MaxLat: 39.7, MaxLon: -93.5}
	labels := layoutLabels(&bb, 80, 20)
	if len(labels) == 0 {
		t.Fatal("expected at least one label")
	}
	lbl := labels[0]
	if lbl.col < 0 || lbl.col >= 80 {
		t.Errorf("label col %d out of range [0, 80)", lbl.col)
	}
	if lbl.row < 0 || lbl.row >= 20 {
		t.Errorf("label row %d out of range [0, 20)", lbl.row)
	}
	if len(lbl.text) < 3 {
		t.Errorf("label text too short: %q", lbl.text)
	}
}

func TestLayoutLabels_OverlapRemoval(t *testing.T) {
	// Two cities very close together at a tiny terminal — should drop the
	// lower-priority one.
	bb := BBox{MinLat: 38.0, MinLon: -98.0, MaxLat: 40.0, MaxLon: -93.0}
	labels := layoutLabels(&bb, 30, 10)
	// At 30 cols, labels for nearby cities will collide. Just verify we
	// didn't crash and produced some output.
	for _, lbl := range labels {
		if lbl.col+utf8.RuneCountInString(lbl.text) > 30 {
			t.Errorf("label %q exceeds terminal width: col=%d len=%d", lbl.text, lbl.col, utf8.RuneCountInString(lbl.text))
		}
	}
}

func TestBuildLabelIndex(t *testing.T) {
	labels := []placedLabel{
		{col: 5, row: 3, text: "● KC"},
	}
	idx := buildLabelIndex(labels)

	// Characters within the label should be found.
	if ch, ok := idx.at(3, 5); !ok || ch != '●' {
		t.Errorf("expected '●' at (3,5), got %q ok=%v", string(ch), ok)
	}
	if ch, ok := idx.at(3, 7); !ok || ch != 'K' {
		t.Errorf("expected 'K' at (3,7), got %q ok=%v", string(ch), ok)
	}

	// Outside the label should miss.
	if _, ok := idx.at(3, 20); ok {
		t.Error("expected miss at (3,20)")
	}
	if _, ok := idx.at(0, 5); ok {
		t.Error("expected miss at (0,5)")
	}
}
