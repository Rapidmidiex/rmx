package http

import (
	"context"
	"database/sql"
	"errors"

	"github.com/brianvoe/gofakeit/v6"
	"github.com/rapidmidiex/oauth"
	"github.com/rapidmidiex/rmx/internal/auth"
	"github.com/rapidmidiex/rmx/internal/auth/store/sqlc"
)

func (s *Service) saveUser(ctx context.Context, user *oauth.User) error {
	conn, err := s.repo.GetConnection(ctx, user.UserID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// user does not exist, create a new one
			if err := s.createUser(ctx, user); err != nil {
				return err
			}
		} else {
			return err
		}
	}

	// user has already logged in with this provider
	// update user data with latest info
	return s.updateUser(ctx, user, conn)
}

func (s *Service) createUser(ctx context.Context, user *oauth.User) error {
	created, err := s.repo.CreateUser(ctx, &sqlc.CreateUserParams{
		Username: gofakeit.Username(),
		Email:    user.Email,
		IsAdmin:  false,
		Picture:  user.AvatarURL,
		Blocked:  false,
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

func (s *Service) updateUser(ctx context.Context, user *oauth.User, conn *auth.Connection) error {
	userInfo, err := s.repo.GetUserByID(ctx, conn.UserID)
	if err != nil {
		return err
	}

	// some values should not be updated
	if _, err := s.repo.UpdateUserByID(ctx, &sqlc.UpdateUserByIDParams{
		ID:       conn.UserID,
		Username: userInfo.Username,
		Email:    user.Email,
		IsAdmin:  userInfo.IsAdmin,
		Picture:  user.AvatarURL,
		Blocked:  userInfo.Blocked,
	}); err != nil {
		return err
	}

	return nil
}
