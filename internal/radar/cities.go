package radar

// City is a named geographic point used for radar map label overlays.
type City struct {
	Name string
	Lat  float64
	Lon  float64
}

// majorCities lists major US cities ordered roughly by population/significance.
// The ordering matters: when labels overlap, earlier (larger) cities win.
var majorCities = []City{
	{"New York", 40.7128, -74.0060},
	{"Los Angeles", 34.0522, -118.2437},
	{"Chicago", 41.8781, -87.6298},
	{"Houston", 29.7604, -95.3698},
	{"Phoenix", 33.4484, -112.0740},
	{"Philadelphia", 39.9526, -75.1652},
	{"San Antonio", 29.4241, -98.4936},
	{"San Diego", 32.7157, -117.1611},
	{"Dallas", 32.7767, -96.7970},
	{"Austin", 30.2672, -97.7431},
	{"Jacksonville", 30.3322, -81.6557},
	{"Columbus", 39.9612, -82.9988},
	{"Indianapolis", 39.7684, -86.1581},
	{"Charlotte", 35.2271, -80.8431},
	{"San Francisco", 37.7749, -122.4194},
	{"Seattle", 47.6062, -122.3321},
	{"Denver", 39.7392, -104.9903},
	{"Washington DC", 38.9072, -77.0369},
	{"Nashville", 36.1627, -86.7816},
	{"Oklahoma City", 35.4676, -97.5164},
	{"El Paso", 31.7619, -106.4850},
	{"Boston", 42.3601, -71.0589},
	{"Portland", 45.5152, -122.6784},
	{"Las Vegas", 36.1699, -115.1398},
	{"Memphis", 35.1495, -90.0490},
	{"Louisville", 38.2527, -85.7585},
	{"Baltimore", 39.2904, -76.6122},
	{"Milwaukee", 43.0389, -87.9065},
	{"Albuquerque", 35.0844, -106.6504},
	{"Tucson", 32.2226, -110.9747},
	{"Sacramento", 38.5816, -121.4944},
	{"Kansas City", 39.0997, -94.5786},
	{"Atlanta", 33.7490, -84.3880},
	{"Omaha", 41.2565, -95.9345},
	{"Colorado Springs", 38.8339, -104.8214},
	{"Raleigh", 35.7796, -78.6382},
	{"Miami", 25.7617, -80.1918},
	{"Minneapolis", 44.9778, -93.2650},
	{"Tampa", 27.9506, -82.4572},
	{"Tulsa", 36.1540, -95.9928},
	{"New Orleans", 29.9511, -90.0715},
	{"Cleveland", 41.4993, -81.6944},
	{"Pittsburgh", 40.4406, -79.9959},
	{"Cincinnati", 39.1031, -84.5120},
	{"St. Louis", 38.6270, -90.1994},
	{"Orlando", 28.5383, -81.3792},
	{"Detroit", 42.3314, -83.0458},
	{"Salt Lake City", 40.7608, -111.8910},
	{"Boise", 43.6150, -116.2023},
	{"Richmond", 37.5407, -77.4360},
	{"Des Moines", 41.5868, -93.6250},
	{"Birmingham", 33.5207, -86.8025},
	{"Little Rock", 34.7465, -92.2896},
	{"Buffalo", 42.8864, -78.8784},
	{"Spokane", 47.6588, -117.4260},
	{"Knoxville", 35.9606, -83.9207},
	{"Chattanooga", 35.0456, -85.3097},
	{"Jackson", 32.2988, -90.1848},
	{"Baton Rouge", 30.4515, -91.1871},
	{"Wichita", 37.6872, -97.3301},
	{"Topeka", 39.0473, -95.6752},
	{"Lincoln", 40.8136, -96.7026},
	{"Sioux Falls", 43.5446, -96.7311},
	{"Fargo", 46.8772, -96.7898},
	{"Bismarck", 46.8083, -100.7837},
	{"Rapid City", 44.0805, -103.2310},
	{"Billings", 45.7833, -108.5007},
	{"Missoula", 46.8721, -113.9940},
	{"Cheyenne", 41.1400, -104.8202},
	{"Charleston", 32.7765, -79.9311},
	{"Savannah", 32.0809, -81.0912},
	{"Mobile", 30.6954, -88.0399},
	{"Springfield", 37.2090, -93.2923},
	{"Norfolk", 36.8508, -76.2859},
	{"Lubbock", 33.5779, -101.8552},
	{"Amarillo", 35.2220, -101.8313},
	{"Corpus Christi", 27.8006, -97.3964},
	{"Shreveport", 32.5252, -93.7502},
	{"Madison", 43.0731, -89.4012},
	{"Green Bay", 44.5192, -88.0198},
	{"Duluth", 46.7867, -92.1005},
	{"Ft. Wayne", 41.0793, -85.1394},
	{"South Bend", 41.6764, -86.2520},
	{"Grand Rapids", 42.9634, -85.6681},
	{"Peoria", 40.6936, -89.5890},
	{"Dayton", 39.7589, -84.1916},
	{"Akron", 41.0814, -81.5190},
	{"Syracuse", 43.0481, -76.1474},
	{"Albany", 42.6526, -73.7562},
	{"Hartford", 41.7658, -72.6734},
	{"Providence", 41.8240, -71.4128},
	{"Tallahassee", 30.4383, -84.2807},
	{"Columbia", 34.0007, -81.0348},
	{"Montgomery", 32.3792, -86.3077},
	{"Reno", 39.5296, -119.8138},
}

// citiesInBBox returns cities that fall within the given geographic extent.
func citiesInBBox(bb BBox) []City {
	var out []City
	for _, c := range majorCities {
		if c.Lat >= bb.MinLat && c.Lat <= bb.MaxLat &&
			c.Lon >= bb.MinLon && c.Lon <= bb.MaxLon {
			out = append(out, c)
		}
	}
	return out
}
