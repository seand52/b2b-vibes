package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"b2b-orders-api/internal/domain"
	"b2b-orders-api/internal/repository"
)

var (
	// ErrClientNotFound is returned when a client is not pre-registered in the system
	ErrClientNotFound = errors.New("client not registered - contact administrator")
	// ErrEmailMismatch is returned when the email does not match the registered client
	ErrEmailMismatch = errors.New("email does not match registered client")
)

// Service handles authentication-related business logic
type Service struct {
	clientRepo repository.ClientRepository
	logger     *slog.Logger
}

// NewService creates a new auth service
func NewService(clientRepo repository.ClientRepository, logger *slog.Logger) *Service {
	return &Service{
		clientRepo: clientRepo,
		logger:     logger,
	}
}

// GetOrLinkClient retrieves a client by Auth0 ID or links an Auth0 ID to an existing client
// matched by email. This is called after successful Auth0 authentication to get/link the
// internal Client record.
//
// Flow:
// 1. Try to find client by auth0_id (already linked) - return immediately if found
// 2. If not found, try to find client by email (pre-registered in Holded)
// 3. If found by email, link the auth0_id to that client and return it
// 4. If not found by email, return ErrClientNotFound (user must be pre-registered)
func (s *Service) GetOrLinkClient(ctx context.Context, auth0ID, email string) (*domain.Client, error) {
	// First, check if client is already linked by Auth0 ID
	client, err := s.clientRepo.GetByAuth0ID(ctx, auth0ID)
	if err == nil {
		s.logger.InfoContext(ctx, "client already linked",
			"client_id", client.ID,
			"auth0_id", auth0ID,
			"email", email,
		)
		return client, nil
	}

	// If not found by Auth0 ID, try to find by email (pre-registered client)
	client, err = s.clientRepo.GetByEmail(ctx, email)
	if err != nil {
		s.logger.WarnContext(ctx, "client not found by email - user not pre-registered",
			"auth0_id", auth0ID,
			"email", email,
			"error", err,
		)
		return nil, fmt.Errorf("%w: %s", ErrClientNotFound, email)
	}

	// Found a pre-registered client - link the Auth0 ID
	if err := s.clientRepo.LinkAuth0ID(ctx, client.ID, auth0ID); err != nil {
		s.logger.ErrorContext(ctx, "failed to link auth0_id to client",
			"client_id", client.ID,
			"auth0_id", auth0ID,
			"email", email,
			"error", err,
		)
		return nil, fmt.Errorf("failed to link auth0_id: %w", err)
	}

	s.logger.InfoContext(ctx, "successfully linked auth0_id to existing client",
		"client_id", client.ID,
		"auth0_id", auth0ID,
		"email", email,
	)

	// Update the client's Auth0ID field and return it
	client.Auth0ID = &auth0ID
	return client, nil
}

// GetClientByAuth0ID retrieves a client by their Auth0 ID
// Returns the client if found, or an error if not found
func (s *Service) GetClientByAuth0ID(ctx context.Context, auth0ID string) (*domain.Client, error) {
	client, err := s.clientRepo.GetByAuth0ID(ctx, auth0ID)
	if err != nil {
		s.logger.WarnContext(ctx, "client not found by auth0_id",
			"auth0_id", auth0ID,
			"error", err,
		)
		return nil, fmt.Errorf("%w: auth0_id=%s", ErrClientNotFound, auth0ID)
	}

	return client, nil
}
