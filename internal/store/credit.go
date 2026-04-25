package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/poly-workshop/slot-art/internal/model"
)

var ErrCreditAlreadyConsumed = fmt.Errorf("credit already consumed")

func creditKey(uid, creditID string) string {
	return fmt.Sprintf("slot:v1:credit:%s:%s", uid, creditID)
}

func (s *Store) CreateCredit(ctx context.Context, credit *model.Credit, ttl time.Duration) error {
	key := creditKey(credit.UID, credit.CreditID)
	if err := s.rdb.HSet(ctx, key, map[string]any{
		"credit_id":             credit.CreditID,
		"uid":                   credit.UID,
		"status":                string(credit.Status),
		"image_count":           credit.ImageCount,
		"remaining_image_count": credit.RemainingImageCount,
		"source":                credit.Source,
		"source_ref":            credit.SourceRef,
		"created_at":            credit.CreatedAt,
		"expires_at":            credit.ExpiresAt,
		"consumed_at":           credit.ConsumedAt,
	}).Err(); err != nil {
		return err
	}
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, "slot:v1:credit_id:"+credit.CreditID, credit.UID, 0)
	pipe.ZAdd(ctx, "slot:v1:credits", redis.Z{Score: float64(credit.CreatedAt), Member: credit.CreditID})
	pipe.ZAdd(ctx, "slot:v1:credits:user:"+credit.UID, redis.Z{Score: float64(credit.CreatedAt), Member: credit.CreditID})
	pipe.ZAdd(ctx, "slot:v1:credits:status:"+string(credit.Status), redis.Z{Score: float64(credit.CreatedAt), Member: credit.CreditID})
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) GetCredit(ctx context.Context, creditID string) (*model.Credit, error) {
	uid, err := s.rdb.Get(ctx, "slot:v1:credit_id:"+creditID).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.getCreditByKey(ctx, creditKey(uid, creditID))
}

func (s *Store) getCreditByKey(ctx context.Context, key string) (*model.Credit, error) {
	m, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	return &model.Credit{
		CreditID:            hGetString(m, "credit_id"),
		UID:                 hGetString(m, "uid"),
		Status:              model.CreditStatus(hGetString(m, "status")),
		ImageCount:          hGetInt(m, "image_count"),
		RemainingImageCount: hGetInt(m, "remaining_image_count"),
		Source:              hGetString(m, "source"),
		SourceRef:           hGetString(m, "source_ref"),
		CreatedAt:           hGetInt64(m, "created_at"),
		ExpiresAt:           hGetInt64(m, "expires_at"),
		ConsumedAt:          hGetInt64(m, "consumed_at"),
	}, nil
}

func (s *Store) ListCredits(ctx context.Context, uid string, limit int) ([]*model.Credit, error) {
	if limit <= 0 {
		limit = 50
	}
	ids, err := s.rdb.ZRevRange(ctx, "slot:v1:credits:user:"+uid, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	credits := make([]*model.Credit, 0, len(ids))
	for _, id := range ids {
		credit, err := s.GetCredit(ctx, id)
		if err == nil {
			credits = append(credits, credit)
		}
	}
	return credits, nil
}

func (s *Store) ListAllCredits(ctx context.Context, uid string, status model.CreditStatus, limit int) ([]*model.Credit, error) {
	if uid != "" {
		credits, err := s.ListCredits(ctx, uid, limit)
		if err != nil || status == "" {
			return credits, err
		}
		filtered := credits[:0]
		for _, credit := range credits {
			if credit.Status == status {
				filtered = append(filtered, credit)
			}
		}
		return filtered, nil
	}
	if limit <= 0 {
		limit = 100
	}
	index := "slot:v1:credits"
	if status != "" {
		index = "slot:v1:credits:status:" + string(status)
	}
	ids, err := s.rdb.ZRevRange(ctx, index, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	credits := make([]*model.Credit, 0, len(ids))
	for _, id := range ids {
		credit, err := s.GetCredit(ctx, id)
		if err == nil {
			credits = append(credits, credit)
		}
	}
	return credits, nil
}

func (s *Store) CountCreditsByStatus(ctx context.Context, status model.CreditStatus) (int64, error) {
	return s.rdb.ZCard(ctx, "slot:v1:credits:status:"+string(status)).Result()
}

func (s *Store) ConsumeCreditIfActive(ctx context.Context, creditID string) error {
	credit, err := s.GetCredit(ctx, creditID)
	if err != nil {
		return err
	}
	script := redis.NewScript(`
		local key = KEYS[1]
		local active_index = KEYS[2]
		local consumed_index = KEYS[3]
		local credit_id = ARGV[1]
		local now = ARGV[2]
		if redis.call("EXISTS", key) == 0 then
			return 1
		end
		local status = redis.call("HGET", key, "status")
		if status ~= "active" then
			return 2
		end
		redis.call("HSET", key, "status", "consumed", "remaining_image_count", 0, "consumed_at", now)
		redis.call("ZREM", active_index, credit_id)
		redis.call("ZADD", consumed_index, now, credit_id)
		return 0
	`)
	result, err := script.Run(ctx, s.rdb, []string{
		creditKey(credit.UID, credit.CreditID),
		"slot:v1:credits:status:" + string(model.CreditActive),
		"slot:v1:credits:status:" + string(model.CreditConsumed),
	}, credit.CreditID, time.Now().Unix()).Int()
	if err != nil {
		return err
	}
	switch result {
	case 0:
		return nil
	case 1:
		return ErrNotFound
	case 2:
		return ErrCreditAlreadyConsumed
	default:
		return fmt.Errorf("unexpected lua result: %d", result)
	}
}
