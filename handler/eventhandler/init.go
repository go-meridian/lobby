package eventhandler

import (
	"github.com/go-meridian/event"
	"github.com/go-meridian/lobby/model/eventmodel"
)

func init() {
	event.SubscribeAsync((*eventmodel.HealthEvent)(nil), onHealth)
}
