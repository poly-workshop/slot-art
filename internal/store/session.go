package store

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/poly-workshop/slot-art/internal/model"
)

func sessionKey(uid string) string { return "slot:v1:session:" + uid }

func (s *Store) CreateSession(ctx context.Context, uid, fingerprint string, ttl time.Duration) (*model.Session, error) {
	key := sessionKey(uid)
	now := time.Now().Unix()
	sess := &model.Session{
		UID:         uid,
		Fingerprint: fingerprint,
		CreatedAt:   now,
		LastSeen:    now,
	}
	if err := s.rdb.HSet(ctx, key, map[string]any{
		"uid":         uid,
		"fingerprint": fingerprint,
		"created_at":  now,
		"last_seen":   now,
	}).Err(); err != nil {
		return nil, err
	}
	pipe := s.rdb.TxPipeline()
	pipe.ZAdd(ctx, "slot:v1:sessions", redis.Z{Score: float64(now), Member: uid})
	pipe.Expire(ctx, key, ttl)
	_, _ = pipe.Exec(ctx)
	return sess, nil
}

func (s *Store) GetSession(ctx context.Context, uid string) (*model.Session, error) {
	key := sessionKey(uid)
	m, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	return &model.Session{
		UID:         hGetString(m, "uid"),
		Fingerprint: hGetString(m, "fingerprint"),
		CreatedAt:   hGetInt64(m, "created_at"),
		LastSeen:    hGetInt64(m, "last_seen"),
	}, nil
}

func (s *Store) TouchSession(ctx context.Context, uid string, ttl time.Duration) error {
	key := sessionKey(uid)
	exists, err := s.rdb.Exists(ctx, key).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return ErrNotFound
	}
	now := time.Now().Unix()
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, "last_seen", now)
	pipe.ZAdd(ctx, "slot:v1:sessions", redis.Z{Score: float64(now), Member: uid})
	pipe.Expire(ctx, key, ttl)
	_, err = pipe.Exec(ctx)
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, uid string) error {
	return s.rdb.Del(ctx, sessionKey(uid)).Err()
}

func (s *Store) UpdateSessionFingerprint(ctx context.Context, uid, fingerprint string, ttl time.Duration) error {
	key := sessionKey(uid)
	s.rdb.HSet(ctx, key, "fingerprint", fingerprint)
	s.rdb.Expire(ctx, key, ttl)
	return nil
}

func (s *Store) AddTaskToUser(ctx context.Context, uid, taskID string, ttl time.Duration) error {
	key := userTasksKey(uid)
	s.rdb.ZAdd(ctx, key, redis.Z{Score: float64(time.Now().Unix()), Member: taskID})
	s.rdb.Expire(ctx, key, ttl)
	return nil
}

func (s *Store) ListUserTasks(ctx context.Context, uid string, limit int) ([]string, error) {
	key := userTasksKey(uid)
	return s.rdb.ZRevRange(ctx, key, 0, int64(limit-1)).Result()
}

func (s *Store) ListSessions(ctx context.Context, limit int) ([]*model.Session, error) {
	if limit <= 0 {
		limit = 100
	}
	uids, err := s.rdb.ZRevRange(ctx, "slot:v1:sessions", 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	sessions := make([]*model.Session, 0, len(uids))
	for _, uid := range uids {
		sess, err := s.GetSession(ctx, uid)
		if err == nil {
			sessions = append(sessions, sess)
		}
	}
	return sessions, nil
}
