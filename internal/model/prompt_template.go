package model

type PromptTemplateFieldType string

const (
	PromptTemplateFieldText     PromptTemplateFieldType = "text"
	PromptTemplateFieldTextarea PromptTemplateFieldType = "textarea"
	PromptTemplateFieldSelect   PromptTemplateFieldType = "select"
	PromptTemplateFieldNumber   PromptTemplateFieldType = "number"
)

type PromptTemplateField struct {
	Name        string                  `json:"name"`
	Label       string                  `json:"label"`
	Type        PromptTemplateFieldType `json:"type"`
	Required    bool                    `json:"required"`
	MaxLength   int                     `json:"max_length,omitempty"`
	Options     []string                `json:"options,omitempty"`
	Placeholder string                  `json:"placeholder,omitempty"`
	HelpText    string                  `json:"help_text,omitempty"`
}

type PromptTemplate struct {
	TemplateID   string                `json:"template_id"`
	Name         string                `json:"name"`
	Description  string                `json:"description,omitempty"`
	Category     string                `json:"category,omitempty"`
	Version      int                   `json:"version"`
	Enabled      bool                  `json:"enabled"`
	TemplateBody string                `json:"template_body"`
	Fields       []PromptTemplateField `json:"fields"`
	Source       string                `json:"source,omitempty"`
	CreatedAt    int64                 `json:"created_at"`
	UpdatedAt    int64                 `json:"updated_at"`
}
