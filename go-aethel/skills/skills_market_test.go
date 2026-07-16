package skills

import "testing"

func TestFilterMarketQuotesUsesFixedAllowlist(t *testing.T) {
	quotes := []MarketQuote{{Symbol: "BTC"}, {Symbol: "ETH"}, {Symbol: "GOLD"}}
	filtered := filterMarketQuotes(quotes, []string{"btc", "UNKNOWN", "gold"})
	if len(filtered) != 2 || filtered[0].Symbol != "BTC" || filtered[1].Symbol != "GOLD" {
		t.Fatalf("unexpected market filter result: %+v", filtered)
	}
}

func TestMarketSkillSchemaRejectsUnknownFieldsByContract(t *testing.T) {
	schema := (&MarketSkill{}).Parameters()
	if schema["additionalProperties"] != false {
		t.Fatal("market tool schema must reject additional properties")
	}
}
