package openai

import (
	"apps/wealth-management/domain"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
)

const maxModelsBody = 1 << 20

type wireModels struct {
	Data []struct {
		ID string `json:"id"`
	} `json:"data"`
}

// Models lists the model ids the endpoint offers (GET {base}/models), sorted
// and without duplicates, so Settings can offer them instead of free text.
func (c *Client) Models(ctx context.Context) ([]string, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/models", nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	c.authorize(request)
	request.Header.Set("Accept", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrAIUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode/100 != 2 {
		return nil, providerError(response)
	}
	var decoded wireModels
	if err := json.NewDecoder(io.LimitReader(response.Body, maxModelsBody)).Decode(&decoded); err != nil {
		return nil, fmt.Errorf("%w: decode models response: %v", domain.ErrAIUnavailable, err)
	}
	models := make([]string, 0, len(decoded.Data))
	for _, model := range decoded.Data {
		if id := strings.TrimSpace(model.ID); id != "" {
			models = append(models, id)
		}
	}
	slices.Sort(models)
	return slices.Compact(models), nil
}
