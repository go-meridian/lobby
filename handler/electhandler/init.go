package electhandler

import (
	"context"

	"github.com/go-meridian/elect"
	"github.com/go-meridian/lobby/config"
	"github.com/go-meridian/logger"
)

var log *logger.Logger

func Init(cfg *config.Config, l *logger.Logger) {
	log = l

	if err := elect.Init(cfg.Elect, log,
		elect.WithOnLeader(func() {
			log.InfoCtx(context.Background(), "this instance is now the leader, starting leader-only tasks")
			// 在这里启动仅 Leader 执行的定时任务
			onLeader()
		}),
		elect.WithOnDemote(func() {
			log.InfoCtx(context.Background(), "this instance lost leadership, stopping leader-only tasks")
			// 在这里停止定时任务
			onDemote()
		}),
	); err != nil {
		log.FatalCtx(context.Background(), "elect.Init error", logger.String("error", err.Error()))
	}
}

func onLeader() {

}

func onDemote() {

}
