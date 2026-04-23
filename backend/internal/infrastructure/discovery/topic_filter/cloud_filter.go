package topicfilter

import util "Agent_Crawl/internal/platform"

var cloudTextTerms = []string{
	"cloud",
	"điện toán đám mây",
	"aws",
	"amazon web services",
	"azure",
	"gcp",
	"google cloud",
	"kubernetes",
	"k8s",
	"docker",
	"container",
	"serverless",
	"terraform",
	"devops",
	"observability",
	"platform engineering",
	"cloud native",
}

var cloudURLTerms = []string{
	"/cloud/",
	"aws",
	"azure",
	"gcp",
	"google-cloud",
	"kubernetes",
	"k8s",
	"docker",
	"serverless",
	"terraform",
	"cloud-native",
}

type CloudTopicFilter struct{}

func NewCloudTopicFilter() TopicHeuristicFilter {
	return CloudTopicFilter{}
}

func (CloudTopicFilter) TopicID() string {
	return "cloud"
}

func (CloudTopicFilter) MatchText(title, desc string) bool {
	text := util.NormalizeText(title + " " + desc)
	return containsAnyTerm(text, cloudTextTerms)
}

func (CloudTopicFilter) MatchURL(rawURL string) bool {
	urlN := normalizeURLForMatching(rawURL)
	return containsAnyTerm(urlN, cloudURLTerms)
}
