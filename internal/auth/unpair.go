package auth

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DeletePairing calls DELETE /auth/pairing on the signaling server to remove
// the pairing record and notify the mobile device.
// Uses ExchangeToken() to get a JWT first, then sends the DELETE request.
func DeletePairing(identity *Identity, serverURL string, client *http.Client) error {
	token, err := ExchangeToken(identity, serverURL, client)
	if err != nil {
		return fmt.Errorf("token exchange: %w", err)
	}

	url := strings.TrimRight(serverURL, "/") + "/auth/pairing"
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		return fmt.Errorf("create delete request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return errors.New(connError(err))
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		return errors.New(serverError(resp.StatusCode, body))
	}

	return nil
}
