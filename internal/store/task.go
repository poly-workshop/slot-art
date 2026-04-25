package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/poly-workshop/slot-art/internal/model"
)

func taskKey(taskID string) string { return "slot:v1:task:" + taskID }

func userTasksKey(uid string) string { return "slot:v1:tasks:user:" + uid }

func (s *Store) CreateTask(ctx context.Context, task *model.Task, ttl time.Duration) error {
	key := taskKey(task.TaskID)
	refKeysJSON, _ := json.Marshal(task.ReferenceImageKeys)
	slotValuesJSON, _ := json.Marshal(task.SlotValues)
	if err := s.rdb.HSet(ctx, key, map[string]any{
		"task_id":               task.TaskID,
		"uid":                   task.UID,
		"credit_id":             task.CreditID,
		"status":                string(task.Status),
		"prompt":                task.Prompt,
		"template_id":           task.TemplateID,
		"template_name":         task.TemplateName,
		"template_version":      task.TemplateVersion,
		"slot_values":           string(slotValuesJSON),
		"rendered_prompt":       task.RenderedPrompt,
		"reference_image_keys":  string(refKeysJSON),
		"provider":              task.Provider,
		"image_count_requested": task.ImageCountRequested,
		"image_count_generated": 0,
		"error_message":         "",
		"created_at":            task.CreatedAt,
		"updated_at":            task.UpdatedAt,
		"completed_at":          0,
	}).Err(); err != nil {
		return err
	}
	s.rdb.ZAdd(ctx, userTasksKey(task.UID), redis.Z{
		Score:  float64(task.UpdatedAt),
		Member: task.TaskID,
	})
	pipe := s.rdb.TxPipeline()
	pipe.ZAdd(ctx, "slot:v1:tasks", redis.Z{Score: float64(task.UpdatedAt), Member: task.TaskID})
	pipe.ZAdd(ctx, "slot:v1:tasks:status:"+string(task.Status), redis.Z{Score: float64(task.UpdatedAt), Member: task.TaskID})
	pipe.Expire(ctx, key, ttl)
	_, _ = pipe.Exec(ctx)
	return nil
}

func (s *Store) GetTask(ctx context.Context, taskID string) (*model.Task, error) {
	m, err := s.rdb.HGetAll(ctx, taskKey(taskID)).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	var refKeys []string
	if j := hGetString(m, "reference_image_keys"); j != "" {
		json.Unmarshal([]byte(j), &refKeys)
	}
	var slotValues map[string]string
	if j := hGetString(m, "slot_values"); j != "" {
		json.Unmarshal([]byte(j), &slotValues)
	}
	completedAt := hGetInt64(m, "completed_at")
	return &model.Task{
		TaskID:              hGetString(m, "task_id"),
		UID:                 hGetString(m, "uid"),
		CreditID:            hGetString(m, "credit_id"),
		Status:              model.TaskStatus(hGetString(m, "status")),
		Prompt:              hGetString(m, "prompt"),
		TemplateID:          hGetString(m, "template_id"),
		TemplateName:        hGetString(m, "template_name"),
		TemplateVersion:     hGetInt(m, "template_version"),
		SlotValues:          slotValues,
		RenderedPrompt:      hGetString(m, "rendered_prompt"),
		ReferenceImageKeys:  refKeys,
		Provider:            hGetString(m, "provider"),
		ImageCountRequested: hGetInt(m, "image_count_requested"),
		ImageCountGenerated: hGetInt(m, "image_count_generated"),
		ErrorMessage:        hGetString(m, "error_message"),
		CreatedAt:           hGetInt64(m, "created_at"),
		UpdatedAt:           hGetInt64(m, "updated_at"),
		CompletedAt:         completedAt,
	}, nil
}

func (s *Store) UpdateTaskStatus(ctx context.Context, taskID string, status model.TaskStatus, errMsg string) error {
	key := taskKey(taskID)
	oldStatus, _ := s.rdb.HGet(ctx, key, "status").Result()
	now := time.Now().Unix()
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, "status", string(status))
	pipe.HSet(ctx, key, "updated_at", now)
	if errMsg != "" {
		pipe.HSet(ctx, key, "error_message", errMsg)
	}
	if status == model.TaskCompleted || status == model.TaskFailed {
		pipe.HSet(ctx, key, "completed_at", now)
	}
	pipe.ZAdd(ctx, "slot:v1:tasks", redis.Z{Score: float64(now), Member: taskID})
	if oldStatus != "" {
		pipe.ZRem(ctx, "slot:v1:tasks:status:"+oldStatus, taskID)
	}
	pipe.ZAdd(ctx, "slot:v1:tasks:status:"+string(status), redis.Z{Score: float64(now), Member: taskID})
	_, err := pipe.Exec(ctx)
	return err
}

func (s *Store) SetTaskImageCount(ctx context.Context, taskID string, count int) error {
	return s.rdb.HSet(ctx, taskKey(taskID), "image_count_generated", count).Err()
}

func (s *Store) ListTasks(ctx context.Context, uid string, limit int) ([]*model.Task, error) {
	if limit <= 0 {
		limit = 50
	}
	ids, err := s.rdb.ZRevRange(ctx, userTasksKey(uid), 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	var tasks []*model.Task
	for _, id := range ids {
		task, err := s.GetTask(ctx, id)
		if err != nil {
			continue
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

func (s *Store) ListAllTasks(ctx context.Context, uid string, status model.TaskStatus, limit int) ([]*model.Task, error) {
	if uid != "" {
		tasks, err := s.ListTasks(ctx, uid, limit)
		if err != nil || status == "" {
			return tasks, err
		}
		filtered := tasks[:0]
		for _, task := range tasks {
			if task.Status == status {
				filtered = append(filtered, task)
			}
		}
		return filtered, nil
	}
	if limit <= 0 {
		limit = 100
	}
	index := "slot:v1:tasks"
	if status != "" {
		index = "slot:v1:tasks:status:" + string(status)
	}
	ids, err := s.rdb.ZRevRange(ctx, index, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	tasks := make([]*model.Task, 0, len(ids))
	for _, id := range ids {
		task, err := s.GetTask(ctx, id)
		if err == nil {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

func (s *Store) CountTasksByStatus(ctx context.Context, status model.TaskStatus) (int64, error) {
	return s.rdb.ZCard(ctx, "slot:v1:tasks:status:"+string(status)).Result()
}
