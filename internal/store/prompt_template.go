package store

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/poly-workshop/slot-art/internal/model"
)

func promptTemplateKey(id string) string { return "slot:v1:prompt_template:" + id }

func (s *Store) CreatePromptTemplate(ctx context.Context, tmpl *model.PromptTemplate) error {
	if tmpl.TemplateID == "" {
		return fmt.Errorf("template_id is required")
	}
	now := time.Now().Unix()
	if tmpl.CreatedAt == 0 {
		tmpl.CreatedAt = now
	}
	if tmpl.UpdatedAt == 0 {
		tmpl.UpdatedAt = now
	}
	if tmpl.Version == 0 {
		tmpl.Version = 1
	}
	fieldsJSON, err := json.Marshal(tmpl.Fields)
	if err != nil {
		return err
	}
	key := promptTemplateKey(tmpl.TemplateID)
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, map[string]any{
		"template_id":   tmpl.TemplateID,
		"name":          tmpl.Name,
		"description":   tmpl.Description,
		"category":      tmpl.Category,
		"version":       tmpl.Version,
		"enabled":       boolInt(tmpl.Enabled),
		"template_body": tmpl.TemplateBody,
		"fields":        string(fieldsJSON),
		"source":        tmpl.Source,
		"created_at":    tmpl.CreatedAt,
		"updated_at":    tmpl.UpdatedAt,
	})
	pipe.ZAdd(ctx, "slot:v1:prompt_templates", redis.Z{Score: float64(tmpl.UpdatedAt), Member: tmpl.TemplateID})
	if tmpl.Enabled {
		pipe.ZAdd(ctx, "slot:v1:prompt_templates:enabled", redis.Z{Score: float64(tmpl.UpdatedAt), Member: tmpl.TemplateID})
	} else {
		pipe.ZRem(ctx, "slot:v1:prompt_templates:enabled", tmpl.TemplateID)
	}
	_, err = pipe.Exec(ctx)
	return err
}

func (s *Store) UpdatePromptTemplate(ctx context.Context, tmpl *model.PromptTemplate) error {
	existing, err := s.GetPromptTemplate(ctx, tmpl.TemplateID)
	if err != nil {
		return err
	}
	if tmpl.CreatedAt == 0 {
		tmpl.CreatedAt = existing.CreatedAt
	}
	if tmpl.Version == 0 {
		tmpl.Version = existing.Version + 1
	}
	tmpl.UpdatedAt = time.Now().Unix()
	return s.CreatePromptTemplate(ctx, tmpl)
}

func (s *Store) GetPromptTemplate(ctx context.Context, templateID string) (*model.PromptTemplate, error) {
	m, err := s.rdb.HGetAll(ctx, promptTemplateKey(templateID)).Result()
	if err != nil {
		return nil, err
	}
	if len(m) == 0 {
		return nil, ErrNotFound
	}
	var fields []model.PromptTemplateField
	if raw := hGetString(m, "fields"); raw != "" {
		_ = json.Unmarshal([]byte(raw), &fields)
	}
	return &model.PromptTemplate{
		TemplateID:   hGetString(m, "template_id"),
		Name:         hGetString(m, "name"),
		Description:  hGetString(m, "description"),
		Category:     hGetString(m, "category"),
		Version:      hGetInt(m, "version"),
		Enabled:      hGetInt(m, "enabled") == 1,
		TemplateBody: hGetString(m, "template_body"),
		Fields:       fields,
		Source:       hGetString(m, "source"),
		CreatedAt:    hGetInt64(m, "created_at"),
		UpdatedAt:    hGetInt64(m, "updated_at"),
	}, nil
}

func (s *Store) ListPromptTemplates(ctx context.Context, includeDisabled bool, limit int) ([]*model.PromptTemplate, error) {
	if limit <= 0 {
		limit = 100
	}
	index := "slot:v1:prompt_templates:enabled"
	if includeDisabled {
		index = "slot:v1:prompt_templates"
	}
	ids, err := s.rdb.ZRevRange(ctx, index, 0, int64(limit-1)).Result()
	if err != nil {
		return nil, err
	}
	items := make([]*model.PromptTemplate, 0, len(ids))
	for _, id := range ids {
		tmpl, err := s.GetPromptTemplate(ctx, id)
		if err == nil {
			items = append(items, tmpl)
		}
	}
	return items, nil
}

func (s *Store) SetPromptTemplateEnabled(ctx context.Context, templateID string, enabled bool) (*model.PromptTemplate, error) {
	tmpl, err := s.GetPromptTemplate(ctx, templateID)
	if err != nil {
		return nil, err
	}
	tmpl.Enabled = enabled
	tmpl.UpdatedAt = time.Now().Unix()
	if err := s.CreatePromptTemplate(ctx, tmpl); err != nil {
		return nil, err
	}
	return s.GetPromptTemplate(ctx, templateID)
}

func (s *Store) DeletePromptTemplate(ctx context.Context, templateID string) error {
	pipe := s.rdb.TxPipeline()
	pipe.Del(ctx, promptTemplateKey(templateID))
	pipe.ZRem(ctx, "slot:v1:prompt_templates", templateID)
	pipe.ZRem(ctx, "slot:v1:prompt_templates:enabled", templateID)
	_, err := pipe.Exec(ctx)
	return err
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
