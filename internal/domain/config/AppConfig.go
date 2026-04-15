package config

type AppConfig struct {
	Config  Config
	Topics  TopicsFile
	Sources SourcesFile
}

type Config struct {
	DatabaseURL string `yaml:"database_url"`

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
	Enabled     bool     `yaml:"enabled"`
}
