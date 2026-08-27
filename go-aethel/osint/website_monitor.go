package osint

// STATUS: DIAMANT VGT SUPREME

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"go-aethel/intelligence"
)

var websiteMonitorScheduler sync.Once

func StartWebsiteMonitorScheduler(ctx context.Context, store *intelligence.Store) {
	if ctx == nil || store == nil {
		return
	}
	websiteMonitorScheduler.Do(func() {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					runDueWebsiteMonitors(ctx, store, now.UTC())
				}
			}
		}()
	})
}

func runDueWebsiteMonitors(ctx context.Context, store *intelligence.Store, now time.Time) {
	executed := 0
	for _, monitor := range store.GetSnapshot().WebsiteMonitors {
		if !monitor.Enabled || monitor.NextCheckAt.After(now) {
			continue
		}
		requestContext, cancel := context.WithTimeout(ctx, 30*time.Second)
		_, _, _ = RunWebsiteMonitor(requestContext, store, monitor.ID)
		cancel()
		executed++
		if executed == 10 {
			return
		}
	}
}

func RunWebsiteMonitor(ctx context.Context, store *intelligence.Store, monitorID string) (intelligence.WebsiteMonitor, *intelligence.WebsiteChange, error) {
	monitor, found := store.GetWebsiteMonitor(strings.TrimSpace(monitorID))
	if !found || !monitor.Enabled {
		return intelligence.WebsiteMonitor{}, nil, errors.New("website monitor unavailable")
	}
	if err := validatePublicCollectorURL(monitor.URL); err != nil {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, monitor.URL, nil)
	if err != nil {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, err
	}
	request.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain,application/json")
	request.Header.Set("User-Agent", "VGT-AETHEL-MONITOR/2.0")
	response, err := newSafeCollectorHTTPClient().Do(request)
	if err != nil {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, errors.New("website collection failed")
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, errors.New("website collection returned non-success status")
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	format := "html"
	switch contentType {
	case "text/html", "application/xhtml+xml", "":
	case "text/plain":
		format = "text"
	case "application/json":
		format = "json"
	default:
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, errors.New("website content type is not monitorable")
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, (8<<20)+1))
	if err != nil || len(raw) == 0 || len(raw) > 8<<20 {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, errors.New("website payload violates size boundary")
	}
	finalURL := response.Request.URL.String()
	if err := validatePublicCollectorURL(finalURL); err != nil {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, errors.New("website redirect failed destination policy")
	}
	headers := map[string]string{"content-type": response.Header.Get("Content-Type"), "etag": response.Header.Get("ETag"), "last-modified": response.Header.Get("Last-Modified")}
	imported, err := store.ImportAcquiredDocument(raw, format, monitor.SourceID, monitor.ID+"."+format, monitor.Domain, intelligence.AcquisitionMetadata{OriginalURL: monitor.URL, FinalURL: finalURL, MIMEType: contentType, ResponseHeaders: headers, FetchedAt: time.Now().UTC()})
	if err != nil {
		_ = store.RecordWebsiteMonitorFailure(monitor.ID)
		return intelligence.WebsiteMonitor{}, nil, err
	}
	return store.RecordWebsiteMonitorSuccess(monitor.ID, imported, response.StatusCode, response.Header.Get("ETag"), response.Header.Get("Last-Modified"))
}
