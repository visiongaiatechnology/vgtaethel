package intelligence

// STATUS: DIAMANT VGT SUPREME

import "testing"

func TestEntityNormalizationTransliteratesCyrillicAndGreek(t *testing.T) {
	for input, expected := range map[string]string{
		"\u041c\u043e\u0441\u043a\u0432\u0430": "moskva",
		"\u0391\u03b8\u03ae\u03bd\u03b1":       "athina",
	} {
		if actual := normalizeEntityName(input); actual != expected {
			t.Fatalf("normalizeEntityName(%q)=%q, expected %q", input, actual, expected)
		}
	}
}

func TestTitlesAreSimilarAcrossTransliteratedScripts(t *testing.T) {
	if !titlesAreSimilar("\u041c\u043e\u0441\u043a\u0432\u0430 energy terminal incident confirmed", "Moskva energy terminal incident confirmed") {
		t.Fatal("transliterated reports were not recognized as likely duplicates")
	}
	if titlesAreSimilar("Moskva energy terminal incident confirmed", "Quarterly agricultural forecast published") {
		t.Fatal("unrelated reports were collapsed")
	}
}
