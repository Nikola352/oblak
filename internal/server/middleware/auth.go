package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"oblak/internal/server/apperr"
	"oblak/internal/server/authkey"
	"oblak/internal/signing"
)

type keyStore interface {
	GetAuthKey(ctx context.Context, authId string) (authkey.AuthKey, error)
}

func RequireAuth(store keyStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isPublic(c.FullPath()) {
			c.Next()
			return
		}

		credential, signature, err := parseAuthHeader(c.GetHeader("Authorization"))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		}

		dateHeader := c.GetHeader("X-Date")
		timestamp, err := time.Parse(time.RFC3339, dateHeader)
		if err != nil || time.Since(timestamp).Abs() > 15*time.Minute {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		}

		authKey, err := store.GetAuthKey(c.Request.Context(), credential)
		if err != nil {
			if errors.Is(err, apperr.ErrNotFound) {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			}
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewReader(body))

		expected, err := signing.Sign(c.Request.Method, c.Request.URL.Path, body, dateHeader, authKey.SecretKey)
		if err != nil {
			c.AbortWithStatus(http.StatusInternalServerError)
			return
		}

		if !hmac.Equal([]byte(expected), []byte(signature)) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		}

		c.Set("user_id", authKey.UserId)
		c.Next()
	}
}

func parseAuthHeader(header string) (credential, signature string, err error) {
	const scheme = "HMAC-SHA256 "
	if !strings.HasPrefix(header, scheme) {
		return "", "", errors.New("invalid scheme")
	}
	params := make(map[string]string)
	for _, part := range strings.Split(header[len(scheme):], ", ") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return "", "", errors.New("malformed parameter")
		}
		params[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	credential = params["Credential"]
	signature = params["Signature"]
	if credential == "" || signature == "" {
		return "", "", errors.New("missing credential or signature")
	}
	return credential, signature, nil
}

func isPublic(path string) bool {
	switch {
	case path == "/health":
		return true
	case strings.HasPrefix(path, "/execute/"):
		return true
	default:
		return false
	}
}
