package services

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type PolicyServiceClient struct {
	serviceClient
}

func NewPolicyServiceClient(httpClient *http.Client, cfg *ServiceConfig) (*PolicyServiceClient, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &PolicyServiceClient{
		serviceClient: serviceClient{
			httpClient:   httpClient,
			baseURL:      cfg.BaseURL,
			apiKey:       cfg.APIKey,
			apiKeyHeader: cfg.APIKeyHeader,
		},
	}, nil
}

func (c *PolicyServiceClient) GetPoliciesForMarketplace(ctx context.Context, userID uuid.UUID, marketplace string) ([]Policy, error) {
	return executeServiceRequest[[]Policy](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodGet,
		path:   "/internal/policies/policies", // I realise the unfortunate path
		query: url.Values{
			"marketplace": []string{marketplace},
		},
		headers: map[string]string{
			c.apiKeyHeader:      c.apiKey,
			serviceUserIDHeader: userID.String(),
		},
	})
}

func (c *PolicyServiceClient) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}
