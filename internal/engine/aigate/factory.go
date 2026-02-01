package aigate

import (
	"os"
	"time"

	"github.com/RodrigoBeloyanis/livespot/internal/config"
	"github.com/RodrigoBeloyanis/livespot/internal/infra/openai"
)

func NewOpenAIClient(cfg config.Config) (*openai.Client, error) {
	key := os.Getenv("OPENAI_API_KEY")
	return openai.NewClient(cfg.OpenAIBaseURL, key, time.Duration(cfg.AIGateTimeoutMs)*time.Millisecond)
}
