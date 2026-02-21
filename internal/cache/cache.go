package cache

import (
	"errors"
)

var ErrNotFound = errors.New("not found")

type Cacher[T any] interface {
	Set(key string, val T)
	Get(key string) (val T, found bool)
}
