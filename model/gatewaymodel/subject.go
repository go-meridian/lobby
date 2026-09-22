package gatewaymodel

import "fmt"

// Subject 生成 NATS subject: gatewaymodel.request.{cmd}
func Subject(cmd string) string {
	return fmt.Sprintf("gatewaymodel.request.%s", cmd)
}
