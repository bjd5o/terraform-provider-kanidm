package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type authResponse struct {
	SessionID string                     `json:"sessionid"`
	State     map[string]json.RawMessage `json:"state"`
}

// AuthenticatePassword logs in as username with password. A nil error means Kanidm accepted the password.
func (c *Client) AuthenticatePassword(ctx context.Context, username, password string) error {
	resp, sessionToken, err := c.authStep(ctx, "", map[string]any{"init": username})
	if err != nil {
		return err
	}
	if msg, ok := deniedMessage(resp); ok {
		return fmt.Errorf("authentication denied for %s: %s", username, msg)
	}
	if sessionToken == "" {
		return fmt.Errorf("authentication for %s did not return a session", username)
	}

	if _, ok := resp.State["choose"]; ok {
		resp, sessionToken, err = c.authStep(ctx, sessionToken, map[string]any{"begin": "password"})
		if err != nil {
			return err
		}
		if msg, ok := deniedMessage(resp); ok {
			return fmt.Errorf("authentication denied for %s: %s", username, msg)
		}
	}

	resp, _, err = c.authStep(ctx, sessionToken, map[string]any{"cred": map[string]string{"password": password}})
	if err != nil {
		return err
	}
	if _, ok := resp.State["success"]; ok {
		return nil
	}
	if msg, ok := deniedMessage(resp); ok {
		return fmt.Errorf("authentication denied for %s: %s", username, msg)
	}
	return fmt.Errorf("authentication for %s did not succeed", username)
}

func deniedMessage(resp *authResponse) (string, bool) {
	raw, ok := resp.State["denied"]
	if !ok {
		return "", false
	}
	var msg string
	if err := json.Unmarshal(raw, &msg); err != nil || msg == "" {
		return "denied", true
	}
	return msg, true
}

func (c *Client) authStep(ctx context.Context, sessionToken string, step any) (*authResponse, string, error) {
	body, err := json.Marshal(map[string]any{"step": step})
	if err != nil {
		return nil, "", fmt.Errorf("marshal auth step: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/v1/auth", bytes.NewReader(body))
	if err != nil {
		return nil, "", fmt.Errorf("create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	// The value is a signed token. The UUID in the JSON body is not accepted.
	if sessionToken != "" {
		req.Header.Set("X-KANIDM-AUTH-SESSION-ID", sessionToken)
		req.AddCookie(&http.Cookie{Name: "auth-session-id", Value: sessionToken})
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("execute auth request: %w", err)
	}
	nextSession := authSessionToken(resp)
	if nextSession == "" {
		nextSession = sessionToken
	}
	if err := c.checkResponse(resp); err != nil {
		return nil, "", fmt.Errorf("auth step: %w", err)
	}

	var result authResponse
	if err := decodeResponse(resp, &result); err != nil {
		return nil, "", err
	}
	return &result, nextSession, nil
}

func authSessionToken(resp *http.Response) string {
	if token := resp.Header.Get("X-KANIDM-AUTH-SESSION-ID"); token != "" {
		return token
	}
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "auth-session-id" && cookie.Value != "" {
			return cookie.Value
		}
	}
	return ""
}
