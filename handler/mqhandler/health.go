package mqhandler

import (
	"github.com/go-meridian/lobby/model/codeerror"
	"github.com/go-meridian/lobby/model/proto/common"
	"github.com/go-meridian/logger"
	"google.golang.org/protobuf/proto"
)

// heartbeatHandler 心跳处理
func heartbeatHandler(connId uint64, requestId uint64, payload []byte) ([]byte, *codeerror.CodeError) {
	log.Info("heartbeatService",
		logger.Uint64("connId", connId),
		logger.Uint64("requestId", requestId),
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
