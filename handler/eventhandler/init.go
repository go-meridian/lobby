package eventhandler

import (
	"github.com/go-meridian/event"
	"github.com/go-meridian/lobby/model/eventmodel"
	"github.com/go-meridian/logger"
)

var log *logger.Logger

func Init(l *logger.Logger) {
	log = l

}

func Register() {
	event.SubscribeAsync((*eventmodel.HealthEvent)(nil), onHealth)
}
