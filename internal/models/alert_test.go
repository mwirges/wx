package models

import "testing"

func TestAlertClassification(t *testing.T) {
	warning := Alert{Event: "Tornado Warning", Severity: "Extreme"}
	if !warning.IsWarning() {
		t.Errorf("expected warning to be true")
	}
	if warning.IsWatch() || warning.IsAdvisory() {
		t.Errorf("expected warning not to be watch or advisory")
	}

	watch := Alert{Event: "Severe Thunderstorm Watch", Severity: "Moderate"}
	if !watch.IsWatch() {
		t.Errorf("expected watch to be true")
	}
	if watch.IsWarning() || watch.IsAdvisory() {
		t.Errorf("expected watch not to be warning or advisory")
	}

	advisory := Alert{Event: "Winter Weather Advisory", Severity: "Minor"}
	if !advisory.IsAdvisory() {
		t.Errorf("expected advisory to be true")
	}
	if advisory.IsWarning() || advisory.IsWatch() {
		t.Errorf("expected advisory not to be warning or watch")
	}
}
