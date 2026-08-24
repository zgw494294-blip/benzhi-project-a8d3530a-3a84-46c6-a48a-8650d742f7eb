package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"benzhi-project-a8d3530a-3a84-46c6-a48a-8650d742f7eb/internal/transport"
)

func main() {
	var addressFlag string
	var dataDir string
	var selfcheck bool
	flag.StringVar(&addressFlag, "addr", "", "监听地址，默认 127.0.0.1:19081")
	flag.StringVar(&dataDir, "data", "data", "本地事件账本目录")
	flag.BoolVar(&selfcheck, "selfcheck", false, "运行有界全链路自检后退出")
	flag.Parse()

	address, err := transport.ResolveAddress(addressFlag, os.Getenv)
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置错误:", err)
		os.Exit(2)
	}
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if selfcheck {
		if err := runSelfcheck(address, logger); err != nil {
			fmt.Fprintln(os.Stderr, "自检失败:", err)
			os.Exit(1)
		}
		fmt.Println("自检通过：建档、监测、整改复测、独立验收、凭据验证和账本恢复均正常")
		return
	}
	if err := runServer(address, dataDir, logger); err != nil {
		fmt.Fprintln(os.Stderr, "服务退出:", err)
		os.Exit(1)
	}
}
