package utils

func First[T, O any](n T, _ O) T {
	return n
}

func Second[T, O any](_ T, n O) O {
	return n
}

func Values[K comparable, V any](m map[K]V) []V {
	values := make([]V, 0, len(m))
	for _, v := range m {
		values = append(values, v)
	}
	return values
}

func Map[T any, R any](input []T, fn func(T) R) []R {
	result := make([]R, len(input))
	for i, v := range input {
		result[i] = fn(v)
	}
	return result
}

func Clone[T any](s []T) []T {
	if s == nil {
		return nil
	}
	dst := make([]T, len(s))
	copy(dst, s)
	return dst
}

// search a key in a generic value hashmap, if key is found cast it to its type else return a default
func GetOrDefault[T any](args map[string]any, key string, defaultValue T) T {
	value, ok := args[key]
	if !ok {
		return defaultValue
	}

	result, ok := value.(T)
	if !ok {
		return defaultValue
	}

	return result
}
