package util

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var reSpace = regexp.MustCompile(`\s+`)

func CleanSpace(s string) string {
	s = strings.TrimSpace(s)
	s = reSpace.ReplaceAllString(s, " ")
	return s
}

func NormalizeText(s string) string {
	s = strings.ToLower(s)
	s = CleanSpace(s)
	return s
}

func FirstNonEmpty(v ...string) string {
	for _, s := range v {
		if strings.TrimSpace(s) != "" {
			return strings.TrimSpace(s)
		}
	}
	return ""
}

func HashText(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:])
}
