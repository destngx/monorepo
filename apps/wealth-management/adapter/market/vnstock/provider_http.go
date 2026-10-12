package vnstock

import (
	"context"
	"fmt"
	"net/http"
)

func (p *Provider) do(ctx context.Context, path string) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.baseURL+path, nil)
	if err != nil {
		return nil, err
	}

	response, err := p.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	if response.StatusCode >= http.StatusBadRequest {
		defer response.Body.Close()
		return nil, fmt.Errorf("vnstock request failed with status %d", response.StatusCode)
	}

	return response, nil
}
