package api

import (
	"context"
	"encoding/json"
	"github.com/nats-io/nats.go"
	"github.com/zyra/autonats"
	"github.com/zyra/autonats/example"
	"time"
)

const timeout = time.Second * 5

type ImageServer interface {
	GetByUserId(ctx context.Context, userId string) ([]*example.Image, error)
	GetCountByUserId(ctx context.Context, userId string) (int, error)
}

type imageHandler struct {
	Server  ImageServer
	nc      *nats.Conn
	runners []*autonats.Runner
}

func (h *imageHandler) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, 2, 2)
	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.Image.GetByUserId", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		var data string
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
			return h.Server.GetByUserId(innerCtx, data)
		}
	}); err != nil {
		return err
	} else {
		h.runners[0] = runner
	}

	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.Image.GetCountByUserId", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		var data string
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
			return h.Server.GetCountByUserId(innerCtx, data)
		}
	}); err != nil {
		return err
	} else {
		h.runners[1] = runner
	}

	return nil
}

func (h *imageHandler) Shutdown() {
	for i := range h.runners {
		h.runners[i].Shutdown()
	}
}

func NewImageHandler(server ImageServer, nc *nats.Conn) autonats.Handler {
	return &imageHandler{
		Server: server,
		nc:     nc,
	}
}

type ImageClient struct {
	nc  *nats.EncodedConn
	log autonats.Logger
}

func (client *ImageClient) GetByUserId(ctx context.Context, userId string) ([]*example.Image, error) {
	var dest []*example.Image

	if err := autonats.SendRequest(ctx, client.nc, "autonats.Image.GetByUserId", userId, &dest); err != nil {
		return nil, err
	} else {
		return dest, nil
	}
}

func (client *ImageClient) GetCountByUserId(ctx context.Context, userId string) (int, error) {
	var dest int

	if err := autonats.SendRequest(ctx, client.nc, "autonats.Image.GetCountByUserId", userId, &dest); err != nil {
		return 0, err
	} else {
		return dest, nil
	}
}

type UserServer interface {
	GetById(ctx context.Context, id []byte) (*example.User, error)
	Create(ctx context.Context, user *example.User) error
}

type userHandler struct {
	Server  UserServer
	nc      *nats.Conn
	runners []*autonats.Runner
}

func (h *userHandler) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, 2, 2)
	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.User.GetById", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		var data []byte
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
			return h.Server.GetById(innerCtx, data)
		}
	}); err != nil {
		return err
	} else {
		h.runners[0] = runner
	}

	if runner, err := autonats.StartRunner(ctx, h.nc, "autonats.User.Create", "autonats", 5, func(msg *nats.Msg) (interface{}, error) {
		var data example.User
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return nil, err
		} else {
			innerCtx, _ := context.WithTimeout(ctx, time.Second*5)
			return nil, h.Server.Create(innerCtx, &data)
		}
	}); err != nil {
		return err
	} else {
		h.runners[1] = runner
	}

	return nil
}

func (h *userHandler) Shutdown() {
	for i := range h.runners {
		h.runners[i].Shutdown()
	}
}

func NewUserHandler(server UserServer, nc *nats.Conn) autonats.Handler {
	return &userHandler{
		Server: server,
		nc:     nc,
	}
}

type UserClient struct {
	nc  *nats.EncodedConn
	log autonats.Logger
}

func (client *UserClient) GetById(ctx context.Context, id []byte) (*example.User, error) {
	var dest example.User

	if err := autonats.SendRequest(ctx, client.nc, "autonats.User.GetById", id, &dest); err != nil {
		return nil, err
	} else {
		return &dest, nil
	}
}

func (client *UserClient) Create(ctx context.Context, user *example.User) error {
	return autonats.SendRequest(ctx, client.nc, "autonats.User.Create", user, nil)
}
