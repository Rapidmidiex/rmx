package user

import (
	"context"

	"github.com/rog-golang-buddies/rmx/internal"
	"github.com/rog-golang-buddies/rmx/internal/dto"
)

var MapRepo = &repo{}

type repo struct{}

func (r repo) Insert(ctx context.Context, u *internal.User) error {
	return dto.ErrNotImplemented
}
