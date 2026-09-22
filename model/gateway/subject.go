package gatewaymodel

import "fmt"

// Subject 生成 NATS subject: gateway.request.{cmd}
func Subject(cmd string) string {
	return fmt.Sprintf("gateway.request.%s", cmd)
}
