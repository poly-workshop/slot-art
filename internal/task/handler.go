package task

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"

	"github.com/poly-workshop/slot-art/internal/config"
	"github.com/poly-workshop/slot-art/internal/idgen"
	"github.com/poly-workshop/slot-art/internal/model"
	"github.com/poly-workshop/slot-art/internal/provider"
	"github.com/poly-workshop/slot-art/internal/store"
)

type Worker struct {
	server    *asynq.Server
	logger    *slog.Logger
	resultTTL time.Duration
	taskTTL   time.Duration
}

func NewWorker(cfg *config.Config, logger *slog.Logger) *Worker {
	srv := asynq.NewServer(
		asynq.RedisClientOpt{
			Addr:     cfg.Asynq.RedisAddr,
			Password: cfg.Asynq.RedisPassword,
		},
		asynq.Config{
			Concurrency: cfg.Asynq.Concurrency,
			Queues:      cfg.Asynq.Queues,
		},
	)
	return &Worker{server: srv, logger: logger, resultTTL: cfg.ResultTTL, taskTTL: cfg.TaskTTL}
}

func (w *Worker) Start(store *store.Store, enqueuer *Enqueuer) error {
	mux := asynq.NewServeMux()
	mux.HandleFunc(TaskTypeImageGenerate, w.makeHandler(store))
	return w.server.Run(mux)
}

func (w *Worker) Stop() {
	w.server.Shutdown()
}

func (w *Worker) makeHandler(st *store.Store) func(context.Context, *asynq.Task) error {
	return func(ctx context.Context, t *asynq.Task) error {
		var payload GeneratePayload
		if err := json.Unmarshal(t.Payload(), &payload); err != nil {
			return asynq.SkipRetry
		}

		w.logger.Info("processing task", "task_id", payload.TaskID, "provider", payload.Provider)

		st.UpdateTaskStatus(ctx, payload.TaskID, model.TaskProcessing, "")

		prov, err := provider.Get(payload.Provider)
		if err != nil {
			st.UpdateTaskStatus(ctx, payload.TaskID, model.TaskFailed, err.Error())
			return asynq.SkipRetry
		}

		var refImgs []provider.ReferenceImage
		for _, r := range payload.ReferenceImages {
			refImgs = append(refImgs, provider.ReferenceImage{
				ContentType: r.ContentType,
				Data:        r.Data,
			})
		}

		result, err := prov.Generate(ctx, provider.GenerateRequest{
			Prompt:          payload.Prompt,
			ReferenceImages: refImgs,
			ImageCount:      payload.ImageCount,
		})
		if err != nil {
			st.UpdateTaskStatus(ctx, payload.TaskID, model.TaskFailed, err.Error())
			return fmt.Errorf("generate: %w", err)
		}

		var images []model.ResultImage
		for _, img := range result.Images {
			images = append(images, model.ResultImage{
				ImageID: idgen.NewImageID(),
				URL:     img.URL,
				Width:   img.Width,
				Height:  img.Height,
				Format:  img.Format,
			})
		}

		if err := st.CreateResult(ctx, &model.Result{
			TaskID:        payload.TaskID,
			Images:        images,
			RevisedPrompt: firstRevisedPrompt(result.Images),
			GeneratedAt:   time.Now().Unix(),
		}, w.resultTTL); err != nil {
			return fmt.Errorf("store result: %w", err)
		}

		// Index each image URL by task_id+image_id for the download endpoint.
		for _, img := range images {
			st.Client().Set(ctx, fmt.Sprintf("slot:v1:task_image:%s:%s", payload.TaskID, img.ImageID), img.URL, w.resultTTL)
		}

		st.SetTaskImageCount(ctx, payload.TaskID, len(images))
		st.UpdateTaskStatus(ctx, payload.TaskID, model.TaskCompleted, "")

		w.logger.Info("task completed", "task_id", payload.TaskID, "images", len(images))
		return nil
	}
}

func firstRevisedPrompt(images []provider.GeneratedImage) string {
	for _, img := range images {
		if img.RevisedPrompt != "" {
			return img.RevisedPrompt
		}
	}
	return ""
}
