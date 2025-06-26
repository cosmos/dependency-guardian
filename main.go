package main

import (
	"github.com/cosmos/dependency-guardian/cmd"
	"go.uber.org/zap"
)

func main() {
	if err := cmd.Execute(); err != nil {
		zap.S().Fatalw("command failed", "error", err)
	}
}