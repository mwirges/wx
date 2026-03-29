package radar

import "testing"

func TestNearestStation(t *testing.T) {
	tests := []struct {
		name     string
		lat, lon float64
		wantID   string
	}{
		{"Kansas City", 39.1, -94.6, "KEAX"},
		{"Chicago", 41.88, -87.63, "KLOT"},
		{"Miami", 25.76, -80.19, "KAMX"},
		{"Seattle", 47.6, -122.3, "KATX"},
		{"Denver", 39.74, -104.99, "KFTG"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := NearestStation(tc.lat, tc.lon)
			if got.ID != tc.wantID {
				t.Errorf("NearestStation(%.2f, %.2f) = %s, want %s", tc.lat, tc.lon, got.ID, tc.wantID)
			}
		})
	}
}

func TestIsStationProduct(t *testing.T) {
	if IsStationProduct(ProductCompositeReflectivity) {
		t.Error("composite reflectivity should not be a station product")
	}
	if !IsStationProduct(ProductBaseReflectivity) {
		t.Error("base reflectivity should be a station product")
	}
	if !IsStationProduct(ProductStormRelativeVelocity) {
		t.Error("storm relative velocity should be a station product")
	}
	if IsStationProduct(ProductEchoTops) {
		t.Error("echo tops should NOT be a station product (uses WMS mosaic)")
	}
}
