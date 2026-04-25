package task

import (
	"github.com/hibiken/asynq"

	"github.com/poly-workshop/slot-art/internal/config"
)

type Enqueuer struct {
	client *asynq.Client
}

func NewEnqueuer(addr, password string) *Enqueuer {
	return &Enqueuer{
		client: asynq.NewClient(asynq.RedisClientOpt{
			Addr:     addr,
			Password: password,
		}),
	}
}

func (e *Enqueuer) EnqueueGenerate(payload *GeneratePayload) (*asynq.TaskInfo, error) {
	task, err := NewGenerateTask(payload)
	if err != nil {
		return nil, err
	}
	return e.client.Enqueue(task,
		asynq.Queue("critical"),
		asynq.Timeout(config.DefaultAsynqTimeout()),
		asynq.MaxRetry(3),
		asynq.Retention(config.DefaultTaskRetention()),
	)
}

func (e *Enqueuer) Close() error {
	return e.client.Close()
}
