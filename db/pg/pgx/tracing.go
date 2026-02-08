package pgx

import (
	"github.com/jackc/pgx/v5/tracelog"
)

func parseTraceLogLevel(lvl string) tracelog.LogLevel {
	logLevel, err := tracelog.LogLevelFromString(lvl)
	if err != nil {
		logLevel = tracelog.LogLevelNone
	}

	return logLevel
}
