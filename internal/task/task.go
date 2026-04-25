package task

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

const TaskTypeImageGenerate = "image:generate"

type GeneratePayload struct {
	TaskID          string       `json:"task_id"`
	UID             string       `json:"uid"`
	Prompt          string       `json:"prompt"`
	CreditID        string       `json:"credit_id"`
	Provider        string       `json:"provider"`
	ImageCount      int          `json:"image_count"`
	ReferenceImages []RefImgData `json:"reference_images"`
}

type RefImgData struct {
	ContentType string `json:"content_type"`
	Data        []byte `json:"data"`
}

func NewGenerateTask(p *GeneratePayload) (*asynq.Task, error) {
	payload, err := json.Marshal(p)
	if err != nil {
		return nil, err
	}
	return asynq.NewTask(TaskTypeImageGenerate, payload), nil
}
