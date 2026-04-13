package redis

const (
	RedisWindowStoreServiceName = "redis"

	ErrWindowStoreAdd      = "WINDOWSTORE_ADD_FAILED"
	ErrWindowStoreAddMulti = "WINDOWSTORE_ADDMULTI_FAILED"
	ErrWindowStoreCount    = "WINDOWSTORE_COUNT_FAILED"
	ErrWindowStoreTrim     = "WINDOWSTORE_TRIM_FAILED"
	ErrWindowStoreSize     = "WINDOWSTORE_SIZE_FAILED"
	ErrWindowStoreClear    = "WINDOWSTORE_CLEAR_FAILED"
	ErrWindowStoreClose    = "WINDOWSTORE_CLOSE_FAILED"
)
