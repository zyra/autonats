package api

import (
	"context"
	"github.com/json-iterator/go"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/not.go"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
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
	Server   ImageServer
	NatsConn *nats.Conn
	runners  []*autonats.Runner
}

func (h *imageHandler) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, 2, 2)
	tracer := opentracing.GlobalTracer()
	if runner, err := autonats.StartRunner(ctx, h.NatsConn, "autonats.Image.GetByUserId", "autonats", 5, func(msg *nats.Msg) {
		t := not.NewTraceMsg(msg)
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			return
		}

		replySpan := tracer.StartSpan("Autonats ImageServer", ext.SpanKindRPCServer, ext.RPCServerOption(sc))
		ext.MessageBusDestination.Set(replySpan, msg.Subject)
		defer replySpan.Finish()
		innerCtx, _ := context.WithTimeout(ctx, timeout)
		innerCtxT := opentracing.ContextWithSpan(innerCtx, replySpan)

		var result []*example.Image

		result, err = h.Server.GetByUserId(innerCtxT, string(msg.Data))

		reply := autonats.GetReply()
		defer autonats.PutReply(reply)

		if err != nil {
			replySpan.LogFields(log.Event("Handler returned error"))
			reply.Error = []byte(err.Error())

		} else if result != nil {
			replySpan.LogFields(log.Event("Handler returned a result"))
			if err := reply.MarshalAndSetData(result); err != nil {
				replySpan.LogFields(log.Error(err))
				return
			}

		}

		replyData, err := reply.MarshalBinary()

		if err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}

		replySpan.LogFields(log.Event("Sending reply"))
		if err := msg.Respond(replyData); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
	}); err != nil {
		return err
	} else {
		h.runners[0] = runner
	}

	if runner, err := autonats.StartRunner(ctx, h.NatsConn, "autonats.Image.GetCountByUserId", "autonats", 5, func(msg *nats.Msg) {
		t := not.NewTraceMsg(msg)
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			return
		}

		replySpan := tracer.StartSpan("Autonats ImageServer", ext.SpanKindRPCServer, ext.RPCServerOption(sc))
		ext.MessageBusDestination.Set(replySpan, msg.Subject)
		defer replySpan.Finish()
		innerCtx, _ := context.WithTimeout(ctx, timeout)
		innerCtxT := opentracing.ContextWithSpan(innerCtx, replySpan)

		var result int

		result, err = h.Server.GetCountByUserId(innerCtxT, string(msg.Data))

		reply := autonats.GetReply()
		defer autonats.PutReply(reply)

		if err != nil {
			replySpan.LogFields(log.Event("Handler returned error"))
			reply.Error = []byte(err.Error())

		} else if result != 0 {
			replySpan.LogFields(log.Event("Handler returned a result"))
			if err := reply.MarshalAndSetData(result); err != nil {
				replySpan.LogFields(log.Error(err))
				return
			}

		}

		replyData, err := reply.MarshalBinary()

		if err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}

		replySpan.LogFields(log.Event("Sending reply"))
		if err := msg.Respond(replyData); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
	}); err != nil {
		h.Shutdown()
		return err
	} else {
		h.runners[1] = runner
	}

	return nil
}

func (h *imageHandler) Shutdown() {
	for i := range h.runners {
		if h.runners[i] != nil {
			_ = h.runners[i].Shutdown()
		}
	}
}

func NewImageHandler(server ImageServer, nc *nats.Conn) autonats.Handler {
	return &imageHandler{
		Server:   server,
		NatsConn: nc,
	}
}

type ImageClient struct{ NatsConn *nats.Conn }

func NewImageClient(nc *nats.Conn) *ImageClient {
	return &ImageClient{NatsConn: nc}
}

func (client *ImageClient) GetByUserId(ctx context.Context, userId string) ([]*example.Image, error) {

	reqSpan, reqCtx := opentracing.StartSpanFromContext(ctx, "Autonats ImageClient Request", ext.SpanKindRPCClient)
	ext.MessageBusDestination.Set(reqSpan, "autonats.Image.GetByUserId")
	defer reqSpan.Finish()
	reqSpan.LogFields(log.Event("Starting request"))

	var t not.TraceMsg
	var err error

	if err = opentracing.GlobalTracer().Inject(reqSpan.Context(), opentracing.Binary, &t); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	if _, err = t.Write([]byte(userId)); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	reqCtx, cancelFn := context.WithTimeout(reqCtx, timeout)
	defer cancelFn()

	var replyMsg *nats.Msg
	if replyMsg, err = client.NatsConn.RequestWithContext(ctx, "autonats.Image.GetByUserId", t.Bytes()); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	reqSpan.LogFields(log.Event("Reply received"))
	reply := autonats.GetReply()
	defer autonats.PutReply(reply)

	if err := reply.UnmarshalBinary(replyMsg.Data); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	if err := reply.GetError(); err != nil {
		return nil, err
	}

	var result []*example.Image
	if err := reply.UnmarshalData(&result); err != nil {
		return nil, err
	}

	return result, nil

}

func (client *ImageClient) GetCountByUserId(ctx context.Context, userId string) (int, error) {

	reqSpan, reqCtx := opentracing.StartSpanFromContext(ctx, "Autonats ImageClient Request", ext.SpanKindRPCClient)
	ext.MessageBusDestination.Set(reqSpan, "autonats.Image.GetCountByUserId")
	defer reqSpan.Finish()
	reqSpan.LogFields(log.Event("Starting request"))

	var t not.TraceMsg
	var err error

	if err = opentracing.GlobalTracer().Inject(reqSpan.Context(), opentracing.Binary, &t); err != nil {
		reqSpan.LogFields(log.Error(err))
		return 0, err
	}

	if _, err = t.Write([]byte(userId)); err != nil {
		reqSpan.LogFields(log.Error(err))
		return 0, err
	}

	reqCtx, cancelFn := context.WithTimeout(reqCtx, timeout)
	defer cancelFn()

	var replyMsg *nats.Msg
	if replyMsg, err = client.NatsConn.RequestWithContext(ctx, "autonats.Image.GetCountByUserId", t.Bytes()); err != nil {
		reqSpan.LogFields(log.Error(err))
		return 0, err
	}

	reqSpan.LogFields(log.Event("Reply received"))
	reply := autonats.GetReply()
	defer autonats.PutReply(reply)

	if err := reply.UnmarshalBinary(replyMsg.Data); err != nil {
		reqSpan.LogFields(log.Error(err))
		return 0, err
	}

	if err := reply.GetError(); err != nil {
		return 0, err
	}

	var result int
	if err := reply.UnmarshalData(&result); err != nil {
		return 0, err
	}

	return result, nil

}

type UserServer interface {
	GetById(ctx context.Context, id []byte) (*example.User, error)
	Create(ctx context.Context, user *example.User) error
}

type userHandler struct {
	Server   UserServer
	NatsConn *nats.Conn
	runners  []*autonats.Runner
}

func (h *userHandler) Run(ctx context.Context) error {
	h.runners = make([]*autonats.Runner, 2, 2)
	tracer := opentracing.GlobalTracer()
	if runner, err := autonats.StartRunner(ctx, h.NatsConn, "autonats.User.GetById", "autonats", 5, func(msg *nats.Msg) {
		t := not.NewTraceMsg(msg)
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			return
		}

		replySpan := tracer.StartSpan("Autonats UserServer", ext.SpanKindRPCServer, ext.RPCServerOption(sc))
		ext.MessageBusDestination.Set(replySpan, msg.Subject)
		defer replySpan.Finish()
		innerCtx, _ := context.WithTimeout(ctx, timeout)
		innerCtxT := opentracing.ContextWithSpan(innerCtx, replySpan)

		var result *example.User

		var data []byte
		if err = jsoniter.Unmarshal(msg.Data, &data); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
		result, err = h.Server.GetById(innerCtxT, data)

		reply := autonats.GetReply()
		defer autonats.PutReply(reply)

		if err != nil {
			replySpan.LogFields(log.Event("Handler returned error"))
			reply.Error = []byte(err.Error())

		} else if result != nil {
			replySpan.LogFields(log.Event("Handler returned a result"))
			if err := reply.MarshalAndSetData(result); err != nil {
				replySpan.LogFields(log.Error(err))
				return
			}

		}

		replyData, err := reply.MarshalBinary()

		if err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}

		replySpan.LogFields(log.Event("Sending reply"))
		if err := msg.Respond(replyData); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
	}); err != nil {
		return err
	} else {
		h.runners[0] = runner
	}

	if runner, err := autonats.StartRunner(ctx, h.NatsConn, "autonats.User.Create", "autonats", 5, func(msg *nats.Msg) {
		t := not.NewTraceMsg(msg)
		sc, err := tracer.Extract(opentracing.Binary, t)
		if err != nil {
			return
		}

		replySpan := tracer.StartSpan("Autonats UserServer", ext.SpanKindRPCServer, ext.RPCServerOption(sc))
		ext.MessageBusDestination.Set(replySpan, msg.Subject)
		defer replySpan.Finish()
		innerCtx, _ := context.WithTimeout(ctx, timeout)
		innerCtxT := opentracing.ContextWithSpan(innerCtx, replySpan)

		var data example.User
		if err = jsoniter.Unmarshal(msg.Data, &data); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
		err = h.Server.Create(innerCtxT, &data)

		reply := autonats.GetReply()
		defer autonats.PutReply(reply)

		if err != nil {
			replySpan.LogFields(log.Event("Handler returned error"))
			reply.Error = []byte(err.Error())

		}

		replyData, err := reply.MarshalBinary()

		if err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}

		replySpan.LogFields(log.Event("Sending reply"))
		if err := msg.Respond(replyData); err != nil {
			replySpan.LogFields(log.Error(err))
			return
		}
	}); err != nil {
		h.Shutdown()
		return err
	} else {
		h.runners[1] = runner
	}

	return nil
}

func (h *userHandler) Shutdown() {
	for i := range h.runners {
		if h.runners[i] != nil {
			_ = h.runners[i].Shutdown()
		}
	}
}

func NewUserHandler(server UserServer, nc *nats.Conn) autonats.Handler {
	return &userHandler{
		Server:   server,
		NatsConn: nc,
	}
}

type UserClient struct{ NatsConn *nats.Conn }

func NewUserClient(nc *nats.Conn) *UserClient {
	return &UserClient{NatsConn: nc}
}

func (client *UserClient) GetById(ctx context.Context, id []byte) (*example.User, error) {

	reqSpan, reqCtx := opentracing.StartSpanFromContext(ctx, "Autonats UserClient Request", ext.SpanKindRPCClient)
	ext.MessageBusDestination.Set(reqSpan, "autonats.User.GetById")
	defer reqSpan.Finish()
	reqSpan.LogFields(log.Event("Starting request"))

	var t not.TraceMsg
	var err error

	if err = opentracing.GlobalTracer().Inject(reqSpan.Context(), opentracing.Binary, &t); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	var data []byte
	data, err = jsoniter.Marshal(id)
	if err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	if _, err = t.Write(data); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	reqCtx, cancelFn := context.WithTimeout(reqCtx, timeout)
	defer cancelFn()

	var replyMsg *nats.Msg
	if replyMsg, err = client.NatsConn.RequestWithContext(ctx, "autonats.User.GetById", t.Bytes()); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	reqSpan.LogFields(log.Event("Reply received"))
	reply := autonats.GetReply()
	defer autonats.PutReply(reply)

	if err := reply.UnmarshalBinary(replyMsg.Data); err != nil {
		reqSpan.LogFields(log.Error(err))
		return nil, err
	}

	if err := reply.GetError(); err != nil {
		return nil, err
	}

	var result example.User
	if err := reply.UnmarshalData(&result); err != nil {
		return nil, err
	}

	return &result, nil

}

func (client *UserClient) Create(ctx context.Context, user *example.User) error {

	reqSpan, reqCtx := opentracing.StartSpanFromContext(ctx, "Autonats UserClient Request", ext.SpanKindRPCClient)
	ext.MessageBusDestination.Set(reqSpan, "autonats.User.Create")
	defer reqSpan.Finish()
	reqSpan.LogFields(log.Event("Starting request"))

	var t not.TraceMsg
	var err error

	if err = opentracing.GlobalTracer().Inject(reqSpan.Context(), opentracing.Binary, &t); err != nil {
		reqSpan.LogFields(log.Error(err))
		return err
	}

	var data []byte
	data, err = jsoniter.Marshal(user)
	if err != nil {
		reqSpan.LogFields(log.Error(err))
		return err
	}

	if _, err = t.Write(data); err != nil {
		reqSpan.LogFields(log.Error(err))
		return err
	}

	reqCtx, cancelFn := context.WithTimeout(reqCtx, timeout)
	defer cancelFn()

	var replyMsg *nats.Msg
	if replyMsg, err = client.NatsConn.RequestWithContext(ctx, "autonats.User.Create", t.Bytes()); err != nil {
		reqSpan.LogFields(log.Error(err))
		return err
	}

	reqSpan.LogFields(log.Event("Reply received"))
	reply := autonats.GetReply()
	defer autonats.PutReply(reply)

	if err := reply.UnmarshalBinary(replyMsg.Data); err != nil {
		reqSpan.LogFields(log.Error(err))
		return err
	}

	if err := reply.GetError(); err != nil {
		return err
	}

	return nil
}
