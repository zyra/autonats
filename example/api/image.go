package api

import (
	"context"
	"github.com/zyra/autonats/example"
)

// @nats:server Image
type Image interface {
	GetByUserId(ctx context.Context, userId string) ([]*example.Image, error)
	GetCountByUserId(ctx context.Context, userId string) (int, error)
}
