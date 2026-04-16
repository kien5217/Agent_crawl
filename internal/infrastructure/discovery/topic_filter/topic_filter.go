package topicfilter

import (
	"strings"

	"Agent_Crawl/internal/domain/config"
	util "Agent_Crawl/internal/platform"
)

const (
	defaultDiscoveryTopic = "cve"
	minTextMatchScore     = 3
	minURLMatchScore      = 4
)

type TopicHeuristicFilter interface {
	TopicID() string
	MatchText(title, desc string) bool
	MatchURL(rawURL string) bool
}

type TopicMatcher struct {
	keywords   map[string]map[string]int
	heuristics map[string]TopicHeuristicFilter
}

func NewTopicMatcher(tf config.TopicsFile) *TopicMatcher {
	keywords := make(map[string]map[string]int, len(tf.Topics))
	for _, topic := range tf.Topics {
		terms := make(map[string]int, len(topic.Keywords))
		for _, kw := range topic.Keywords {
			term := util.NormalizeText(kw.Term)
			if term == "" {
				continue
			}
			terms[term] = kw.Weight
		}
		keywords[topic.ID] = terms
	}

	heuristics := make(map[string]TopicHeuristicFilter)
	for _, filter := range []TopicHeuristicFilter{
		NewCVETopicFilter(),
		NewAITopicFilter(),
		NewCloudTopicFilter(),
	} {
		heuristics[filter.TopicID()] = filter
	}

	return &TopicMatcher{keywords: keywords, heuristics: heuristics}
}

func (m *TopicMatcher) MatchText(topicIDs []string, title, desc string) (string, bool) {
	if topicID, ok := m.matchHeuristicText(topicIDs, title, desc); ok {
		return topicID, true
	}

	titleN := util.NormalizeText(title)
	descN := util.NormalizeText(desc)
	bestTopicID, bestScore := m.match(topicIDs, func(term string, weight int) int {
		score := 0
		if containsTerm(titleN, term) {
			score += weight * 3
		}
		if containsTerm(descN, term) {
			score += weight
		}
		return score
	})
	if bestScore < minTextMatchScore {
		return "", false
	}
	return bestTopicID, true
}

func (m *TopicMatcher) MatchURL(topicIDs []string, rawURL string) (string, bool) {
	if topicID, ok := m.matchHeuristicURL(topicIDs, rawURL); ok {
		return topicID, true
	}

	urlN := normalizeURLForMatching(rawURL)
	bestTopicID, bestScore := m.match(topicIDs, func(term string, weight int) int {
		if containsTerm(urlN, term) {
			return weight * 2
		}
		return 0
	})
	if bestScore < minURLMatchScore {
		return "", false
	}
	return bestTopicID, true
}

func (m *TopicMatcher) match(topicIDs []string, scoreForTerm func(term string, weight int) int) (string, int) {
	bestTopicID := ""
	bestScore := 0
	for _, topicID := range discoveryTopicIDs(topicIDs) {
		terms, ok := m.keywords[topicID]
		if !ok {
			continue
		}
		score := 0
		for term, weight := range terms {
			score += scoreForTerm(term, weight)
		}
		if score > bestScore {
			bestTopicID = topicID
			bestScore = score
		}
	}
	return bestTopicID, bestScore
}

func (m *TopicMatcher) matchHeuristicText(topicIDs []string, title, desc string) (string, bool) {
	for _, topicID := range discoveryTopicIDs(topicIDs) {
		filter, ok := m.heuristics[topicID]
		if !ok {
			continue
		}
		if filter.MatchText(title, desc) {
			return topicID, true
		}
	}
	return "", false
}

func (m *TopicMatcher) matchHeuristicURL(topicIDs []string, rawURL string) (string, bool) {
	for _, topicID := range discoveryTopicIDs(topicIDs) {
		filter, ok := m.heuristics[topicID]
		if !ok {
			continue
		}
		if filter.MatchURL(rawURL) {
			return topicID, true
		}
	}
	return "", false
}

func discoveryTopicIDs(topicIDs []string) []string {
	if len(topicIDs) == 0 {
		return []string{defaultDiscoveryTopic}
	}

	seen := make(map[string]struct{}, len(topicIDs))
	result := make([]string, 0, len(topicIDs))
	for _, topicID := range topicIDs {
		topicID = strings.TrimSpace(strings.ToLower(topicID))
		if topicID == "" {
			continue
		}
		if _, ok := seen[topicID]; ok {
			continue
		}
		seen[topicID] = struct{}{}
		result = append(result, topicID)
	}
	if len(result) == 0 {
		return []string{defaultDiscoveryTopic}
	}
	return result
}

func normalizeURLForMatching(rawURL string) string {
	replacer := strings.NewReplacer(
		"/", " ",
		"-", " ",
		"_", " ",
		"?", " ",
		"&", " ",
		"=", " ",
		".", " ",
		":", " ",
		"%", " ",
	)
	return util.NormalizeText(replacer.Replace(strings.ToLower(rawURL)))
}

func containsTerm(haystack, term string) bool {
	if haystack == "" || term == "" {
		return false
	}
	searchFrom := 0
	for {
		idx := strings.Index(haystack[searchFrom:], term)
		if idx < 0 {
			return false
		}
		idx += searchFrom
		end := idx + len(term)
		if len(term) > 2 || (isBoundary(haystack, idx-1) && isBoundary(haystack, end)) {
			return true
		}
		searchFrom = idx + len(term)
	}
}

func isBoundary(s string, idx int) bool {
	if idx < 0 || idx >= len(s) {
		return true
	}
	b := s[idx]
	return !(b >= 'a' && b <= 'z' || b >= '0' && b <= '9')
}

func containsAnyTerm(haystack string, terms []string) bool {
	for _, term := range terms {
		if containsTerm(haystack, util.NormalizeText(term)) {
			return true
		}
	}
	return false
}
