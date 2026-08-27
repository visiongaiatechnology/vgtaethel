// STATUS: DIAMANT VGT SUPREME
package agent

import "testing"

func TestMatchUserLocationCityRequiredForExistentialGate(t *testing.T) {
	// Remote Japan quake must not match a German home city.
	cityMatch, countryMatch := matchUserLocation("96 km E of Yamada, Japan", "Berlin", "Germany")
	if cityMatch || countryMatch {
		t.Fatalf("remote Japan place must not match Berlin/Germany: city=%v country=%v", cityMatch, countryMatch)
	}

	// Same-country but different city: country only.
	cityMatch, countryMatch = matchUserLocation("3 km SW of Munich, Germany", "Berlin", "Germany")
	if cityMatch {
		t.Fatal("Munich event must not city-match Berlin")
	}
	if !countryMatch {
		t.Fatal("Munich, Germany must country-match Germany")
	}

	// True city hit.
	cityMatch, countryMatch = matchUserLocation("5 km N of Berlin, Germany", "Berlin", "Deutschland")
	if !cityMatch {
		t.Fatal("Berlin place must city-match Berlin")
	}
	if !countryMatch {
		t.Fatal("Germany place must country-match Deutschland alias")
	}
}

func TestMatchUserLocationRejectsShortTokensAndSubstrings(t *testing.T) {
	// "us" must not match inside "Russia" / random text when too short — min length 3.
	cityMatch, countryMatch := matchUserLocation("10 km S of Moscow, Russia", "", "us")
	if cityMatch || countryMatch {
		t.Fatalf("short country token 'us' must not match: city=%v country=%v", cityMatch, countryMatch)
	}

	// Substring false positive: city "ham" must not match "Hamburg" via bare contains —
	// containsPlaceToken requires full token; "ham" as whole word would match only if present.
	cityMatch, _ = matchUserLocation("2 km E of Hamburg, Germany", "Hamburg", "Germany")
	if !cityMatch {
		t.Fatal("Hamburg must match city Hamburg")
	}
	cityMatch, _ = matchUserLocation("2 km E of Hamburg, Germany", "Ham", "Germany")
	if cityMatch {
		t.Fatal("'Ham' must not token-match Hamburg")
	}
}

func TestHasConfiguredHome(t *testing.T) {
	if hasConfiguredHome("", "") {
		t.Fatal("empty home must be unconfigured")
	}
	if !hasConfiguredHome("Berlin", "") {
		t.Fatal("city alone is configured")
	}
	if !hasConfiguredHome("", "Germany") {
		t.Fatal("country alone is configured")
	}
}

func TestContainsPlaceTokenBoundaries(t *testing.T) {
	if !containsPlaceToken("near Tokyo, Japan", "tokyo") {
		t.Fatal("expected tokyo match")
	}
	if containsPlaceToken("near Tokyo, Japan", "okyo") {
		t.Fatal("partial token must not match")
	}
	if containsPlaceToken("", "berlin") {
		t.Fatal("empty haystack")
	}
	if containsPlaceToken("berlin", "be") {
		t.Fatal("short needle rejected")
	}
}
