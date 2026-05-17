package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"oblak/internal/api"
	"oblak/internal/server/apperr"
	"oblak/internal/server/authkey"
	"oblak/internal/signing"
)

type keyStore interface {
	GetAuthKey(ctx context.Context, authId string) (authkey.AuthKey, error)
}

func RequireAuth(store keyStore) api.StrictMiddlewareFunc {
	return func(f api.StrictHandlerFunc, operationID string) api.StrictHandlerFunc {
		if isPublic(operationID) {
			return f
		}
		return func(c *gin.Context, req interface{}) (interface{}, error) {
			credential, signature, err := parseAuthHeader(c.GetHeader("Authorization"))
			if err != nil {
				return nil, apperr.ErrUnauthorized
			}

			dateHeader := c.GetHeader("X-Date")
			timestamp, err := time.Parse(time.RFC3339, dateHeader)
			if err != nil || time.Since(timestamp).Abs() > 15*time.Minute {
				return nil, apperr.ErrUnauthorized
			}

			authKey, err := store.GetAuthKey(c.Request.Context(), credential)
			if err != nil {
				if errors.Is(err, apperr.ErrNotFound) {
					return nil, apperr.ErrUnauthorized
				}
				return nil, err
			}

			body, err := io.ReadAll(c.Request.Body)
			if err != nil {
				return nil, err
			}
			c.Request.Body = io.NopCloser(bytes.NewReader(body))

			expected, err := signing.Sign(c.Request.Method, c.Request.URL.Path, body, dateHeader, authKey.SecretKey)
			if err != nil {
				return nil, err
			}

			if !hmac.Equal([]byte(expected), []byte(signature)) {
				return nil, apperr.ErrUnauthorized
			}

			c.Set("user_id", authKey.UserId)

			return f(c, req)
		}
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

func isPublic(operationID string) bool {
	switch operationID {
	case "GetHealth":
		return true
	default:
		return false
	}
}
