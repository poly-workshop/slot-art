package model

type ResultImage struct {
	ImageID string `json:"image_id"`
	URL     string `json:"url"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
	Format  string `json:"format"`
}

type Result struct {
	TaskID        string        `json:"task_id"`
	Images        []ResultImage `json:"images"`
	RevisedPrompt string        `json:"revised_prompt,omitempty"`
	GeneratedAt   int64         `json:"generated_at"`
}
