package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/poly-workshop/slot-art/internal/model"
)

var ErrCDKeyRedeemed = fmt.Errorf("cdkey already redeemed")
var ErrCDKeyRevoked = fmt.Errorf("cdkey revoked")
var ErrCDKeyExpired = fmt.Errorf("cdkey expired")

func NormalizeCDKey(key string) string {
	key = strings.ToUpper(strings.TrimSpace(key))
	key = strings.ReplaceAll(key, "-", "")
	key = strings.ReplaceAll(key, " ", "")
	return key
}

func HashCDKey(key string) string {
	sum := sha256.Sum256([]byte(NormalizeCDKey(key)))
	return hex.EncodeToString(sum[:])
}

func MaskCDKey(key string) string {
	normalized := NormalizeCDKey(key)
	if len(normalized) <= 8 {
		return "****"
	}
	return normalized[:4] + "****" + normalized[len(normalized)-4:]
}

func cdkeyKey(hash string) string { return "slot:v1:cdkey:" + hash }

func cdkeyIDKey(id string) string { return "slot:v1:cdkey_id:" + id }

func (s *Store) CreateCDKeys(ctx context.Context, cdkeys []*model.CDKey) error {
	pipe := s.rdb.TxPipeline()
	for _, cdkey := range cdkeys {
		key := cdkeyKey(cdkey.KeyHash)
		pipe.HSet(ctx, key, map[string]any{
			"cdkey_id":        cdkey.CDKeyID,
			"key_hash":        cdkey.KeyHash,
			"masked_key":      cdkey.MaskedKey,
			"batch_id":        cdkey.BatchID,
			"status":          string(cdkey.Status),
			"image_count":     cdkey.ImageCount,
			"expires_at":      cdkey.ExpiresAt,
			"redeemed_by_uid": cdkey.RedeemedByUID,
			"redeemed_at":     cdkey.RedeemedAt,
			"created_by":      cdkey.CreatedBy,
			"created_at":      cdkey.CreatedAt,
			"note":            cdkey.Note,
		})
		pipe.Set(ctx, cdkeyIDKey(cdkey.CDKeyID), cdkey.KeyHash, 0)
		pipe.ZAdd(ctx, "slot:v1:cdkeys", redis.Z{Score: float64(cdkey.CreatedAt), Member: cdkey.CDKeyID})
		pipe.ZAdd(ctx, "slot:v1:cdkeys:status:"+string(cdkey.Status), redis.Z{Score: float64(cdkey.CreatedAt), Member: cdkey.CDKeyID})
		if cdkey.BatchID != "" {
			pipe.ZAdd(ctx, "slot:v1:cdkeys:batch:"+cdkey.BatchID, redis.Z{Score: float64(cdkey.CreatedAt), Member: cdkey.CDKeyID})
		}
	}
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) GetCDKey(ctx context.Context, cdkeyID string) (*model.CDKey, error) {
	hash, err := s.rdb.Get(ctx, cdkeyIDKey(cdkeyID)).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.GetCDKeyByHash(ctx, hash)
}

func (s *Store) GetCDKeyByHash(ctx context.Context, hash string) (*model.CDKey, error) {
	m, err := s.rdb.HGetAll(ctx, cdkeyKey(hash)).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	return &model.CDKey{
		CDKeyID:       hGetString(m, "cdkey_id"),
		KeyHash:       hGetString(m, "key_hash"),
		MaskedKey:     hGetString(m, "masked_key"),
		BatchID:       hGetString(m, "batch_id"),
		Status:        model.CDKeyStatus(hGetString(m, "status")),
		ImageCount:    hGetInt(m, "image_count"),
		ExpiresAt:     hGetInt64(m, "expires_at"),
		RedeemedByUID: hGetString(m, "redeemed_by_uid"),
		RedeemedAt:    hGetInt64(m, "redeemed_at"),
		CreatedBy:     hGetString(m, "created_by"),
		CreatedAt:     hGetInt64(m, "created_at"),
		Note:          hGetString(m, "note"),
	}, nil
}

func (s *Store) ListCDKeys(ctx context.Context, status model.CDKeyStatus, batchID string, limit int) ([]*model.CDKey, error) {
	if limit <= 0 {
		limit = 50
	}
	index := "slot:v1:cdkeys"
	if status != "" {
		index = "slot:v1:cdkeys:status:" + string(status)
	} else if batchID != "" {
		index = "slot:v1:cdkeys:batch:" + batchID
	}
	ids, err := s.rdb.ZRevRange(ctx, index, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	items := make([]*model.CDKey, 0, len(ids))
	for _, id := range ids {
		cdkey, err := s.GetCDKey(ctx, id)
		if err == nil {
			items = append(items, cdkey)
		}
	}
	return items, nil
}

func (s *Store) CountCDKeysByStatus(ctx context.Context, status model.CDKeyStatus) (int64, error) {
	return s.rdb.ZCard(ctx, "slot:v1:cdkeys:status:"+string(status)).Result()
}

func (s *Store) RevokeCDKey(ctx context.Context, cdkeyID string) (*model.CDKey, error) {
	hash, err := s.rdb.Get(ctx, cdkeyIDKey(cdkeyID)).Result()
	if err == redis.Nil {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	script := redis.NewScript(`
		local key = KEYS[1]
		local created_index = KEYS[2]
		local revoked_index = KEYS[3]
		local cdkey_id = ARGV[1]
		local now = ARGV[2]
		if redis.call("EXISTS", key) == 0 then
			return 1
		end
		local status = redis.call("HGET", key, "status")
		if status == "redeemed" then
			return 2
		end
		if status ~= "revoked" then
			redis.call("HSET", key, "status", "revoked")
			redis.call("ZREM", created_index, cdkey_id)
			redis.call("ZADD", revoked_index, now, cdkey_id)
		end
		return 0
	`)
	result, err := script.Run(ctx, s.rdb, []string{
		cdkeyKey(hash),
		"slot:v1:cdkeys:status:" + string(model.CDKeyCreated),
		"slot:v1:cdkeys:status:" + string(model.CDKeyRevoked),
	}, cdkeyID, time.Now().Unix()).Int()
	if err != nil {
		return nil, err
	}
	switch result {
	case 0:
		return s.GetCDKey(ctx, cdkeyID)
	case 1:
		return nil, ErrNotFound
	case 2:
		return nil, ErrCDKeyRedeemed
	default:
		return nil, fmt.Errorf("unexpected lua result: %d", result)
	}
}

func (s *Store) RedeemCDKey(ctx context.Context, plainKey, uid string, credit *model.Credit, creditTTL time.Duration) (*model.CDKey, error) {
	hash := HashCDKey(plainKey)
	key := cdkeyKey(hash)
	now := time.Now().Unix()
	script := redis.NewScript(`
		local cdkey_key = KEYS[1]
		local cdkeys_created = KEYS[2]
		local cdkeys_redeemed = KEYS[3]
		local cdkeys_expired = KEYS[4]
		local credit_key = KEYS[5]
		local user_credits = KEYS[6]
		local all_credits = KEYS[7]
		local active_credits = KEYS[8]
		local now = tonumber(ARGV[1])
		local uid = ARGV[2]
		local credit_id = ARGV[3]
		local credit_ttl = tonumber(ARGV[4])
		if redis.call("EXISTS", cdkey_key) == 0 then
			return 1
		end
		local status = redis.call("HGET", cdkey_key, "status")
		local cdkey_id = redis.call("HGET", cdkey_key, "cdkey_id")
		local expires_at = tonumber(redis.call("HGET", cdkey_key, "expires_at") or "0")
		if status == "revoked" then
			return 2
		end
		if status == "redeemed" then
			return 3
		end
		if expires_at > 0 and expires_at < now then
			redis.call("HSET", cdkey_key, "status", "expired")
			redis.call("ZREM", cdkeys_created, cdkey_id)
			redis.call("ZADD", cdkeys_expired, now, cdkey_id)
			return 4
		end
		if status ~= "created" then
			return 3
		end
		local image_count = redis.call("HGET", cdkey_key, "image_count")
		redis.call("HSET", cdkey_key,
			"status", "redeemed",
			"redeemed_by_uid", uid,
			"redeemed_at", now,
			"credit_id", credit_id)
		redis.call("ZREM", cdkeys_created, cdkey_id)
		redis.call("ZADD", cdkeys_redeemed, now, cdkey_id)
		redis.call("HSET", credit_key,
			"credit_id", credit_id,
			"uid", uid,
			"status", "active",
			"image_count", image_count,
			"remaining_image_count", image_count,
			"source", "cdkey",
			"source_ref", cdkey_id,
			"created_at", now,
			"expires_at", ARGV[5],
			"consumed_at", 0)
		redis.call("SET", "slot:v1:credit_id:" .. credit_id, uid)
		redis.call("ZADD", user_credits, now, credit_id)
		redis.call("ZADD", all_credits, now, credit_id)
		redis.call("ZADD", active_credits, now, credit_id)
		if credit_ttl > 0 then
			redis.call("EXPIRE", credit_key, credit_ttl)
		end
		return 0
	`)
	result, err := script.Run(ctx, s.rdb, []string{
		key,
		"slot:v1:cdkeys:status:" + string(model.CDKeyCreated),
		"slot:v1:cdkeys:status:" + string(model.CDKeyRedeemed),
		"slot:v1:cdkeys:status:" + string(model.CDKeyExpired),
		creditKey(uid, credit.CreditID),
		"slot:v1:credits:user:" + uid,
		"slot:v1:credits",
		"slot:v1:credits:status:" + string(model.CreditActive),
	}, now, uid, credit.CreditID, int(creditTTL.Seconds()), credit.ExpiresAt).Int()
	if err != nil {
		return nil, err
	}
	switch result {
	case 0:
		return s.GetCDKeyByHash(ctx, hash)
	case 1:
		return nil, ErrNotFound
	case 2:
		return nil, ErrCDKeyRevoked
	case 3:
		return nil, ErrCDKeyRedeemed
	case 4:
		return nil, ErrCDKeyExpired
	default:
		return nil, fmt.Errorf("unexpected lua result: %d", result)
	}
}
