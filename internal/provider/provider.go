package provider

import (
	"context"
)

type Provider interface {
	Name() string
	Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error)
}

type GenerateRequest struct {
	Prompt          string
	ReferenceImages []ReferenceImage
	ImageCount      int
}

type ReferenceImage struct {
	ContentType string
	Data        []byte
}

type GeneratedImage struct {
	URL           string `json:"url"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Format        string `json:"format"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type GenerateResponse struct {
	Images []GeneratedImage `json:"images"`
}

