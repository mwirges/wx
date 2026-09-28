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
	if IsStationProduct(ProductPrecipType) {
		t.Error("precip type should NOT be a station product (uses WMS mosaic)")
	}
	if !IsStationProduct(ProductOneHourPrecip) {
		t.Error("one hour precip should be a station product (uses RIDGE N1P)")
	}
	if !IsStationProduct(ProductStormTotalPrecip) {
		t.Error("storm total precip should be a station product (uses RIDGE NTP)")
	}
}

func TestStationsWithinRadius(t *testing.T) {
	// Fort Wayne, IN (approx 41.08, -85.14)
	lat, lon := 41.0793, -85.1394
	// Within 65km should include KIWX (North Webster, ~56km away)
	closeStations := StationsWithinRadius(lat, lon, 65.0)
	if len(closeStations) != 1 || closeStations[0].ID != "KIWX" {
		t.Errorf("StationsWithinRadius(65km) = %v, want [KIWX]", closeStations)
	}

	// Within 300km should include multiple stations (KIWX, KIND, KLOT, KGRR, KDTX, KCLE, KILN, etc.)
	regionalStations := StationsWithinRadius(lat, lon, 300.0)
	if len(regionalStations) < 5 {
		t.Errorf("StationsWithinRadius(300km) = %d stations, want >= 5", len(regionalStations))
	}
}
