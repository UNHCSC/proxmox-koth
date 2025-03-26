package lib

import "github.com/z46-dev/go-logger"

var Log *logger.Logger = logger.NewLogger().SetPrefix("[KOTH]", logger.BoldRed).IncludeTimestamp()
