package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

var errInvalidAuth = errors.New("invalid API key or JWT")

// requireWriteAuth protects mutating API endpoints using API key or JWT.
func (s *Server) requireWriteAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}

		if err := s.authorizeRequest(r); err != nil {
			if errors.Is(err, errInvalidAuth) {
				writeError(w, http.StatusUnauthorized, "unauthorized")
				return
			}
			writeError(w, http.StatusServiceUnavailable, err.Error())
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) authorizeRequest(r *http.Request) error {
	apiKey := strings.TrimSpace(s.appCfg.Config.Auth.APIKey)
	jwtSecret := strings.TrimSpace(s.appCfg.Config.Auth.JWTSecret)

	if apiKey == "" && jwtSecret == "" {
		return errors.New("API auth is not configured")
	}

	if apiKey != "" {
		if key := strings.TrimSpace(r.Header.Get("X-API-Key")); key != "" && key == apiKey {
			return nil
		}
	}

	if jwtSecret != "" {
		if token := bearerToken(r.Header.Get("Authorization")); token != "" {
			if isValidJWT(token, jwtSecret) {
				return nil
			}
		}
	}

	return errInvalidAuth
}

func bearerToken(authHeader string) string {
	authHeader = strings.TrimSpace(authHeader)
	if authHeader == "" {
		return ""
	}
	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

func isValidJWT(tokenString, secret string) bool {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return false
	}

	headerJSON, err := decodeJWTPart(parts[0])
	if err != nil {
		return false
	}
	var header struct {
		Alg string `json:"alg"`
		Typ string `json:"typ"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return false
	}
	if header.Alg != "HS256" {
		return false
	}

	claimsJSON, err := decodeJWTPart(parts[1])
	if err != nil {
		return false
	}
	var claims struct {
		Exp int64 `json:"exp"`
		Nbf int64 `json:"nbf"`
		Iat int64 `json:"iat"`
	}
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return false
	}

	now := time.Now().Unix()
	if claims.Exp != 0 && now >= claims.Exp {
		return false
	}
	if claims.Nbf != 0 && now < claims.Nbf {
		return false
	}
	if claims.Iat != 0 && now+60 < claims.Iat {
		return false
	}

	input := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(input))
	expected := mac.Sum(nil)

	provided, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}

	return hmac.Equal(expected, provided)
}

func decodeJWTPart(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}
