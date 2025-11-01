package contextx

import "context"

func WithValue(ctx context.Context, key ContextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

func GetString(ctx context.Context, key ContextKey) string {
	v, ok := ctx.Value(key).(string)
	if ok {
		return v
	}
	return ""
}

func GetInt(ctx context.Context, key ContextKey) int {
	v, ok := ctx.Value(key).(int)
	if ok {
		return v
	}
	return 0
}

func GetBool(ctx context.Context, key ContextKey) bool {
	v, ok := ctx.Value(key).(bool)
	if ok {
		return v
	}
	return false
}

func GetFloat64(ctx context.Context, key ContextKey) float64 {
	v, ok := ctx.Value(key).(float64)
	if ok {
		return v
	}
	return 0
}

func Get(ctx context.Context, key ContextKey) interface{} {
	return ctx.Value(key)
}

func Set(ctx context.Context, key ContextKey, value interface{}) context.Context {
	return context.WithValue(ctx, key, value)
}

func MustGetString(ctx context.Context, key ContextKey) string {
	v := ctx.Value(key)
	s, ok := v.(string)
	if ok {
		return s
	}
	panic("context value is not a string")
}

func MustGetInt(ctx context.Context, key ContextKey) int {
	v := ctx.Value(key)
	i, ok := v.(int)
	if ok {
		return i
	}
	panic("context value is not an int")
}

func MustGetBool(ctx context.Context, key ContextKey) bool {
	v := ctx.Value(key)
	b, ok := v.(bool)
	if ok {
		return b
	}
	panic("context value is not a bool")
}

func MustGetFloat64(ctx context.Context, key ContextKey) float64 {
	v := ctx.Value(key)
	f, ok := v.(float64)
	if ok {
		return f
	}
	panic("context value is not a float64")
}

func Has(ctx context.Context, key ContextKey) bool {
	return ctx.Value(key) != nil
}

func Delete(ctx context.Context, key ContextKey) context.Context {
	return context.WithValue(ctx, key, nil)
}
