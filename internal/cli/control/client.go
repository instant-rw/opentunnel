package control

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/opentunnel/opentunnel/internal/gen/api"
)

const DefaultAPIURL = "https://opts.ink/api/v1"

var ErrAuthorizationPending = errors.New("device authorization pending")

type Client struct {
	api   *api.ClientWithResponses
	token string
}

func New(baseURL, token string) (*Client, error) {
	generated, err := api.NewClientWithResponses(strings.TrimRight(baseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("create API client: %w", err)
	}
	return &Client{api: generated, token: token}, nil
}

func (c *Client) SetToken(token string) {
	c.token = token
}

func (c *Client) CreateDeviceAuthorization(ctx context.Context) (api.DeviceAuthorization, error) {
	response, err := c.api.CreateDeviceAuthorizationWithResponse(ctx)
	if err != nil {
		return api.DeviceAuthorization{}, fmt.Errorf("start device authorization: %w", err)
	}
	if response.StatusCode() != http.StatusCreated || response.JSON201 == nil {
		return api.DeviceAuthorization{}, responseError(response.StatusCode(), response.ApplicationproblemJSONDefault)
	}
	return *response.JSON201, nil
}

func (c *Client) ExchangeDeviceCode(ctx context.Context, code string) (string, error) {
	response, err := c.api.ExchangeDeviceCodeWithResponse(ctx, api.DeviceTokenRequest{DeviceCode: code})
	if err != nil {
		return "", fmt.Errorf("exchange device code: %w", err)
	}
	if response.StatusCode() == http.StatusAccepted {
		return "", ErrAuthorizationPending
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return "", responseError(response.StatusCode(), response.ApplicationproblemJSONDefault)
	}
	return response.JSON200.AccessToken, nil
}

func (c *Client) Logout(ctx context.Context) error {
	response, err := c.api.LogoutWithResponse(ctx, c.authorize)
	if err != nil {
		return fmt.Errorf("revoke CLI token: %w", err)
	}
	if response.StatusCode() != http.StatusNoContent {
		return responseError(response.StatusCode(), response.ApplicationproblemJSONDefault)
	}
	return nil
}

func (c *Client) ListDomains(ctx context.Context) ([]api.Domain, error) {
	response, err := c.api.ListDomainsWithResponse(ctx, c.authorize)
	if err != nil {
		return nil, fmt.Errorf("list domains: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.ApplicationproblemJSONDefault)
	}
	return *response.JSON200, nil
}

func (c *Client) CreateDomain(ctx context.Context, slug string) (api.Domain, error) {
	response, err := c.api.CreateDomainWithResponse(ctx, api.CreateDomainRequest{Slug: slug}, c.authorize)
	if err != nil {
		return api.Domain{}, fmt.Errorf("create domain: %w", err)
	}
	if response.StatusCode() != http.StatusCreated || response.JSON201 == nil {
		return api.Domain{}, responseError(response.StatusCode(), response.ApplicationproblemJSONDefault)
	}
	return *response.JSON201, nil
}

func (c *Client) ListRequests(ctx context.Context, domainID openapi_types.UUID, limit int) ([]api.CapturedRequest, error) {
	params := &api.ListRequestsParams{Limit: &limit}
	response, err := c.api.ListRequestsWithResponse(ctx, domainID, params, c.authorize)
	if err != nil {
		return nil, fmt.Errorf("list requests: %w", err)
	}
	if response.StatusCode() != http.StatusOK || response.JSON200 == nil {
		return nil, responseError(response.StatusCode(), response.ApplicationproblemJSONDefault)
	}
	return response.JSON200.Items, nil
}

func (c *Client) authorize(_ context.Context, request *http.Request) error {
	if c.token == "" {
		return errors.New("authentication required; run `opentunnel login`")
	}
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("User-Agent", "opentunnel-cli")
	return nil
}

func responseError(status int, problem *api.Problem) error {
	if problem != nil {
		if problem.Detail != nil && *problem.Detail != "" {
			return fmt.Errorf("%s (%d): %s", problem.Title, status, *problem.Detail)
		}
		return fmt.Errorf("%s (%d)", problem.Title, status)
	}
	return fmt.Errorf("OpenTunnel API returned HTTP %d", status)
}

func PollInterval(authorization api.DeviceAuthorization) time.Duration {
	seconds := authorization.Interval
	if seconds < 1 {
		seconds = 5
	}
	return time.Duration(seconds) * time.Second
}
