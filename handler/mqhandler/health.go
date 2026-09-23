package mqhandler

import (
	"context"

	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/lobby/model/proto/common"
	"github.com/go-meridian/logger"
	"google.golang.org/protobuf/proto"
)

// heartbeatHandler 心跳处理
func heartbeatHandler(ctx context.Context, connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError) {
	log.InfoCtx(ctx, "heartbeatService",
		logger.Uint64("connId", connId),
	)

	rsp := &common.ErrMsg{
		RetCode: 0,
		ErrMsg:  "pong",
	}
	data, err := proto.Marshal(rsp)
	if err != nil {
		return nil, codeerror.SystemError.Msg("marshal error")
	}
	return data, nil
}
