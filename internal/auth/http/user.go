package http

import (
	"database/sql"
	"errors"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/rapidmidiex/oauth"
	"github.com/rapidmidiex/rmx/internal/auth/store/sqlc"
	"golang.org/x/net/context"
)

func (s *Service) saveUser(ctx context.Context, user *oauth.User) error {
	// TODO: detect if user does have an account or not
	_, err := s.repo.GetConnection(ctx, user.UserID)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return err
		}
	}

	// user does not exist, create a new one
	created, err := s.repo.CreateUser(ctx, &sqlc.CreateUserParams{
		Username:      gofakeit.Username(),
		Email:         user.Email,
		EmailVerified: false,
		IsAdmin:       false,
		Picture:       user.AvatarURL,
		Blocked:       false,
	})
	if err != nil {
		return err
	}

	if _, err := s.repo.CreateConnection(ctx, &sqlc.CreateConnectionParams{
		ProviderID: user.UserID,
		UserID:     created.ID,
	}); err != nil {
		return err
	}

	return nil
}
