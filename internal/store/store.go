package store

import (
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Store struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *Store {
	return &Store{rdb: rdb}
}

func (s *Store) Client() *redis.Client {
	return s.rdb
}

func hGetInt(m map[string]string, key string) int {
	if v, ok := m[key]; ok {
		var n int
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}

func hGetInt64(m map[string]string, key string) int64 {
	if v, ok := m[key]; ok {
		var n int64
		fmt.Sscanf(v, "%d", &n)
		return n
	}
	return 0
}

func hGetString(m map[string]string, key string) string {
	return m[key]
}

var ErrNotFound = fmt.Errorf("not found")
