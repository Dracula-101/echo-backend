package handler

import (
	"context"
	"encoding/json"
	"time"

	"shared/pkg/logger"
	"ws-service/internal/protocol"
)

type PingHandler struct {
	deps *Dependencies
	log  logger.Logger
}

func NewPingHandler(deps *Dependencies, log logger.Logger) *PingHandler {
	return &PingHandler{deps: deps, log: log}
}

func (h *PingHandler) Handle(ctx context.Context, hctx *Context) error {
	var pingPayload protocol.PingPayload
	if len(hctx.Payload) > 0 {
		if err := json.Unmarshal(hctx.Payload, &pingPayload); err != nil {
			return err
		}
	}

	serverTimestamp := time.Now().UTC()
	pongPayload := protocol.PongPayload{
		ClientTimestamp: pingPayload.ClientTimestamp,
		ServerTimestamp: serverTimestamp,
	}
	if !pingPayload.ClientTimestamp.IsZero() {
		pongPayload.Latency = serverTimestamp.Sub(pingPayload.ClientTimestamp).Milliseconds()
	}

	pong := protocol.NewServerMessage(protocol.MsgTypePong, hctx.RequestID).WithPayload(pongPayload)

	data, _ := json.Marshal(pong)
	return hctx.Conn.Send(data)
}
