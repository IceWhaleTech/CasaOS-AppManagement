package common

import "context"

type (
	keyTypeProperties       int
	keyTypeInterpolationMap int
)

const (
	keyProperties       keyTypeProperties       = iota
	keyInterpolationMap keyTypeInterpolationMap = iota
)

// Returns a new context with the given properties for events.
func WithProperties(ctx context.Context, properties map[string]string) context.Context {
	return withMap(ctx, keyProperties, properties)
}

// Returns the properties from the context for events
func PropertiesFromContext(ctx context.Context) map[string]string {
	return mapFromContext(ctx, keyProperties)
}

// Returns a new context with the given interpolation map for compose app intepolation.
func WithInterpolationMap(ctx context.Context, interpolationMap map[string]string) context.Context {
	return withMap(ctx, keyInterpolationMap, interpolationMap)
}

// Returns the interpolation map from the context for compose app intepolation.
func InterpolationMapFromContext(ctx context.Context) map[string]string {
	return mapFromContext(ctx, keyInterpolationMap)
}

func withMap[T any](ctx context.Context, key T, value map[string]string) context.Context {
	return context.WithValue(ctx, key, value)
}

func mapFromContext[T any](ctx context.Context, key T) map[string]string {
	value := ctx.Value(key)
	if value == nil {
		return nil
	}

	if properties, ok := value.(map[string]string); ok {
		return properties
	}

	return nil
}
