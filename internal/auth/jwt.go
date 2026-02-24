package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// TokenResponse represents the server's response to a token exchange request.
type TokenResponse struct {
	Token string `json:"token"`
	Error string `json:"error,omitempty"`
}

// ExchangeToken signs a challenge with the identity key and exchanges it for a JWT.
// serverURL should be the base URL of the signaling server (e.g., "https://signal.pocketmux.dev").
func ExchangeToken(id *Identity, serverURL string, client *http.Client) (string, error) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	signature := id.SignChallenge(id.DeviceID, timestamp)

	body := fmt.Sprintf(`{"deviceId":%q,"timestamp":%q,"signature":%q}`,
		id.DeviceID, timestamp, signature)

	url := strings.TrimRight(serverURL, "/") + "/auth/token"
	resp, err := client.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("token exchange request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	var tokenResp TokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("token exchange failed (%d): %s", resp.StatusCode, tokenResp.Error)
	}

	if tokenResp.Token == "" {
		return "", fmt.Errorf("token exchange returned empty token")
	}

	return tokenResp.Token, nil
}
