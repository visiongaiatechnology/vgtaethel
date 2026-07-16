package skills

import "testing"

func TestValidWeatherCity(t *testing.T) {
	for _, city := range []string{"Köln", "New York", "Saint-Étienne", "L'Aquila"} {
		if !validWeatherCity(city) {
			t.Fatalf("expected %q to be accepted", city)
		}
	}
	for _, city := range []string{"", "A", "Köln/../../", "https://example.test", "<script>"} {
		if validWeatherCity(city) {
			t.Fatalf("expected %q to be rejected", city)
		}
	}
}

func TestWeatherDescription(t *testing.T) {
	if got := weatherDescription(0); got != "klar" {
		t.Fatalf("weather code 0 = %q", got)
	}
	if got := weatherDescription(95); got != "Gewitter" {
		t.Fatalf("weather code 95 = %q", got)
	}
}
