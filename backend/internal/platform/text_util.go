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

func SimHashText(s string) uint64 {
	tokens := strings.Fields(NormalizeText(s))
	if len(tokens) == 0 {
		return 0
	}

	var vector [64]int
	for _, token := range tokens {
		h := sha256.Sum256([]byte(token))
		for bit := 0; bit < 64; bit++ {
			if (h[bit/8]>>(7-uint(bit)%8))&1 == 1 {
				vector[bit]++
			} else {
				vector[bit]--
			}
		}
	}

	var fingerprint uint64
	for bit := 0; bit < 64; bit++ {
		if vector[bit] > 0 {
			fingerprint |= 1 << uint(63-bit)
		}
	}
	return fingerprint
}
