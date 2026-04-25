package store

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/poly-workshop/slot-art/internal/model"
)

func refImgKey(uid, refID string) string { return fmt.Sprintf("slot:v1:refimg:%s:%s", uid, refID) }

func userRefImgsKey(uid string) string { return "slot:v1:refimgs:user:" + uid }

func (s *Store) CreateRefImg(ctx context.Context, img *model.RefImg, ttl time.Duration) error {
	key := refImgKey(img.UID, img.RefID)
	fields := map[string]any{
		"ref_id":       img.RefID,
		"uid":          img.UID,
		"filename":     img.Filename,
		"content_type": img.ContentType,
		"size_bytes":   img.SizeBytes,
		"width":        img.Width,
		"height":       img.Height,
		"uploaded_at":  img.UploadedAt,
	}
	if img.Data != nil {
		fields["data"] = img.Data
	}
	if img.FilePath != "" {
		fields["file_path"] = img.FilePath
	}
	if err := s.rdb.HSet(ctx, key, fields).Err(); err != nil {
		return err
	}
	s.rdb.ZAdd(ctx, userRefImgsKey(img.UID), redis.Z{
		Score:  float64(img.UploadedAt),
		Member: img.RefID,
	})
	s.rdb.Expire(ctx, key, ttl)
	s.rdb.Expire(ctx, userRefImgsKey(img.UID), ttl)
	return nil
}

func (s *Store) GetRefImg(ctx context.Context, uid, refID string) (*model.RefImg, error) {
	key := refImgKey(uid, refID)
	m, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	img := &model.RefImg{
		RefID:       hGetString(m, "ref_id"),
		UID:         hGetString(m, "uid"),
		Filename:    hGetString(m, "filename"),
		ContentType: hGetString(m, "content_type"),
		SizeBytes:   hGetInt64(m, "size_bytes"),
		Width:       hGetInt(m, "width"),
		Height:      hGetInt(m, "height"),
		UploadedAt:  hGetInt64(m, "uploaded_at"),
	}
	if v, ok := m["data"]; ok {
		img.Data = []byte(v)
	}
	img.FilePath = hGetString(m, "file_path")
	return img, nil
}

func (s *Store) DeleteRefImg(ctx context.Context, uid, refID string) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, refImgKey(uid, refID))
	pipe.ZRem(ctx, userRefImgsKey(uid), refID)
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) ListRefImgs(ctx context.Context, uid string) ([]*model.RefImg, error) {
	ids, err := s.rdb.ZRevRange(ctx, userRefImgsKey(uid), 0, -1).Result()
	if err != nil {
		return nil, err
	}
	var imgs []*model.RefImg
	for _, id := range ids {
		img, err := s.GetRefImg(ctx, uid, id)
		if err != nil {
			continue
		}
		imgs = append(imgs, img)
	}
	return imgs, nil
}
