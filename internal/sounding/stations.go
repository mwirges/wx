package sounding

import (
	"math"
)

// Station represents a NOAA NWS Upper-Air Radiosonde (RAOB) sounding station.
type Station struct {
	ID    string  // 3-letter or standard identifier (e.g. "ILX", "OUN", "DTX")
	Name  string  // Full facility or city name
	State string  // US state or territory
	Lat   float64 // Latitude (degrees N)
	Lon   float64 // Longitude (degrees E/W)
}

// Stations is the authoritative list of NOAA NWS upper-air radiosonde launch sites.
var Stations = []Station{
	// Midwest & Great Lakes
	{ID: "ILX", Name: "Lincoln", State: "IL", Lat: 40.15, Lon: -89.34},
	{ID: "DVN", Name: "Davenport", State: "IA", Lat: 41.61, Lon: -90.58},
	{ID: "ILN", Name: "Wilmington", State: "OH", Lat: 39.42, Lon: -83.82},
	{ID: "DTX", Name: "Detroit / White Lake", State: "MI", Lat: 42.70, Lon: -83.47},
	{ID: "APX", Name: "Gaylord", State: "MI", Lat: 44.91, Lon: -84.72},
	{ID: "GRB", Name: "Green Bay", State: "WI", Lat: 44.48, Lon: -88.13},
	{ID: "MPX", Name: "Minneapolis / Chanhassen", State: "MN", Lat: 44.85, Lon: -93.56},
	{ID: "INL", Name: "International Falls", State: "MN", Lat: 48.57, Lon: -93.40},

	// Central Plains & Ozarks
	{ID: "OAX", Name: "Omaha / Valley", State: "NE", Lat: 41.32, Lon: -96.37},
	{ID: "LBF", Name: "North Platte", State: "NE", Lat: 41.13, Lon: -100.68},
	{ID: "TOP", Name: "Topeka", State: "KS", Lat: 39.07, Lon: -95.63},
	{ID: "DDC", Name: "Dodge City", State: "KS", Lat: 37.77, Lon: -100.00},
	{ID: "SGF", Name: "Springfield", State: "MO", Lat: 37.23, Lon: -93.38},
	{ID: "OUN", Name: "Norman", State: "OK", Lat: 35.18, Lon: -97.44},

	// Northern Plains & High Plains
	{ID: "ABR", Name: "Aberdeen", State: "SD", Lat: 45.45, Lon: -98.42},
	{ID: "UNR", Name: "Rapid City", State: "SD", Lat: 44.07, Lon: -103.21},
	{ID: "BIS", Name: "Bismarck", State: "ND", Lat: 46.77, Lon: -100.75},
	{ID: "GGW", Name: "Glasgow", State: "MT", Lat: 48.21, Lon: -106.63},
	{ID: "TFX", Name: "Great Falls", State: "MT", Lat: 47.46, Lon: -111.38},
	{ID: "CYS", Name: "Cheyenne", State: "WY", Lat: 41.15, Lon: -104.81},
	{ID: "RIW", Name: "Riverton", State: "WY", Lat: 43.06, Lon: -108.48},

	// Southern Plains & Texas
	{ID: "AMA", Name: "Amarillo", State: "TX", Lat: 35.22, Lon: -101.71},
	{ID: "FWD", Name: "Fort Worth", State: "TX", Lat: 32.83, Lon: -97.30},
	{ID: "MAF", Name: "Midland", State: "TX", Lat: 31.94, Lon: -102.19},
	{ID: "EPZ", Name: "El Paso / Santa Teresa", State: "NM", Lat: 31.87, Lon: -106.70},
	{ID: "DRT", Name: "Del Rio", State: "TX", Lat: 29.37, Lon: -100.92},
	{ID: "CRP", Name: "Corpus Christi", State: "TX", Lat: 27.77, Lon: -97.51},
	{ID: "BRO", Name: "Brownsville", State: "TX", Lat: 25.91, Lon: -97.43},

	// Southeast & Gulf Coast
	{ID: "LZK", Name: "Little Rock", State: "AR", Lat: 34.83, Lon: -92.26},
	{ID: "SHV", Name: "Shreveport", State: "LA", Lat: 32.45, Lon: -93.84},
	{ID: "LCH", Name: "Lake Charles", State: "LA", Lat: 30.13, Lon: -93.22},
	{ID: "LIX", Name: "Slidell / New Orleans", State: "LA", Lat: 30.34, Lon: -89.83},
	{ID: "JAN", Name: "Jackson", State: "MS", Lat: 32.32, Lon: -90.08},
	{ID: "BMX", Name: "Birmingham / Shelby", State: "AL", Lat: 33.17, Lon: -86.77},
	{ID: "BNA", Name: "Nashville", State: "TN", Lat: 36.12, Lon: -86.68},
	{ID: "FFC", Name: "Peachtree City / Atlanta", State: "GA", Lat: 33.36, Lon: -84.57},
	{ID: "TLH", Name: "Tallahassee", State: "FL", Lat: 30.40, Lon: -84.35},
	{ID: "JAX", Name: "Jacksonville", State: "FL", Lat: 30.50, Lon: -81.70},
	{ID: "TBW", Name: "Tampa Bay / Ruskin", State: "FL", Lat: 27.71, Lon: -82.40},
	{ID: "MFL", Name: "Miami", State: "FL", Lat: 25.75, Lon: -80.38},
	{ID: "EYW", Name: "Key West", State: "FL", Lat: 24.55, Lon: -81.76},
	{ID: "CHS", Name: "Charleston", State: "SC", Lat: 32.90, Lon: -80.03},
	{ID: "GSO", Name: "Greensboro", State: "NC", Lat: 36.08, Lon: -79.95},
	{ID: "MHX", Name: "Morehead City / Newport", State: "NC", Lat: 34.78, Lon: -76.88},

	// Mid-Atlantic & Northeast
	{ID: "RNK", Name: "Blacksburg", State: "VA", Lat: 37.20, Lon: -80.41},
	{ID: "IAD", Name: "Sterling / Dulles", State: "VA", Lat: 38.98, Lon: -77.46},
	{ID: "WAL", Name: "Wallops Island", State: "VA", Lat: 37.94, Lon: -75.47},
	{ID: "PIT", Name: "Pittsburgh", State: "PA", Lat: 40.53, Lon: -80.22},
	{ID: "BUF", Name: "Buffalo", State: "NY", Lat: 42.94, Lon: -78.73},
	{ID: "ALB", Name: "Albany", State: "NY", Lat: 42.75, Lon: -73.80},
	{ID: "OKX", Name: "Upton / Brookhaven", State: "NY", Lat: 40.87, Lon: -72.87},
	{ID: "CHH", Name: "Chatham", State: "MA", Lat: 41.67, Lon: -69.97},
	{ID: "GYX", Name: "Gray / Portland", State: "ME", Lat: 43.89, Lon: -70.26},
	{ID: "CAR", Name: "Caribou", State: "ME", Lat: 46.87, Lon: -68.01},

	// Rocky Mountains & Southwest
	{ID: "DNR", Name: "Denver", State: "CO", Lat: 39.77, Lon: -104.88},
	{ID: "GJT", Name: "Grand Junction", State: "CO", Lat: 39.12, Lon: -108.53},
	{ID: "ABQ", Name: "Albuquerque", State: "NM", Lat: 35.04, Lon: -106.62},
	{ID: "TUS", Name: "Tucson", State: "AZ", Lat: 32.13, Lon: -110.96},
	{ID: "FGZ", Name: "Flagstaff", State: "AZ", Lat: 35.23, Lon: -111.82},
	{ID: "VEF", Name: "Las Vegas", State: "NV", Lat: 36.05, Lon: -115.18},
	{ID: "DRA", Name: "Desert Rock / Mercury", State: "NV", Lat: 36.62, Lon: -116.03},
	{ID: "REV", Name: "Reno", State: "NV", Lat: 39.56, Lon: -119.80},
	{ID: "SLC", Name: "Salt Lake City", State: "UT", Lat: 40.79, Lon: -111.96},
	{ID: "BOI", Name: "Boise", State: "ID", Lat: 43.57, Lon: -116.22},

	// Pacific Northwest & West Coast
	{ID: "OTX", Name: "Spokane", State: "WA", Lat: 47.68, Lon: -117.63},
	{ID: "UIL", Name: "Quillayute", State: "WA", Lat: 47.95, Lon: -124.55},
	{ID: "SLE", Name: "Salem", State: "OR", Lat: 44.91, Lon: -123.00},
	{ID: "MFR", Name: "Medford", State: "OR", Lat: 42.37, Lon: -122.87},
	{ID: "OAK", Name: "Oakland", State: "CA", Lat: 37.74, Lon: -122.22},
	{ID: "VBG", Name: "Vandenberg AFB", State: "CA", Lat: 34.73, Lon: -120.57},
	{ID: "EDW", Name: "Edwards AFB", State: "CA", Lat: 34.91, Lon: -117.88},
	{ID: "NKX", Name: "San Diego / Miramar", State: "CA", Lat: 32.87, Lon: -117.15},

	// Non-contiguous US
	{ID: "JSJ", Name: "San Juan", State: "PR", Lat: 18.43, Lon: -65.99},
	{ID: "ANC", Name: "Anchorage", State: "AK", Lat: 61.17, Lon: -150.02},
	{ID: "FAI", Name: "Fairbanks", State: "AK", Lat: 64.82, Lon: -147.87},
}

// FindClosestStation returns the nearest NOAA upper-air sounding station and distance in km.
func FindClosestStation(lat, lon float64) (Station, float64) {
	var best Station
	bestDist := math.MaxFloat64

	for _, s := range Stations {
		d := haversineKm(lat, lon, s.Lat, s.Lon)
		if d < bestDist {
			bestDist = d
			best = s
		}
	}
	return best, bestDist
}

// FindStationByID returns the station by ID, or false if not found.
func FindStationByID(id string) (Station, bool) {
	for _, s := range Stations {
		if s.ID == id {
			return s, true
		}
	}
	return Station{}, false
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0 // Earth radius km
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180.0)*math.Cos(lat2*math.Pi/180.0)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return r * c
}
