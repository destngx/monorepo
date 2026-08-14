package service

import (
	"testing"

	"apps/ai-gateway/config"

	"github.com/stretchr/testify/assert"
)

const retiredGitHubModelsProvider = "github-models"

func TestRegistryDoesNotRegisterRetiredGitHubModelsProvider(t *testing.T) {
	registry := NewRegistry(&config.Config{})

	_, err := registry.Get(retiredGitHubModelsProvider)

	assert.Error(t, err)
	assert.NotContains(t, registry.Providers(), retiredGitHubModelsProvider)
}
