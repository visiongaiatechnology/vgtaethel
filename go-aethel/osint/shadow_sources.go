package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

type ShadowSource struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Type      string `json:"type"` // rss | telegram | web
	Domain    string `json:"domain"`
	Enabled   bool   `json:"enabled"`
	Priority  int    `json:"priority"`
	LastState string `json:"last_state,omitempty"`
	LastError string `json:"last_error,omitempty"`
	LastFetch string `json:"last_fetch,omitempty"`
}

func shadowSourceID(name, rawURL string) string {
	sum := sha256.Sum256([]byte(strings.ToLower(strings.TrimSpace(name + "|" + rawURL))))
	return "shadow-src-" + hex.EncodeToString(sum[:8])
}

func defaultShadowSources() []ShadowSource {
	// Pages without a direct RSS endpoint remain editable WEB sources and are
	// never silently treated as XML feeds.
	raw := []struct{ name, url, domain, kind string }{
		{"Janes", "https://www.janes.com/rss/news", "military", "rss"}, {"ISW Ukraine", "https://isw.pub/UkraineRSS", "military", "rss"},
		{"Defense News", "https://www.defensenews.com/arc/outboundfeeds/rss/", "military", "rss"}, {"Long War Journal", "https://www.longwarjournal.org/feed", "military", "rss"},
		{"UK Ministry of Defence", "https://www.gov.uk/government/organisations/ministry-of-defence/about.rss", "official", "rss"}, {"SouthFront", "https://southfront.press/feed/", "military", "rss"},
		{"USNI News", "https://news.usni.org/feed", "military", "rss"}, {"United Nations News", "https://news.un.org/feed/subscribe/en/news/all/rss.xml", "official", "rss"},
		{"White House Presidential Actions", "https://www.whitehouse.gov/presidential-actions/feed/", "official", "rss"}, {"European Commission Press Corner", "https://ec.europa.eu/commission/presscorner/api/rss?language=en", "official", "rss"},
		{"Bundesregierung", "https://www.bundesregierung.de/service/rss/breg-de/1151242/feed.xml", "official", "rss"}, {"Kremlin President", "https://en.kremlin.ru/events/president/news/feed", "official", "rss"},
		{"China Foreign Ministry", "https://www.fmprc.gov.cn/mfa_eng/xwfw_665399/s2459_665414/2460_665416/rss.xml", "official", "rss"}, {"Al Jazeera", "https://www.aljazeera.com/xml/rss/all.xml", "geopolitics", "rss"},
		{"African Union Press", "https://au.int/en/press-releases", "official", "web"}, {"South African Government", "https://www.gov.za/rss-feeds", "official", "web"},
		{"International Crisis Group", "https://www.crisisgroup.org/rss-0", "geopolitics", "rss"}, {"ZeroHedge", "https://www.zerohedge.com/feed", "economy", "rss"},
		{"Bank for International Settlements", "https://www.bis.org/publ/rss.xml", "economy", "rss"}, {"IMF News", "https://www.imf.org/en/News/RSS", "economy", "rss"},
		{"OilPrice", "https://oilprice.com/rss/main", "energy", "rss"}, {"World Bank News", "https://www.worldbank.org/en/news/rss", "economy", "rss"},
		{"US EIA", "https://www.eia.gov/tools/rssfeeds/", "energy", "web"}, {"BleepingComputer", "https://www.bleepingcomputer.com/feed/", "cyber", "rss"},
		{"The Hacker News", "https://thehackernews.com/rss.xml", "cyber", "rss"}, {"MIT Technology Review", "https://www.technologyreview.com/feed/", "technology", "rss"},
		{"CERT-Bund WID", "https://wid.cert-bund.de/content/public/securityAdvisory/rss", "cyber", "rss"}, {"Ars Technica", "https://feeds.arstechnica.com/arstechnica/index", "technology", "rss"},
		{"CISA Updates", "https://www.cisa.gov/about/contact-us/subscribe-updates-cisa", "cyber", "web"}, {"FortiGuard", "https://www.fortiguard.com/rss-feeds", "cyber", "web"},
		{"War on the Rocks", "https://warontherocks.com/feed/", "military", "rss"}, {"Breaking Defense", "https://breakingdefense.com/full-rss-feed/", "military", "rss"},
		{"Naval News", "https://www.navalnews.com/feed/", "military", "rss"}, {"The War Zone", "https://www.twz.com/feed/", "military", "rss"},
		{"Shephard Media", "https://www.shephardmedia.com/news/feed/", "military", "rss"}, {"UK Defence Journal", "https://ukdefencejournal.org.uk/feed/", "military", "rss"},
		{"Army Technology", "https://www.army-technology.com/feed/", "military", "rss"}, {"Naval Technology", "https://www.naval-technology.com/news/feed/", "military", "rss"},
		{"Airforce Technology", "https://www.airforce-technology.com/feed/", "military", "rss"}, {"US Navy RSS", "https://www.navy.mil/Resources/RSS-Feeds/", "official", "web"},
		{"US Space Force RSS", "https://www.spaceforce.mil/RSS/", "official", "web"}, {"Bundeswehr", "https://www.bundeswehr.de/service/rss/de/517054/feed", "official", "rss"},
		{"BMVg", "https://www.bmvg.de/de/service/aktuelle-nachrichten-per-rss-feed-13640", "official", "web"}, {"ISW Research", "https://www.iswresearch.org/feeds/posts/default", "military", "rss"},
		{"SpaceNews", "https://spacenews.com/feed/", "space", "rss"}, {"Defense Update", "https://www.defense-update.com/feed", "military", "rss"},
		{"SIPRI", "https://www.sipri.org/rss/combined.xml", "military", "rss"}, {"Chatham House", "https://www.chathamhouse.org/rss-feeds", "geopolitics", "web"},
		{"RUSI", "https://www.rusi.org/rusi-rss-feeds", "military", "web"}, {"Brookings Research", "https://www.brookings.edu/feeds/rss/research/", "geopolitics", "rss"},
		{"Jamestown Foundation", "https://jamestown.org/feed/", "geopolitics", "rss"}, {"Lowy Interpreter", "https://www.lowyinstitute.org/the-interpreter/rss.xml", "geopolitics", "rss"},
		{"The Diplomat", "https://thediplomat.com/feed/", "geopolitics", "rss"}, {"Geopolitical Futures", "https://geopoliticalfutures.com/feed/", "geopolitics", "rss"},
		{"Foreign Policy", "https://foreignpolicy.com/feed/", "geopolitics", "rss"}, {"Deutsche Welle English", "https://rss.dw.com/rdf/rss-en-all", "geopolitics", "rss"},
		{"South China Morning Post", "https://www.scmp.com/rss/91/feed", "geopolitics", "rss"}, {"Nikkei Asia", "https://asia.nikkei.com/rss/feed/nar", "economy", "rss"},
		{"Moscow Times", "https://www.themoscowtimes.com/rss/news", "geopolitics", "rss"}, {"Meduza English", "https://meduza.io/rss/en/all", "geopolitics", "rss"},
		{"Middle East Eye", "https://www.middleeasteye.net/rss", "geopolitics", "rss"}, {"Anadolu Agency", "https://www.aa.com.tr/en/rss/default?cat=guncel", "geopolitics", "rss"},
		{"Bank of England", "https://www.bankofengland.co.uk/rss/news", "economy", "rss"}, {"Bank of Japan", "https://www.boj.or.jp/en/rss/whatsnew.xml", "economy", "rss"},
		{"Bundesbank", "https://www.bundesbank.de/en/homepage/rss/deutsche-bundesbank-s-rss-feed-620440", "economy", "rss"}, {"Swiss National Bank", "https://www.snb.ch/public/en/rss/news", "economy", "rss"},
		{"Bank of Canada", "https://www.bankofcanada.ca/feed/", "economy", "rss"}, {"Reserve Bank of Australia", "https://www.rba.gov.au/rss/rss-cb-rdp.xml", "economy", "rss"},
		{"Financial Times Central Banks", "https://www.ft.com/central-banks?format=rss", "economy", "rss"}, {"OPEC", "https://www.opec.org/opec_web/en/feeds.htm", "energy", "web"},
		{"Oil and Gas Journal", "https://www.ogj.com/rss", "energy", "rss"}, {"Sanctions News", "https://sanctionsnews.bakermckenzie.com/feed/", "sanctions", "rss"},
		{"Microsoft Security", "https://www.microsoft.com/security/blog/feed/", "cyber", "rss"}, {"CrowdStrike", "https://www.crowdstrike.com/en-us/blog/feed/", "cyber", "rss"},
		{"Google Threat Intelligence", "https://cloud.google.com/blog/topics/threat-intelligence/rss", "cyber", "rss"}, {"Krebs on Security", "https://krebsonsecurity.com/feed/", "cyber", "rss"},
		{"SANS ISC", "https://isc.sans.edu/rssfeed.xml", "cyber", "rss"}, {"The Record", "https://therecord.media/feed", "cyber", "rss"},
		{"SecurityWeek", "https://www.securityweek.com/feed/", "cyber", "rss"}, {"Zero Day Initiative", "https://www.thezdi.com/blog?format=rss", "cyber", "rss"},
		{"SentinelLabs", "https://www.sentinelone.com/labs/feed/", "cyber", "rss"}, {"PortSwigger Research", "https://portswigger.net/research/rss", "cyber", "rss"},
		{"Press TV", "https://www.presstv.ir/rss.xml", "geopolitics", "rss"}, {"Tehran Times", "https://www.tehrantimes.com/rss", "geopolitics", "rss"},
		{"Times of Israel", "https://www.timesofisrael.com/feed/", "geopolitics", "rss"}, {"Al Arabiya English", "https://english.alarabiya.net/feed/flipboard/en.xml", "geopolitics", "rss"},
		{"Asharq Al-Awsat English", "https://english.aawsat.com/feed", "geopolitics", "rss"}, {"MercoPress", "https://en.mercopress.com/rss", "geopolitics", "rss"},
		{"InSight Crime", "https://insightcrime.org/feed/", "security", "rss"}, {"Folha de S.Paulo", "https://feeds.folha.uol.com.br/emcimadahora/rss091.xml", "geopolitics", "rss"},
		{"ISS Africa", "https://issafrica.org/feed", "security", "rss"}, {"The Africa Report", "https://www.theafricareport.com/feed/", "geopolitics", "rss"},
		{"Telegram militaernews", "https://t.me/s/militaernews", "military", "telegram"},
	}
	result := make([]ShadowSource, 0, len(raw))
	for _, source := range raw {
		result = append(result, ShadowSource{ID: shadowSourceID(source.name, source.url), Name: source.name, URL: source.url, Type: source.kind, Domain: source.domain, Enabled: true, Priority: 3})
	}
	return result
}
