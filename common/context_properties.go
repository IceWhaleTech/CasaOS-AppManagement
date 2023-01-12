package common

import "context"

type contextKeyType int

const contextKey contextKeyType = iota

func WithProperties(ctx context.Context, properties map[string]string) context.Context {
	return context.WithValue(ctx, contextKey, properties)
}

func PropertiesFromContext(ctx context.Context) map[string]string {
	value := ctx.Value(contextKey)
	if value == nil {
		return map[string]string{}
	}

	if properties, ok := value.(map[string]string); ok {
		return properties
	}

	return map[string]string{}
}
