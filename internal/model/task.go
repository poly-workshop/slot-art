package model

type TaskStatus string

const (
	TaskPending    TaskStatus = "pending"
	TaskProcessing TaskStatus = "processing"
	TaskCompleted  TaskStatus = "completed"
	TaskFailed     TaskStatus = "failed"
)

type Task struct {
	TaskID              string            `json:"task_id"`
	UID                 string            `json:"uid"`
	CreditID            string            `json:"credit_id"`
	Status              TaskStatus        `json:"status"`
	Prompt              string            `json:"prompt"`
	TemplateID          string            `json:"template_id,omitempty"`
	TemplateName        string            `json:"template_name,omitempty"`
	TemplateVersion     int               `json:"template_version,omitempty"`
	SlotValues          map[string]string `json:"slot_values,omitempty"`
	RenderedPrompt      string            `json:"rendered_prompt,omitempty"`
	ReferenceImageKeys  []string          `json:"reference_image_keys"`
	Provider            string            `json:"provider"`
	ImageCountRequested int               `json:"image_count_requested"`
	ImageCountGenerated int               `json:"image_count_generated"`
	ErrorMessage        string            `json:"error_message,omitempty"`
	CreatedAt           int64             `json:"created_at"`
	UpdatedAt           int64             `json:"updated_at"`
	CompletedAt         int64             `json:"completed_at,omitempty"`
}
