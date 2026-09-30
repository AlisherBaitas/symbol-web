package ai

import (
	"net/http"
	"os"
	"time"
)

type AIClient struct {
	baseURL string
	model string
	client *http.Client
}

// Create new AI Client
func NewAIClient() *AIClient {
    return &AIClient{
        baseURL: os.Getenv("LLM_BASE_URL"),
        model:   os.Getenv("LLM_MODEL"),
        client:  &http.Client{Timeout: 10 * time.Second},
    }
}



