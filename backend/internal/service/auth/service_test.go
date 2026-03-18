package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
	"b2b-orders-api/internal/testutil"
)

func TestService_GetOrLinkClient(t *testing.T) {
	clientID := uuid.New()
	existingClient := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-123",
		Email:    "test@example.com",
	}

	tests := []struct {
		name               string
		auth0ID            string
		email              string
		getByAuth0IDResult *domain.Client
		getByAuth0IDErr    error
		getByEmailResult   *domain.Client
		getByEmailErr      error
		linkAuth0IDErr     error
		wantClient         *domain.Client
		wantErr            error
		wantLinked         bool
	}{
		{
			name:               "already linked client",
			auth0ID:            "auth0|123",
			email:              "test@example.com",
			getByAuth0IDResult: existingClient,
			wantClient:         existingClient,
		},
		{
			name:             "new user links to existing client",
			auth0ID:          "auth0|456",
			email:            "test@example.com",
			getByAuth0IDErr:  repository.ErrNotFound,
			getByEmailResult: existingClient,
			wantClient:       existingClient,
			wantLinked:       true,
		},
		{
			name:            "user not pre-registered",
			auth0ID:         "auth0|789",
			email:           "unknown@example.com",
			getByAuth0IDErr: repository.ErrNotFound,
			getByEmailErr:   repository.ErrNotFound,
			wantErr:         ErrClientNotFound,
		},
		{
			name:             "link fails",
			auth0ID:          "auth0|999",
			email:            "test@example.com",
			getByAuth0IDErr:  repository.ErrNotFound,
			getByEmailResult: existingClient,
			linkAuth0IDErr:   errors.New("db error"),
			wantErr:          errors.New("failed to link auth0_id"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &testutil.MockClientRepo{
				GetByAuth0IDResult: tt.getByAuth0IDResult,
				GetByAuth0IDErr:    tt.getByAuth0IDErr,
				GetByEmailResult:   tt.getByEmailResult,
				GetByEmailErr:      tt.getByEmailErr,
				LinkAuth0IDErr:     tt.linkAuth0IDErr,
			}

			svc := NewService(repo, testutil.NewDiscardLogger())
			client, err := svc.GetOrLinkClient(context.Background(), tt.auth0ID, tt.email)

			if tt.wantErr != nil {
				require.Error(t, err)
				if errors.Is(tt.wantErr, ErrClientNotFound) {
					assert.ErrorIs(t, err, ErrClientNotFound)
				}
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantClient.ID, client.ID)

			if tt.wantLinked {
				assert.Equal(t, tt.auth0ID, repo.LinkedAuth0ID)
				assert.Equal(t, clientID, repo.LinkedClientID)
			}
		})
	}
}

func TestService_GetClientByAuth0ID(t *testing.T) {
	clientID := uuid.New()
	existingClient := &domain.Client{
		ID:       clientID,
		HoldedID: "holded-123",
		Email:    "test@example.com",
	}

	tests := []struct {
		name       string
		auth0ID    string
		repoResult *domain.Client
		repoErr    error
		wantErr    bool
	}{
		{
			name:       "client found",
			auth0ID:    "auth0|123",
			repoResult: existingClient,
		},
		{
			name:    "client not found",
			auth0ID: "auth0|unknown",
			repoErr: repository.ErrNotFound,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &testutil.MockClientRepo{
				GetByAuth0IDResult: tt.repoResult,
				GetByAuth0IDErr:    tt.repoErr,
			}

			svc := NewService(repo, testutil.NewDiscardLogger())
			client, err := svc.GetClientByAuth0ID(context.Background(), tt.auth0ID)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.repoResult.ID, client.ID)
		})
	}
}
