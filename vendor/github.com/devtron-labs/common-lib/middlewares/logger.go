package middlewares

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
)

// InterceptorLogger adapts go-kit logger to interceptor logger.
func InterceptorLogger(enableLogger bool, lg *zap.SugaredLogger) logging.Logger {
	return logging.LoggerFunc(func(ctx context.Context, lvl logging.Level, msg string, fields ...any) {
		if !enableLogger {
			return
		}
		finalReq := extractedFields(fields)
		message := fmt.Sprintf("AUDIT_LOG: level: %v,requestMethod: %s, requestPayload: %s", lvl, fields[1], finalReq)
		lg.Info(message)
		fmt.Println("hello")
	})
}
func extractedFields(fields []any) []byte {
	length := len(fields)
	req := make(map[string]interface{})
	marshal, _ := json.Marshal(fields[length-1])
	err := json.Unmarshal(marshal, &req)
	if err != nil {
		return nil
	}
	removedFields := []string{"RunInCtx", "chartContent", "valuesYaml"}
	for _, field := range removedFields {
		delete(req, field)
	}
	finalReq, _ := json.Marshal(req)
	return finalReq
}
func GenerateLogFields(ctx context.Context, meta interceptors.CallMeta) logging.Fields {
	fields := logging.Fields{logging.MethodFieldKey, meta.Method}
	ctx = logging.InjectFields(ctx, fields)
	return fields
}
