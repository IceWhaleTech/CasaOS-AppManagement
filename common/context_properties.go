package common

import "context"

const KeyNameContext = "context"

func WithProperties(ctx context.Context, properties map[string]string) context.Context {
	return context.WithValue(ctx, []byte(KeyNameContext), properties)
}

func PropertiesFromContext(ctx context.Context) map[string]string {
	value := ctx.Value([]byte(KeyNameContext))
	if value == nil {
		return map[string]string{}
	}

	if properties, ok := value.(map[string]string); ok {
		return properties
	}

	return map[string]string{}
}
