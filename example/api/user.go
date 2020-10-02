package api

import (
	"context"
	"github.com/zyra/autonats/example"
)

// @nats:server User
type User interface {
	GetById(ctx context.Context, id []byte) (*example.User, error)
	Create(ctx context.Context, user *example.User) error
}
