package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type OpenAIConfig struct {
	APIKey   string
	Endpoint string
	Model    string
}

type OpenAIProvider struct {
	client   *http.Client
	apiKey   string
	endpoint string
	model    string
}

func NewOpenAI(cfg OpenAIConfig) *OpenAIProvider {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = "https://api.openai.com"
	}
	return &OpenAIProvider{
		client:   &http.Client{Timeout: 120 * time.Second},
		apiKey:   cfg.APIKey,
		model:    cfg.Model,
		endpoint: endpoint,
	}
}

func (p *OpenAIProvider) Name() string { return "openai_image2" }

type openAIRequest struct {
	Model          string `json:"model"`
	Prompt         string `json:"prompt"`
	N              int    `json:"n"`
	ResponseFormat string `json:"response_format"`
}

type openAIResponse struct {
	Data []struct {
		URL           string `json:"url"`
		RevisedPrompt string `json:"revised_prompt"`
	} `json:"data"`
}

func (p *OpenAIProvider) Generate(ctx context.Context, req GenerateRequest) (*GenerateResponse, error) {
	body := openAIRequest{
		Model:          p.model,
		Prompt:         req.Prompt,
		N:              req.ImageCount,
		ResponseFormat: "url",
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+"/v1/images/generations", bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("api error %d: %s", resp.StatusCode, string(respBody))
	}

	var apiResp openAIResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	images := make([]GeneratedImage, 0, len(apiResp.Data))
	for _, d := range apiResp.Data {
		images = append(images, GeneratedImage{
			URL:           d.URL,
			Format:        "png",
			RevisedPrompt: d.RevisedPrompt,
		})
	}

	return &GenerateResponse{Images: images}, nil
}
