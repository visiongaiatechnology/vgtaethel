package skills

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"go-aethel/security"
)

type MarketSkill struct{}

type MarketQuote struct {
	Symbol         string    `json:"symbol"`
	Name           string    `json:"name"`
	Currency       string    `json:"currency"`
	Price          float64   `json:"price"`
	Change24H      float64   `json:"change_24h_percent"`
	ObservedAt     time.Time `json:"observed_at"`
	Source         string    `json:"source"`
	InstrumentNote string    `json:"instrument_note,omitempty"`
}

var marketCatalog = map[string]struct{ ID, Name, Note string }{
	"BTC":  {ID: "bitcoin", Name: "Bitcoin"},
	"ETH":  {ID: "ethereum", Name: "Ethereum"},
	"SOL":  {ID: "solana", Name: "Solana"},
	"GOLD": {ID: "pax-gold", Name: "Gold", Note: "PAX Gold (PAXG) token proxy; kein Börsen-XAU-Spotfixing"},
}

var marketCache struct {
	sync.Mutex
	at     time.Time
	quotes []MarketQuote
}

func (s *MarketSkill) Name() string { return "market_lookup" }
func (s *MarketSkill) Description() string {
	return "Liest aktuelle Kurse für BTC, ETH, SOL und einen transparent gekennzeichneten Gold-Proxy und öffnet damit das Sphere-Marktbild."
}
func (s *MarketSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *MarketSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{"type": "object", "properties": map[string]interface{}{"symbols": map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string", "enum": []string{"BTC", "ETH", "SOL", "GOLD"}}, "maxItems": 4}}, "additionalProperties": false}
}
func (s *MarketSkill) Execute(args json.RawMessage) (string, error) {
	var request struct {
		Symbols []string `json:"symbols"`
	}
	if err := json.Unmarshal(args, &request); err != nil {
		return "", errors.New("ungültige Marktanfrage")
	}
	quotes, err := LookupMarketQuotes(context.Background(), request.Symbols)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(quotes)
	if err != nil {
		return "", errors.New("Marktdaten konnten nicht serialisiert werden")
	}
	return string(encoded), nil
}

func LookupMarketQuotes(ctx context.Context, requested []string) ([]MarketQuote, error) {
	marketCache.Lock()
	if time.Since(marketCache.at) < 30*time.Second && len(marketCache.quotes) > 0 {
		quotes := filterMarketQuotes(marketCache.quotes, requested)
		marketCache.Unlock()
		return quotes, nil
	}
	marketCache.Unlock()

	ids := make([]string, 0, len(marketCatalog))
	for _, item := range marketCatalog {
		ids = append(ids, item.ID)
	}
	sort.Strings(ids)
	endpoint := "https://api.coingecko.com/api/v3/simple/price?ids=" + strings.Join(ids, ",") + "&vs_currencies=usd&include_24hr_change=true&include_last_updated_at=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("Marktanfrage konnte nicht vorbereitet werden")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "VGT-AETHEL-MARKET/1.0")
	client := &http.Client{Timeout: 8 * time.Second, CheckRedirect: func(redirect *http.Request, via []*http.Request) error {
		if len(via) > 2 || redirect.URL.Scheme != "https" || !strings.EqualFold(redirect.URL.Hostname(), "api.coingecko.com") {
			return errors.New("Marktdienst-Weiterleitung wurde blockiert")
		}
		return nil
	}}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.New("Marktdienst ist momentan nicht erreichbar")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Marktdienst antwortete mit HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	if err != nil {
		return nil, errors.New("Marktdaten konnten nicht gelesen werden")
	}
	var payload map[string]struct {
		USD     float64 `json:"usd"`
		Change  float64 `json:"usd_24h_change"`
		Updated int64   `json:"last_updated_at"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, errors.New("Marktdatenformat ist ungültig")
	}
	quotes := make([]MarketQuote, 0, len(marketCatalog))
	for symbol, item := range marketCatalog {
		value, ok := payload[item.ID]
		if !ok || value.USD <= 0 {
			continue
		}
		observed := time.Unix(value.Updated, 0).UTC()
		if value.Updated <= 0 {
			observed = time.Now().UTC()
		}
		quotes = append(quotes, MarketQuote{Symbol: symbol, Name: item.Name, Currency: "USD", Price: value.USD, Change24H: value.Change, ObservedAt: observed, Source: "CoinGecko", InstrumentNote: item.Note})
	}
	sort.Slice(quotes, func(i, j int) bool { return quotes[i].Symbol < quotes[j].Symbol })
	if len(quotes) == 0 {
		return nil, errors.New("Marktdienst lieferte keine verwertbaren Kurse")
	}
	marketCache.Lock()
	marketCache.at = time.Now()
	marketCache.quotes = append([]MarketQuote(nil), quotes...)
	marketCache.Unlock()
	return filterMarketQuotes(quotes, requested), nil
}

func filterMarketQuotes(quotes []MarketQuote, requested []string) []MarketQuote {
	if len(requested) == 0 {
		return append([]MarketQuote(nil), quotes...)
	}
	allowed := make(map[string]bool, len(requested))
	for _, symbol := range requested {
		if _, ok := marketCatalog[strings.ToUpper(strings.TrimSpace(symbol))]; ok {
			allowed[strings.ToUpper(strings.TrimSpace(symbol))] = true
		}
	}
	filtered := make([]MarketQuote, 0, len(allowed))
	for _, quote := range quotes {
		if allowed[quote.Symbol] {
			filtered = append(filtered, quote)
		}
	}
	return filtered
}
