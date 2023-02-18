package fp

import "errors"

func FMap[T any, U any](vs []T, f func(T) U) (us []U) {
	us = make([]U, len(vs))

	for i, v := range vs {
		us[i] = f(v)
	}

	return
}

var ErrTuple = errors.New(`"key/value" pair is missing "value"`)

type Tuple [2]string

func (t Tuple) HasValue() bool { return len(t) == 2 }
