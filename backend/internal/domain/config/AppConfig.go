package config

type AppConfig struct {
	Config  Config
	Topics  TopicsFile
	Sources SourcesFile
}

type Config struct {
	DatabaseURL string `yaml:"database_url"`

	Auth struct {
		APIKey    string `yaml:"api_key"`
		JWTSecret string `yaml:"jwt_secret"`
	} `yaml:"auth"`

	HTTP struct {
		TimeoutSeconds int    `yaml:"timeout_seconds"`
		UserAgent      string `yaml:"user_agent"`
		MaxBytes       int64  `yaml:"max_bytes"`
	} `yaml:"http"`

	Scheduler struct {
		EnqueueLimitPerSource int `yaml:"enqueue_limit_per_source"`
	} `yaml:"scheduler"`

	Worker struct {
		MaxAttempts         int `yaml:"max_attempts"`
		RetryBackoffSeconds int `yaml:"retry_backoff_seconds"`
		BatchSize           int `yaml:"batch_size"`
		FetchConcurrency    int `yaml:"fetch_concurrency_per_worker"`
	} `yaml:"worker"`

	Classify struct {
		MinScoreToAccept int `yaml:"min_score_to_accept"`
	} `yaml:"classify"`

	Sitemap struct {
		Enabled                bool `yaml:"enabled"`
		MaxURLsPerSourcePerRun int  `yaml:"max_urls_per_source_per_run"`
		MaxSitemapsPerIndex    int  `yaml:"max_sitemaps_per_index"`
		HTTPTimeoutSeconds     int  `yaml:"http_timeout_seconds"`
	} `yaml:"sitemap"`
}

type TopicsFile struct {
	Topics []Topic `yaml:"topics"`
}

type Topic struct {
	ID       string    `yaml:"id"`
	Name     string    `yaml:"name"`
	Keywords []Keyword `yaml:"keywords"`
}

type Keyword struct {
	Term   string `yaml:"term"`
	Weight int    `yaml:"weight"`
}

type SourcesFile struct {
	Sources []Source `yaml:"sources"`
}

type Source struct {
	ID          string   `yaml:"id"`
	Name        string   `yaml:"name"`
	Domain      string   `yaml:"domain"`
	RSSURL      string   `yaml:"rss_url"`
	SitemapURLs []string `yaml:"sitemap_urls"`
	TopicIDs    []string `yaml:"topic_ids"`
	Enabled     bool     `yaml:"enabled"`

	// ExcludeChildSitemapPatterns: skip child sitemaps whose URL contains any of these substrings.
	// Useful for filtering out category/tag/author/page sitemaps.
	ExcludeChildSitemapPatterns []string `yaml:"exclude_child_sitemap_patterns"`

	// SitemapMinLastmod: skip child sitemaps last-modified before this date (RFC3339 or YYYY-MM-DD).
	// Useful for ignoring old archive sitemaps.
	SitemapMinLastmod string `yaml:"sitemap_min_lastmod"`
}
