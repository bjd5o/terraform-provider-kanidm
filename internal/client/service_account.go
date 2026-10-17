package client

import (
	"context"
	"fmt"
	"net/url"
)

const (
	serviceAccountAttrValidFrom = "account_valid_from"
	serviceAccountAttrExpireAt  = "account_expire"
)

// ServiceAccount represents a Kanidm service account
type ServiceAccount struct {
	UUID           string
	Name           string
	SPN            string
	DisplayName    string
	EntryManagedBy []string // Account/group IDs that can manage this entry
	Mail           []string
	ValidFrom      string
	ExpireAt       string
}

// CreateServiceAccount creates a new service account. It does not mint an API token.
func (c *Client) CreateServiceAccount(ctx context.Context, name, displayName string, entryManagedBy []string) (*ServiceAccount, error) {
	attrs := map[string]any{
		"name": []string{name},
	}

	if displayName != "" {
		attrs["displayname"] = []string{displayName}
	}

	if len(entryManagedBy) > 0 {
		attrs["entry_managed_by"] = entryManagedBy
	}

	req := NewCreateRequest(attrs)

	resp, err := c.doRequest(ctx, "POST", "/v1/service_account", req)
	if err != nil {
		return nil, fmt.Errorf("create service account: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return &ServiceAccount{
		Name:           name,
		DisplayName:    displayName,
		EntryManagedBy: entryManagedBy,
	}, nil
}

// GetServiceAccount retrieves a service account by ID
func (c *Client) GetServiceAccount(ctx context.Context, id string) (*ServiceAccount, error) {
	resp, err := c.doRequest(ctx, "GET", "/v1/service_account/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, fmt.Errorf("get service account: %w", err)
	}

	var entry Entry
	if err := decodeResponse(resp, &entry); err != nil {
		return nil, err
	}

	return &ServiceAccount{
		UUID:           firstNonEmpty(entry.GetString("entryuuid"), entry.GetString("uuid")),
		Name:           entry.GetString("name"),
		SPN:            entry.GetString("spn"),
		DisplayName:    entry.GetString("displayname"),
		EntryManagedBy: entry.GetStringSlice("entry_managed_by"),
		Mail:           entry.GetStringSlice("mail"),
		ValidFrom:      entry.GetString(serviceAccountAttrValidFrom),
		ExpireAt:       entry.GetString(serviceAccountAttrExpireAt),
	}, nil
}

// UpdateServiceAccount updates a service account
func (c *Client) UpdateServiceAccount(ctx context.Context, id string, name *string, displayName *string, entryManagedBy []string, mail []string) error {
	attrs := make(map[string]any)

	if name != nil {
		attrs["name"] = []string{*name}
	}

	if displayName != nil {
		attrs["displayname"] = []string{*displayName}
	}

	if entryManagedBy != nil {
		attrs["entry_managed_by"] = entryManagedBy
	}

	if mail != nil {
		attrs["mail"] = mail
	}

	req := NewUpdateRequest(attrs)

	resp, err := c.doRequest(ctx, "PATCH", "/v1/service_account/"+id, req)
	if err != nil {
		return fmt.Errorf("update service account: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// DeleteServiceAccount deletes a service account
func (c *Client) DeleteServiceAccount(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", "/v1/service_account/"+id, nil)
	if err != nil {
		return fmt.Errorf("delete service account: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

// APIToken is an API token issued for a service account. The secret itself is not returned by list.
type APIToken struct {
	TokenID string `json:"token_id"`
	Label   string `json:"label"`
}

// ListServiceAccountAPITokens lists API tokens issued for a service account.
func (c *Client) ListServiceAccountAPITokens(ctx context.Context, id string) ([]APIToken, error) {
	resp, err := c.doRequest(ctx, "GET", "/v1/service_account/"+url.PathEscape(id)+"/_api_token", nil)
	if err != nil {
		return nil, fmt.Errorf("list service account api tokens: %w", err)
	}

	var tokens []APIToken
	if err := decodeResponse(resp, &tokens); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (c *Client) SetServiceAccountValidFrom(ctx context.Context, id, validFrom string) error {
	resp, err := c.doRequest(ctx, "PUT", "/v1/service_account/"+id+"/_attr/"+serviceAccountAttrValidFrom, []string{validFrom})
	if err != nil {
		return fmt.Errorf("set service account valid_from: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

func (c *Client) ClearServiceAccountValidFrom(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", "/v1/service_account/"+id+"/_attr/"+serviceAccountAttrValidFrom, nil)
	if err != nil {
		return fmt.Errorf("clear service account valid_from: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

func (c *Client) SetServiceAccountExpireAt(ctx context.Context, id, expireAt string) error {
	resp, err := c.doRequest(ctx, "PUT", "/v1/service_account/"+id+"/_attr/"+serviceAccountAttrExpireAt, []string{expireAt})
	if err != nil {
		return fmt.Errorf("set service account expire_at: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

func (c *Client) ClearServiceAccountExpireAt(ctx context.Context, id string) error {
	resp, err := c.doRequest(ctx, "DELETE", "/v1/service_account/"+id+"/_attr/"+serviceAccountAttrExpireAt, nil)
	if err != nil {
		return fmt.Errorf("clear service account expire_at: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	return nil
}

func (c *Client) SetServiceAccountUnix(ctx context.Context, id string, gidNumber *int64, shell *string) error {
	body := map[string]any{}
	if gidNumber != nil {
		body["gidnumber"] = *gidNumber
	}
	if shell != nil {
		body["shell"] = *shell
	}
	resp, err := c.doRequest(ctx, "POST", "/v1/service_account/"+id+"/_unix", body)
	if err != nil {
		return fmt.Errorf("set service account unix: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	return nil
}
