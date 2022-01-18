package middlewares

import (
	"context"
	"net/http"

	log "github.com/sirupsen/logrus"
)

type loggerKeyType int
const LoggerKey  loggerKeyType = iota


type handler struct{ Log *log.Entry }

func New(mainLog *log.Entry) (h handler) { 
	return handler{
		Log: mainLog,
	} 
}

func GetLogReq(r *http.Request) *log.Entry {
	return getLogCtx(r.Context())
 }
 
 func getLogCtx(ctx context.Context) *log.Entry {
	logger := ctx.Value(LoggerKey).(*log.Entry)
 
	if logger == nil {
	   log.Fatal("Logger is missing in the context") // panics
	}
 
	return logger
 }

func (h handler) LoggingMiddleware(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		
		if traceSessionId, ok := GetTraceSessionIdCtx(ctx); ok {
			h.Log = h.Log.WithFields(log.Fields{"trace-session": traceSessionId})
		}
		if requestId, ok := GetRequestIdCtx(ctx); ok {
			h.Log = h.Log.WithFields(log.Fields{"request-id": requestId})
		}

		ctx = context.WithValue(ctx, LoggerKey, h.Log)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	}
	
	return http.HandlerFunc(fn)
}
