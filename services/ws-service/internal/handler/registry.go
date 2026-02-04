package handler

import (
	"context"
	"fmt"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
)

type Registry struct {
	handlers map[protocol.MessageType]Handler
	log      logger.Logger
}

func NewRegistry(log logger.Logger) *Registry {
	return &Registry{
		handlers: make(map[protocol.MessageType]Handler),
		log:      log,
	}
}

func (r *Registry) Register(msgType protocol.MessageType, h Handler) {
	r.handlers[msgType] = h
	r.log.Debug("handler registered",
		logger.String("message_type", string(msgType)),
	)
}

func (r *Registry) Get(msgType protocol.MessageType) (Handler, bool) {
	h, ok := r.handlers[msgType]
	return h, ok
}

func (r *Registry) Handle(ctx context.Context, hctx *Context, msgType protocol.MessageType) error {
	h, ok := r.handlers[msgType]
	if !ok {
		return fmt.Errorf("no handler registered for message type: %s", msgType)
	}
	return h.Handle(ctx, hctx)
}

func (r *Registry) RegisterDefaults(deps *Dependencies, log logger.Logger) {
	r.Register(protocol.MsgTypeSubscribe, NewSubscriptionHandler(deps, log))
	r.Register(protocol.MsgTypeUnsubscribe, NewUnsubscribeHandler(deps, log))
	r.Register(protocol.MsgTypePresenceUpdate, NewPresenceUpdateHandler(deps, log))
	r.Register(protocol.MsgTypePresenceQuery, NewPresenceQueryHandler(deps, log))
	r.Register(protocol.MsgTypeTypingStart, NewTypingStartHandler(deps, log))
	r.Register(protocol.MsgTypeTypingStop, NewTypingStopHandler(deps, log))
	r.Register(protocol.MsgTypeMarkRead, NewMarkReadHandler(deps, log))
	r.Register(protocol.MsgTypeMarkDelivered, NewMarkDeliveredHandler(deps, log))
	r.Register(protocol.MsgTypePing, NewPingHandler(deps, log))
}
