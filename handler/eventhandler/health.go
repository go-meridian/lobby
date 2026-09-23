package eventhandler

import (
	"context"

	"github.com/go-meridian/lobby/model/eventmodel"
	"github.com/go-meridian/logger"
)

func onHealth(evt *eventmodel.HealthEvent) {
	ctx := logger.WithRequestId(context.Background(), evt.RequestID)
	log.InfoCtx(ctx, "health event received",
		logger.String("status", evt.Status),
	)
}
