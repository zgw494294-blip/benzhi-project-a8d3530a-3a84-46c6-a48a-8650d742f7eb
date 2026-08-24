package transport

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const DefaultAddress = "127.0.0.1:19081"

func ResolveAddress(flagAddress string, getenv func(string) string) (string, error) {
	address := strings.TrimSpace(flagAddress)
	if address == "" {
		if portText := strings.TrimSpace(getenv("PORT")); portText != "" {
			port, err := strconv.Atoi(portText)
			if err != nil || port < 1 || port > 65535 {
				return "", fmt.Errorf("PORT 必须是 1 到 65535 的端口号")
			}
			address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
		} else {
			address = DefaultAddress
		}
	}
	host, portText, err := net.SplitHostPort(address)
	if err != nil {
		return "", fmt.Errorf("监听地址必须采用 host:port 格式: %w", err)
	}
	if strings.TrimSpace(host) == "" {
		return "", fmt.Errorf("监听地址必须包含主机")
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 0 || port > 65535 {
		return "", fmt.Errorf("监听端口无效")
	}
	return net.JoinHostPort(host, strconv.Itoa(port)), nil
}
