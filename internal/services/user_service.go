package services

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

type UserServiceClient struct {
	serviceClient
}

func NewUserServiceClient(httpClient *http.Client, cfg *ServiceConfig) (*UserServiceClient, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}

	return &UserServiceClient{
		serviceClient: serviceClient{
			httpClient:   httpClient,
			baseURL:      cfg.BaseURL,
			apiKey:       cfg.APIKey,
			apiKeyHeader: cfg.APIKeyHeader,
		},
	}, nil
}

func (c *UserServiceClient) GetDueSchedules(ctx context.Context) ([]ProfilePolicyScheduleResponse, error) {
	return executeServiceRequest[[]ProfilePolicyScheduleResponse](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodGet,
		path:   "/internal/users/schedules/due",
	})
}

func (c *UserServiceClient) SetProcessing(ctx context.Context, userID uuid.UUID, profileID int64) error {
	_, err := executeServiceRequest[struct{}](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodPost,
		path:   "/internal/users/schedules/process",
		body: processProfilePolicyScheduleRequest{
			ProfileID: profileID,
			UserID:    userID,
		},
	})
	return err
}

func (c *UserServiceClient) DriveSchedule(ctx context.Context, userID uuid.UUID, profileID int64, state ScheduleState, by *int64) error {
	_, err := executeServiceRequest[struct{}](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodPost,
		path:   "/internal/users/schedules/drive",
		body: driveProfilePolicyScheduleRequest{
			processProfilePolicyScheduleRequest: processProfilePolicyScheduleRequest{
				ProfileID: profileID,
				UserID:    userID,
			},
			TimeoutMinutes: by,
		},
		query: url.Values{
			"state": {string(state)},
		},
	})
	return err
}

func (c *UserServiceClient) GetAttachedPolicies(ctx context.Context, userID uuid.UUID, profileID int64) ([]AttachedPolicy, error) {
	return executeServiceRequest[[]AttachedPolicy](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodGet,
		path:   fmt.Sprintf("/internal/users/user/attach/%d", profileID),
		headers: map[string]string{
			serviceUserIDHeader: userID.String(),
		},
	})
}

func (c *UserServiceClient) GetProfiles(ctx context.Context, userID uuid.UUID) ([]RegionProfile, error) {
	sellers, err := executeServiceRequest[[]Seller](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodGet,
		path:   "/internal/users/user/profiles",
		headers: map[string]string{
			serviceUserIDHeader: userID.String(),
		},
	})
	if err != nil {
		return nil, err
	}

	// Count number of profiles to accumulate
	numProfiles := 0
	for _, seller := range sellers {
		numProfiles += len(seller.Profiles)
	}

	// Flatten grouped profiles
	flatProfiles := make([]RegionProfile, 0, numProfiles)
	for _, seller := range sellers {
		for _, profile := range seller.Profiles {
			flatProfiles = append(flatProfiles, profile)
		}
	}
	return flatProfiles, nil
}

func (c *UserServiceClient) GetTokens(ctx context.Context, userID uuid.UUID) (*UserTokensResponse, error) {
	tokens, err := executeServiceRequest[UserTokensResponse](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodGet,
		path:   "/internal/users/user/tokens",
		headers: map[string]string{
			serviceUserIDHeader: userID.String(),
		},
	})
	if err != nil {
		return nil, err
	}
	return &tokens, nil
}

func (c *UserServiceClient) CreateBid(ctx context.Context, userID uuid.UUID, req CreateBidRequest) (*BidResponse, error) {
	bid, err := executeServiceRequest[BidResponse](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodPost,
		path:   "/internal/users/user/bids",
		body:   req,
		headers: map[string]string{
			serviceUserIDHeader: userID.String(),
		},
	})
	if err != nil {
		return nil, err
	}

	return &bid, nil
}

func (c *UserServiceClient) Log(ctx context.Context, userID uuid.UUID, profileID int64, log string) (*CreatedUserLogResponse, error) {
	logObj, err := executeServiceRequest[CreatedUserLogResponse](ctx, &c.serviceClient, serviceRequest{
		method: http.MethodPost,
		path:   fmt.Sprintf("/internal/users/user/logs/%d", profileID),
		headers: map[string]string{
			serviceUserIDHeader: userID.String(),
		},
		body: createUserLogRequest{
			Log: log,
		},
	})

	return &logObj, err
}

func (c *UserServiceClient) CloseIdleConnections() {
	c.httpClient.CloseIdleConnections()
}
