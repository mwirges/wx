"""CLI-shaped JSON for shell tests. Not a weather source."""

PAYLOAD = {
    "conditions": {
        "station": "KFWA",
        "observed_at": "2026-10-04T15:00:00Z",
        "location": "Fort Wayne, IN",
        "description": "Partly Cloudy",
        "condition_code": "partly-cloudy-day",
        "temperature_c": 22.2,
        "temperature_f": 72.0,
        "feels_like_f": 72.0,
        "feels_like_c": 22.2,
        "humidity_pct": 55,
        "wind_mph": 12,
        "wind_kph": 19.3,
        "wind_direction": "NW",
        "pressure_inhg": 30.05,
        "pressure_hpa": 1017,
        "visibility_mi": 10,
        "visibility_m": 16093,
        "nowcast": {
            "headline": "Dry next 6h",
            "is_active_precip": False,
            "summary": "No precipitation expected.",
            "primary_phase": "none",
            "intervals": [
                {
                    "start_time": "2026-10-04T15:00:00Z",
                    "summary": "Clear",
                    "probability": 0,
                    "phase": "none",
                }
            ],
        },
    },
    "forecast": {
        "periods": [
            {
                "name": "This Afternoon",
                "start_time": "2026-10-04T18:00:00Z",
                "is_daytime": True,
                "temperature_f": 74,
                "temperature_c": 23,
                "short_description": "Partly Sunny",
                "probability_of_precipitation": 10,
            }
        ],
        "hourly": [
            {
                "name": "3 PM",
                "start_time": "2026-10-04T19:00:00Z",
                "is_daytime": True,
                "temperature_f": 72,
                "temperature_c": 22,
                "short_description": "Partly Cloudy",
                "probability_of_precipitation": 5,
            }
        ],
    },
    "alerts": [
        {
            "event": "Severe Thunderstorm Warning",
            "headline": "Storm near New Haven",
            "severity": "Severe",
            "urgency": "Immediate",
            "description": "Hail and wind.",
        }
    ],
}

QUIET = {
    "conditions": {
        "station": "KFWA",
        "location": "Fort Wayne, IN",
        "temperature_f": 68,
        "temperature_c": 20,
        "condition_code": "clear-day",
    },
    "forecast": {"periods": [], "hourly": []},
    "alerts": [],
}

WATCH = {
    "conditions": {"station": "KORD", "location": "Chicago, IL", "temperature_f": 80, "temperature_c": 26.7},
    "alerts": [{"event": "Flood Watch", "severity": "Moderate", "headline": "Flooding possible"}],
}
