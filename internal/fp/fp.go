package fp

func FMap[T any, U any](vs []T, f func(T) U) (us []U) {
	us = make([]U, len(vs))

	for i, v := range vs {
		us[i] = f(v)
	}

	return
}
