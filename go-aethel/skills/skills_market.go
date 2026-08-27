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
	Category       string    `json:"category"` // commodity, crypto, index, forex
	Currency       string    `json:"currency"`
	Price          float64   `json:"price"`
	Change24H      float64   `json:"change_24h_percent"`
	ObservedAt     time.Time `json:"observed_at"`
	Source         string    `json:"source"`
	InstrumentNote string    `json:"instrument_note,omitempty"`
}

type MarketInstrument struct {
	Symbol     string
	Name       string
	Category   string // commodity, crypto, index, forex
	Currency   string
	SourceType string // "coingecko" or "yahoo"
	SourceID   string
	Note       string
}

var marketCatalog = []MarketInstrument{
	// Commodities (Rohstoffe)
	{Symbol: "BRENT", Name: "Brent Crude Oil", Category: "commodity", Currency: "USD", SourceType: "yahoo", SourceID: "BZ=F", Note: "Nordsee Öl-Futures (USD/bbl)"},
	{Symbol: "WTI", Name: "WTI Crude Oil", Category: "commodity", Currency: "USD", SourceType: "yahoo", SourceID: "CL=F", Note: "US Crude Oil Futures (USD/bbl)"},
	{Symbol: "GOLD", Name: "Gold Spot", Category: "commodity", Currency: "USD", SourceType: "yahoo", SourceID: "GC=F", Note: "Gold Futures (USD/oz)"},
	{Symbol: "SILVER", Name: "Silber Spot", Category: "commodity", Currency: "USD", SourceType: "yahoo", SourceID: "SI=F", Note: "Silber Futures (USD/oz)"},
	{Symbol: "NATGAS", Name: "Erdgas (Natural Gas)", Category: "commodity", Currency: "USD", SourceType: "yahoo", SourceID: "NG=F", Note: "Natural Gas Futures (USD/MMBtu)"},
	{Symbol: "COPPER", Name: "Kupfer", Category: "commodity", Currency: "USD", SourceType: "yahoo", SourceID: "HG=F", Note: "Copper Futures (USD/lb)"},

	// Cryptocurrencies (Krypto)
	{Symbol: "BTC", Name: "Bitcoin", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "bitcoin"},
	{Symbol: "ETH", Name: "Ethereum", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "ethereum"},
	{Symbol: "SOL", Name: "Solana", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "solana"},
	{Symbol: "XRP", Name: "Ripple (XRP)", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "ripple"},
	{Symbol: "BNB", Name: "Binance Coin", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "binancecoin"},
	{Symbol: "DOGE", Name: "Dogecoin", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "dogecoin"},
	{Symbol: "ADA", Name: "Cardano", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "cardano"},
	{Symbol: "AVAX", Name: "Avalanche", Category: "crypto", Currency: "USD", SourceType: "coingecko", SourceID: "avalanche-2"},

	// Indices & Equities (Indizes & Aktien)
	{Symbol: "DAX", Name: "DAX 40", Category: "index", Currency: "EUR", SourceType: "yahoo", SourceID: "^GDAXI", Note: "Deutscher Aktienindex"},
	{Symbol: "SP500", Name: "S&P 500", Category: "index", Currency: "USD", SourceType: "yahoo", SourceID: "^GSPC", Note: "US Benchmark Index"},
	{Symbol: "NASDAQ", Name: "Nasdaq 100", Category: "index", Currency: "USD", SourceType: "yahoo", SourceID: "^IXIC", Note: "Tech Index"},
	{Symbol: "NVDA", Name: "NVIDIA Corp.", Category: "index", Currency: "USD", SourceType: "yahoo", SourceID: "NVDA", Note: "AI Hardware Leader"},
	{Symbol: "AAPL", Name: "Apple Inc.", Category: "index", Currency: "USD", SourceType: "yahoo", SourceID: "AAPL", Note: "Consumer Tech"},
	{Symbol: "TSLA", Name: "Tesla Inc.", Category: "index", Currency: "USD", SourceType: "yahoo", SourceID: "TSLA", Note: "EV & Energy"},
	{Symbol: "MSFT", Name: "Microsoft", Category: "index", Currency: "USD", SourceType: "yahoo", SourceID: "MSFT", Note: "Cloud & Enterprise"},

	// Forex (Devisen)
	{Symbol: "EURUSD", Name: "EUR / USD", Category: "forex", Currency: "USD", SourceType: "yahoo", SourceID: "EURUSD=X"},
	{Symbol: "GBPUSD", Name: "GBP / USD", Category: "forex", Currency: "USD", SourceType: "yahoo", SourceID: "GBPUSD=X"},
	{Symbol: "USDJPY", Name: "USD / JPY", Category: "forex", Currency: "JPY", SourceType: "yahoo", SourceID: "JPY=X"},
}

var marketCache struct {
	sync.Mutex
	at     time.Time
	quotes []MarketQuote
}

func (s *MarketSkill) Name() string { return "market_lookup" }
func (s *MarketSkill) Description() string {
	return "Liest aktuelle Markt- und Finanzkurse für Rohstoffe (Öl/Brent/WTI, Gold, Silber, Erdgas, Kupfer), Krypto (BTC, ETH, SOL, XRP, BNB, DOGE), Indizes/Aktien (DAX, SP500, NASDAQ, NVDA, AAPL, TSLA) und Devisen (EURUSD). Öffnet das Märkte-Modul in Sphere."
}
func (s *MarketSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *MarketSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"symbols": map[string]interface{}{
				"type":     "array",
				"items":    map[string]interface{}{"type": "string"},
				"maxItems": 50,
			},
		},
		"additionalProperties": false,
	}
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

	var (
		wg            sync.WaitGroup
		quotesLock    sync.Mutex
		fetchedQuotes []MarketQuote
	)

	// Group CoinGecko instruments
	var geckoIDs []string
	geckoMap := make(map[string]MarketInstrument)
	for _, inst := range marketCatalog {
		if inst.SourceType == "coingecko" {
			geckoIDs = append(geckoIDs, inst.SourceID)
			geckoMap[inst.SourceID] = inst
		}
	}

	// 1. Fetch CoinGecko
	if len(geckoIDs) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cgQuotes := fetchCoinGeckoQuotes(ctx, geckoIDs, geckoMap)
			quotesLock.Lock()
			fetchedQuotes = append(fetchedQuotes, cgQuotes...)
			quotesLock.Unlock()
		}()
	}

	// 2. Fetch Yahoo Finance symbols concurrently
	for _, inst := range marketCatalog {
		if inst.SourceType == "yahoo" {
			wg.Add(1)
			go func(instrument MarketInstrument) {
				defer wg.Done()
				quote, err := fetchYahooQuote(ctx, instrument)
				if err == nil && quote.Price > 0 {
					quotesLock.Lock()
					fetchedQuotes = append(fetchedQuotes, quote)
					quotesLock.Unlock()
				}
			}(inst)
		}
	}

	wg.Wait()

	if len(fetchedQuotes) == 0 {
		// Fallback to cached quotes if live fetch failed completely
		marketCache.Lock()
		defer marketCache.Unlock()
		if len(marketCache.quotes) > 0 {
			return filterMarketQuotes(marketCache.quotes, requested), nil
		}
		return nil, errors.New("Marktdienst ist momentan nicht erreichbar")
	}

	// Sort alphabetically by category, then symbol
	sort.Slice(fetchedQuotes, func(i, j int) bool {
		if fetchedQuotes[i].Category != fetchedQuotes[j].Category {
			return fetchedQuotes[i].Category < fetchedQuotes[j].Category
		}
		return fetchedQuotes[i].Symbol < fetchedQuotes[j].Symbol
	})

	marketCache.Lock()
	marketCache.at = time.Now()
	marketCache.quotes = append([]MarketQuote(nil), fetchedQuotes...)
	marketCache.Unlock()

	return filterMarketQuotes(fetchedQuotes, requested), nil
}

func fetchCoinGeckoQuotes(ctx context.Context, ids []string, instMap map[string]MarketInstrument) []MarketQuote {
	endpoint := "https://api.coingecko.com/api/v3/simple/price?ids=" + strings.Join(ids, ",") + "&vs_currencies=usd&include_24hr_change=true&include_last_updated_at=true"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) VGT-AETHEL/2.0")

	client := &http.Client{Timeout: 6 * time.Second}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return nil
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256<<10))
	if err != nil {
		return nil
	}

	var payload map[string]struct {
		USD     float64 `json:"usd"`
		Change  float64 `json:"usd_24h_change"`
		Updated int64   `json:"last_updated_at"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}

	var results []MarketQuote
	for id, item := range payload {
		inst, ok := instMap[id]
		if !ok || item.USD <= 0 {
			continue
		}
		observed := time.Unix(item.Updated, 0).UTC()
		if item.Updated <= 0 {
			observed = time.Now().UTC()
		}
		results = append(results, MarketQuote{
			Symbol:         inst.Symbol,
			Name:           inst.Name,
			Category:       inst.Category,
			Currency:       inst.Currency,
			Price:          item.USD,
			Change24H:      item.Change,
			ObservedAt:     observed,
			Source:         "CoinGecko",
			InstrumentNote: inst.Note,
		})
	}
	return results
}

func fetchYahooQuote(ctx context.Context, inst MarketInstrument) (MarketQuote, error) {
	endpoint := fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d", inst.SourceID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return MarketQuote{}, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return MarketQuote{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return MarketQuote{}, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 128<<10))
	if err != nil {
		return MarketQuote{}, err
	}

	var yahooResp struct {
		Chart struct {
			Result []struct {
				Meta struct {
					RegularMarketPrice float64 `json:"regularMarketPrice"`
					PreviousClose      float64 `json:"previousClose"`
					ChartPreviousClose float64 `json:"chartPreviousClose"`
					Currency           string  `json:"currency"`
				} `json:"meta"`
			} `json:"result"`
		} `json:"chart"`
	}

	if err := json.Unmarshal(body, &yahooResp); err != nil || len(yahooResp.Chart.Result) == 0 {
		return MarketQuote{}, errors.New("parse error")
	}

	meta := yahooResp.Chart.Result[0].Meta
	price := meta.RegularMarketPrice
	if price <= 0 {
		return MarketQuote{}, errors.New("zero price")
	}

	prevClose := meta.PreviousClose
	if prevClose <= 0 {
		prevClose = meta.ChartPreviousClose
	}

	changePct := 0.0
	if prevClose > 0 {
		changePct = ((price - prevClose) / prevClose) * 100.0
	}

	curr := inst.Currency
	if meta.Currency != "" {
		curr = strings.ToUpper(meta.Currency)
	}

	return MarketQuote{
		Symbol:         inst.Symbol,
		Name:           inst.Name,
		Category:       inst.Category,
		Currency:       curr,
		Price:          price,
		Change24H:      changePct,
		ObservedAt:     time.Now().UTC(),
		Source:         "Yahoo Finance",
		InstrumentNote: inst.Note,
	}, nil
}

func filterMarketQuotes(quotes []MarketQuote, requested []string) []MarketQuote {
	if len(requested) == 0 {
		return append([]MarketQuote(nil), quotes...)
	}
	allowed := make(map[string]bool, len(requested))
	for _, symbol := range requested {
		allowed[strings.ToUpper(strings.TrimSpace(symbol))] = true
	}
	filtered := make([]MarketQuote, 0, len(allowed))
	for _, quote := range quotes {
		if allowed[quote.Symbol] {
			filtered = append(filtered, quote)
		}
	}
	return filtered
}

type SphereMarketOverviewSkill struct{}

func (s *SphereMarketOverviewSkill) Name() string { return "sphere_market_overview" }
func (s *SphereMarketOverviewSkill) Description() string {
	return "Zeigt Marktübersicht und Echtzeitkurse für Rohstoffe (Öl, Gold), Aktien, Indizes und Krypto in Sphere."
}
func (s *SphereMarketOverviewSkill) RiskLevel() security.RiskLevel { return security.RiskLow }
func (s *SphereMarketOverviewSkill) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"assets":    map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"symbols":   map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
			"region":    map[string]interface{}{"type": "string"},
			"timeframe": map[string]interface{}{"type": "string"},
		},
		"additionalProperties": true,
	}
}
func (s *SphereMarketOverviewSkill) Execute(args json.RawMessage) (string, error) {
	var input struct {
		Assets  []string `json:"assets"`
		Symbols []string `json:"symbols"`
	}
	_ = json.Unmarshal(args, &input)
	reqSymbols := append(input.Assets, input.Symbols...)
	var cleanSymbols []string
	for _, sym := range reqSymbols {
		norm := strings.ToUpper(strings.TrimSpace(sym))
		norm = strings.ReplaceAll(norm, "/", "")
		norm = strings.ReplaceAll(norm, " ", "")
		if norm == "OIL" || norm == "ÖL" {
			cleanSymbols = append(cleanSymbols, "BRENT", "WTI")
		} else if norm == "SP500" || norm == "SPX" || norm == "S&P500" {
			cleanSymbols = append(cleanSymbols, "SP500")
		} else if norm == "NASDAQ" || norm == "NDX" {
			cleanSymbols = append(cleanSymbols, "NASDAQ")
		} else {
			cleanSymbols = append(cleanSymbols, norm)
		}
	}
	quotes, err := LookupMarketQuotes(context.Background(), cleanSymbols)
	if err != nil {
		return "", err
	}
	encoded, err := json.Marshal(quotes)
	if err != nil {
		return "", errors.New("Marktdaten konnten nicht serialisiert werden")
	}
	return string(encoded), nil
}


