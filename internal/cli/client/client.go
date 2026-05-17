package client

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"oblak/internal/api"
	cliconfig "oblak/internal/cli/config"
	"oblak/internal/signing"
)

func NewSignedClient(profile cliconfig.Profile) (*api.ClientWithResponses, error) {
	return api.NewClientWithResponses(
		"http://"+profile.Endpoint,
		api.WithRequestEditorFn(signingEditor(profile.AuthID, profile.SecretKey)),
	)
}

func signingEditor(authID, secretKey string) api.RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		var body []byte
		if req.Body != nil {
			var err error
			body, err = io.ReadAll(req.Body)
			if err != nil {
				return fmt.Errorf("read body: %w", err)
			}
			req.Body = io.NopCloser(bytes.NewReader(body))
		}

		timestamp := time.Now().UTC().Format(time.RFC3339)

		sig, err := signing.Sign(req.Method, req.URL.Path, body, timestamp, secretKey)
		if err != nil {
			return fmt.Errorf("sign: %w", err)
		}

		req.Header.Set("Authorization", fmt.Sprintf("HMAC-SHA256 Credential=%s, Signature=%s", authID, sig))
		req.Header.Set("X-Date", timestamp)
		return nil
	}
}
