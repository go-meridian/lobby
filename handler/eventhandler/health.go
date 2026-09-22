package eventhandler

import (
	"github.com/go-meridian/lobby/model/eventmodel"
	"github.com/go-meridian/logger"
)

func onHealth(evt *eventmodel.HealthEvent) {
	logger.L().Info("health event received",
		logger.String("requestID", evt.RequestID),
		logger.String("status", evt.Status),
	)
}
