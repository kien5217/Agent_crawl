package topicfilter

import util "Agent_Crawl/internal/platform"

var aiTextTerms = []string{
	"artificial intelligence",
	"trí tuệ nhân tạo",
	"machine learning",
	"học máy",
	"deep learning",
	"neural network",
	"llm",
	"gpt",
	"openai",
	"anthropic",
	"gemini",
	"claude",
	"copilot",
	"prompt engineering",
	"generative ai",
	"ai model",
	"foundation model",
}

var aiURLTerms = []string{
	"/ai/",
	"artificial-intelligence",
	"machine-learning",
	"deep-learning",
	"generative-ai",
	"llm",
	"openai",
	"gpt",
	"gemini",
	"claude",
	"copilot",
}

type AITopicFilter struct{}

func NewAITopicFilter() TopicHeuristicFilter {
	return AITopicFilter{}
}

func (AITopicFilter) TopicID() string {
	return "ai"
}

func (AITopicFilter) MatchText(title, desc string) bool {
	text := util.NormalizeText(title + " " + desc)
	return containsAnyTerm(text, aiTextTerms)
}

func (AITopicFilter) MatchURL(rawURL string) bool {
	urlN := normalizeURLForMatching(rawURL)
	return containsAnyTerm(urlN, aiURLTerms)
}
