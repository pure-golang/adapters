// Recovery recovers panic and logs it on ERROR level. 500 http status is returned
package middleware

import (
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/pure-golang/adapters/logger"
)

func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			err := recover()
			if err == nil {
				return
			}

			rawStack := strings.ReplaceAll(string(debug.Stack()), "\t", "")
			var stack []string
			for _, line := range strings.Split(rawStack, "\n") {
				if line != "" {
					stack = append(stack, line)
				}
			}

			logger.FromContext(r.Context()).
				With("err", err).
				With("stack", stack).
				Error("Panic recovered from handler")
			w.WriteHeader(http.StatusInternalServerError)
		}()

		next.ServeHTTP(w, r)
	})
}
