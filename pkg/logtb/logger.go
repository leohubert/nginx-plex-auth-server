package logtb

import (
	"context"

	"go.uber.org/zap"
)

type Format string

const (
	FormatPretty = Format("pretty")
	FormatJSON   = Format("json")
)

type Options struct {
	Format Format
}

func NewLogger(opts Options) (*zap.Logger, func()) {

	loggerOpts := zap.NewProductionConfig()
	if opts.Format == FormatPretty {
		loggerOpts = zap.NewDevelopmentConfig()
	}
	loggerOpts.DisableStacktrace = true
	logger, err := loggerOpts.Build()
	if err != nil {
		panic(err)
	}

	return logger, func() {
		_ = logger.Sync()
	}
}

type loggerCtxKeyType struct{}

var loggerCtxKey = loggerCtxKeyType{}

func ExtractLogger(ctx context.Context) *zap.Logger {
	v := ctx.Value(loggerCtxKey)
	if v == nil {
		return nil
	}
	return v.(*zap.Logger)
}

func InjectLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}
