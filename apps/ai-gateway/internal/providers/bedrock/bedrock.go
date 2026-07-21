package bedrock

import (
	"context"
	"fmt"
	"io"
	"sync"

	"apps/ai-gateway/internal/domain"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrock"
	bedrocktypes "github.com/aws/aws-sdk-go-v2/service/bedrock/types"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"
)

const (
	roleSystem    = "system"
	roleUser      = "user"
	roleAssistant = "assistant"
	roleTool      = "tool"

	objectChatCompletion      = "chat.completion"
	objectChatCompletionChunk = "chat.completion.chunk"

	sseDataPrefix = "data: "
	sseDone       = "[DONE]"
)

type Provider struct {
	client  *bedrockruntime.Client
	control *bedrock.Client
	region  string
	ready   bool
	mu      sync.RWMutex
	probe   probeResult
}

type probeResult struct {
	discovered int
	candidates int
	model      string
}

func New(ctx context.Context, region string) (*Provider, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(cfg)
	return &Provider{
		client:  client,
		control: bedrock.NewFromConfig(cfg),
		region:  region,
	}, nil
}

func (p *Provider) Name() string { return domain.ProviderBedrock }

func (p *Provider) Chat(ctx context.Context, req domain.ChatRequest) (*domain.ChatResponse, error) {
	input, err := p.convertToBedrockRequest(req)
	if err != nil {
		return nil, err
	}

	output, err := p.client.Converse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("bedrock converse error: %w", err)
	}

	return p.convertToOpenAIResponse(output, req.Model), nil
}

func (p *Provider) ChatStream(ctx context.Context, req domain.ChatRequest, w io.Writer) (domain.Usage, error) {
	input, err := p.convertToBedrockStreamRequest(req)
	if err != nil {
		return domain.Usage{}, err
	}

	output, err := p.client.ConverseStream(ctx, input)
	if err != nil {
		return domain.Usage{}, fmt.Errorf("bedrock converse stream error: %w", err)
	}

	return p.processStream(output, w, req.Model)
}

func (p *Provider) Responses(ctx context.Context, req domain.ResponsesRequest) (*domain.ResponsesResponse, error) {
	return nil, domain.UnsupportedResponsesError(p.Name())
}

func (p *Provider) ResponsesStream(ctx context.Context, req domain.ResponsesRequest, w io.Writer) (domain.Usage, error) {
	return domain.Usage{}, domain.UnsupportedResponsesError(p.Name())
}

func (p *Provider) ListModels(ctx context.Context) (*domain.ModelsResponse, error) {
	knownModels := []domain.ModelInfo{
		{ID: "anthropic.claude-3-5-sonnet-20240620-v1:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "anthropic.claude-3-5-sonnet-20241022-v2:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "anthropic.claude-3-opus-20240229-v1:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "anthropic.claude-3-haiku-20240307-v1:0", Object: "model", OwnedBy: "anthropic"},
		{ID: "meta.llama3-70b-instruct-v1:0", Object: "model", OwnedBy: "meta"},
	}
	return &domain.ModelsResponse{
		Object: "list",
		Data:   knownModels,
	}, nil
}

func (p *Provider) Embeddings(ctx context.Context, req domain.EmbeddingRequest) (*domain.EmbeddingResponse, error) {
	return nil, fmt.Errorf("embeddings not implemented for bedrock in this gateway yet")
}

func (p *Provider) IsConfigured() bool {
	return p.client != nil
}

func (p *Provider) Ping(ctx context.Context) error {
	models, err := p.control.ListFoundationModels(ctx, &bedrock.ListFoundationModelsInput{
		ByOutputModality: bedrocktypes.ModelModalityText,
	})
	if err != nil {
		return fmt.Errorf("bedrock model discovery failed: %w", err)
	}

	candidates := 0
	for _, model := range models.ModelSummaries {
		if model.ModelId != nil && model.ModelLifecycle != nil &&
			model.ModelLifecycle.Status == bedrocktypes.FoundationModelLifecycleStatusActive &&
			supportsOnDemand(model.InferenceTypesSupported) {
			candidates++
		}
	}
	p.mu.Lock()
	p.probe.discovered = len(models.ModelSummaries)
	p.probe.candidates = candidates
	p.mu.Unlock()

	var lastErr error
	for _, model := range models.ModelSummaries {
		if model.ModelId == nil || model.ModelLifecycle == nil ||
			model.ModelLifecycle.Status != bedrocktypes.FoundationModelLifecycleStatusActive ||
			!supportsOnDemand(model.InferenceTypesSupported) {
			continue
		}

		_, err = p.client.Converse(ctx, &bedrockruntime.ConverseInput{
			ModelId: model.ModelId,
			Messages: []types.Message{{
				Role:    types.ConversationRoleUser,
				Content: []types.ContentBlock{&types.ContentBlockMemberText{Value: "ping"}},
			}},
			InferenceConfig: &types.InferenceConfiguration{MaxTokens: aws.Int32(1)},
		})
		if err == nil {
			p.mu.Lock()
			p.probe.model = aws.ToString(model.ModelId)
			p.mu.Unlock()
			return nil
		}
		lastErr = fmt.Errorf("model %q: %w", aws.ToString(model.ModelId), err)
	}
	if lastErr != nil {
		return lastErr
	}
	return fmt.Errorf("bedrock model discovery returned no active on-demand text models")
}

func (p *Provider) ReadySummary() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return fmt.Sprintf("region=%s discovered_models=%d probe_candidates=%d probe_model=%s", p.region, p.probe.discovered, p.probe.candidates, p.probe.model)
}

func supportsOnDemand(inferenceTypes []bedrocktypes.InferenceType) bool {
	for _, inferenceType := range inferenceTypes {
		if inferenceType == bedrocktypes.InferenceTypeOnDemand {
			return true
		}
	}
	return false
}

func (p *Provider) Usage(ctx context.Context) (any, error) {
	return nil, fmt.Errorf("usage not supported by bedrock")
}

func (p *Provider) IsReady() bool   { return p.ready }
func (p *Provider) SetReady(r bool) { p.ready = r }
