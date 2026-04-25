package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/poly-workshop/slot-art/internal/model"
)

func (s *Store) CreateResult(ctx context.Context, result *model.Result, ttl time.Duration) error {
	key := "slot:v1:result:" + result.TaskID
	imagesJSON, _ := json.Marshal(result.Images)
	if err := s.rdb.HSet(ctx, key, map[string]any{
		"task_id":        result.TaskID,
		"images":         string(imagesJSON),
		"revised_prompt": result.RevisedPrompt,
		"generated_at":   result.GeneratedAt,
	}).Err(); err != nil {
		return err
	}
	s.rdb.Expire(ctx, key, ttl)
	return nil
}

func (s *Store) GetResult(ctx context.Context, taskID string) (*model.Result, error) {
	m, err := s.rdb.HGetAll(ctx, "slot:v1:result:"+taskID).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	var images []model.ResultImage
	if j := hGetString(m, "images"); j != "" {
		json.Unmarshal([]byte(j), &images)
	}
	return &model.Result{
		TaskID:        hGetString(m, "task_id"),
		Images:        images,
		RevisedPrompt: hGetString(m, "revised_prompt"),
		GeneratedAt:   hGetInt64(m, "generated_at"),
	}, nil
}
