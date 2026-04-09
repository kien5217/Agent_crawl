package discovery

import (
	"context"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"Agent_Crawl/internal/config"
	"Agent_Crawl/internal/db"

	"github.com/jackc/pgx/v5"
	"github.com/rs/zerolog/log"
)

type SitemapDiscovery struct {
	cfg     config.Config
	sources config.SourcesFile
	client  *http.Client
}

func NewSitemapDiscovery(cfg config.Config, sources config.SourcesFile) *SitemapDiscovery {
	to := cfg.Sitemap.HTTPTimeoutSeconds
	if to <= 0 {
		to = cfg.HTTP.TimeoutSeconds
	}
	return &SitemapDiscovery{
		cfg:     cfg,
		sources: sources,
		client:  &http.Client{Timeout: time.Duration(to) * time.Second},
	}
}

func (d *SitemapDiscovery) Enqueue(ctx context.Context, conn *pgx.Conn) (int, error) {
	if !d.cfg.Sitemap.Enabled {
		return 0, nil
	}

	total := 0
	for _, s := range d.sources.Sources {
		if !s.Enabled || len(s.SitemapURLs) == 0 {
			continue
		}
		log.Info().Str("source", s.ID).Int("sitemaps", len(s.SitemapURLs)).Msg("sitemap discovery start")
		added, err := d.enqueueSource(ctx, conn, s)
		if err != nil {
			log.Warn().Err(err).Str("source", s.ID).Msg("sitemap discovery failed")
			continue
		}
		log.Info().Str("source", s.ID).Int("enqueued", added).Msg("sitemap discovery done")
		total += added
	}
	return total, nil
}

func (d *SitemapDiscovery) enqueueSource(ctx context.Context, conn *pgx.Conn, s config.Source) (int, error) {
	limit := d.cfg.Sitemap.MaxURLsPerSourcePerRun
	if limit <= 0 {
		limit = 20000
	}

	added := 0
	for _, smURL := range s.SitemapURLs {
		if added >= limit {
			break
		}
		n, err := d.processSitemapAny(ctx, conn, s, smURL, limit-added, 0)
		if err != nil {
			log.Warn().Err(err).Str("sitemap", smURL).Msg("process sitemap failed")
			continue
		}
		added += n
	}
	return added, nil
}

func (d *SitemapDiscovery) processSitemapAny(ctx context.Context, conn *pgx.Conn, s config.Source, sitemapURL string, remaining int, depth int) (int, error) {
	if remaining <= 0 {
		return 0, nil
	}
	if depth > 5 {
		return 0, errors.New("sitemap depth too deep")
	}

	b, err := d.httpGetBytes(ctx, sitemapURL)
	if err != nil {
		return 0, err
	}

	// detect which root: sitemapindex or urlset
	root := detectRootTag(b)
	switch root {
	case "sitemapindex":
		var idx sitemapIndex
		if err := xml.Unmarshal(b, &idx); err != nil {
			return 0, err
		}
		maxChildren := d.cfg.Sitemap.MaxSitemapsPerIndex
		if maxChildren <= 0 {
			maxChildren = 200
		}

		added := 0
		for i, child := range idx.Sitemaps {
			if i >= maxChildren || added >= remaining {
				break
			}
			loc := strings.TrimSpace(child.Loc)
			if loc == "" {
				continue
			}
			n, err := d.processSitemapAny(ctx, conn, s, loc, remaining-added, depth+1)
			if err != nil {
				log.Warn().Err(err).Str("child", loc).Msg("child sitemap failed")
				continue
			}
			added += n
		}
		return added, nil

	case "urlset":
		var us urlSet
		if err := xml.Unmarshal(b, &us); err != nil {
			return 0, err
		}
		added := 0
		for _, u := range us.URLs {
			if added >= remaining {
				break
			}
			loc := strings.TrimSpace(u.Loc)
			if loc == "" {
				continue
			}
			// Lọc CVE theo URL (2A)
			if !LooksLikeCVEByURL(loc) {
				continue
			}
			norm, domain, ok := NormalizeURL(loc)
			if !ok {
				continue
			}
			ins, err := db.EnqueueURL(ctx, conn, "cve", s.ID, norm, domain, 0) // priority thấp hơn RSS
			if err != nil {
				log.Warn().Err(err).Str("url", norm).Msg("enqueue failed")
				continue
			}
			if ins {
				added++
			}
		}
		return added, nil

	default:
		return 0, errors.New("unknown sitemap root")
	}
}

func (d *SitemapDiscovery) httpGetBytes(ctx context.Context, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", d.cfg.HTTP.UserAgent)
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return nil, errors.New("http status " + resp.Status)
	}
	// sitemap có thể lớn; MVP đọc vừa phải (10MB)
	r := io.LimitReader(resp.Body, 10*1024*1024)
	return io.ReadAll(r)
}

func detectRootTag(b []byte) string {
	s := strings.ToLower(string(b))
	// crude but effective MVP
	if strings.Contains(s, "<sitemapindex") {
		return "sitemapindex"
	}
	if strings.Contains(s, "<urlset") {
		return "urlset"
	}
	return ""
}

type sitemapIndex struct {
	XMLName  xml.Name      `xml:"sitemapindex"`
	Sitemaps []sitemapItem `xml:"sitemap"`
}
type sitemapItem struct {
	Loc string `xml:"loc"`
}

type urlSet struct {
	XMLName xml.Name  `xml:"urlset"`
	URLs    []urlItem `xml:"url"`
}
type urlItem struct {
	Loc string `xml:"loc"`
}
