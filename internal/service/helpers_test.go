package service

import (
	"strings"
	"testing"

	"github.com/poly-workshop/slot-art/internal/model"
)

func TestRenderPromptValidatesAndRendersSlots(t *testing.T) {
	tmpl := &model.PromptTemplate{
		Enabled:      true,
		TemplateBody: "Create a {{ style }} image of {{ subject }}.",
		Fields: []model.PromptTemplateField{
			{Name: "subject", Type: model.PromptTemplateFieldText, Required: true, MaxLength: 20},
			{Name: "style", Type: model.PromptTemplateFieldSelect, Required: true, Options: []string{"pixel art", "watercolor"}},
		},
	}

	got, err := renderPrompt(tmpl, map[string]string{"subject": "cat", "style": "pixel art"})
	if err != nil {
		t.Fatalf("renderPrompt() error = %v", err)
	}
	if got != "Create a pixel art image of cat." {
		t.Fatalf("renderPrompt() = %q", got)
	}
}

func TestRenderPromptRejectsInvalidInput(t *testing.T) {
	tmpl := &model.PromptTemplate{
		Enabled:      true,
		TemplateBody: "Create {{ subject }} in {{ unknown }}.",
		Fields: []model.PromptTemplateField{
			{Name: "subject", Type: model.PromptTemplateFieldText, Required: true, MaxLength: 3},
		},
	}

	if _, err := renderPrompt(tmpl, map[string]string{"subject": "kitten"}); err == nil || !strings.Contains(err.Error(), "exceeds max length") {
		t.Fatalf("renderPrompt() max length error = %v", err)
	}
	if _, err := renderPrompt(tmpl, map[string]string{"subject": "cat"}); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("renderPrompt() unknown placeholder error = %v", err)
	}
}

func TestPublicTaskConversionDoesNotLeakPrompt(t *testing.T) {
	task := &model.Task{
		TaskID:         "task_1",
		TemplateID:     "tmpl_1",
		TemplateName:   "Safe summary",
		Prompt:         "secret rendered prompt",
		RenderedPrompt: "secret rendered prompt",
		SlotValues:     map[string]string{"subject": "secret"},
	}

	publicTask := toProtoPublicTask(task, nil)
	if publicTask.GetTemplateName() != "Safe summary" {
		t.Fatalf("TemplateName = %q", publicTask.GetTemplateName())
	}
	if strings.Contains(publicTask.String(), "secret") {
		t.Fatalf("public task leaked prompt or slot values: %s", publicTask.String())
	}
}
