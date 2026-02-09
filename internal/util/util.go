package util

func GetOrDefault[T any](i *T, defaultValue T) T {
	if i == nil {
		return defaultValue
	}
	return *i
}

func ByPtr[T any](value T) *T {
	return &value
}
