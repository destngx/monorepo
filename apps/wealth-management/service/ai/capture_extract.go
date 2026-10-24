package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
)

type extractionOutcome struct {
	value domain.CaptureExtraction
	err   error
}

// reason is what the review screen shows for a material without events.
func (o extractionOutcome) reason() string {
	switch {
	case o.err == nil:
		return firstNonBlank(o.value.Reason, "Nothing readable.")
	case errors.Is(o.err, domain.ErrAIOutput):
		return "The model's answer could not be read. Try again or use another model."
	case IsUserFacing(o.err):
		return o.err.Error()
	default:
		return "Extraction failed. Try again."
	}
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// extractAll reads every material, a few at a time. A material that fails
// is reported as unreadable; when all fail for a reason other than the
// model's output, the first error is returned instead.
func (s *Service) extractAll(ctx context.Context, request port.CaptureRequest, system string, screenLabels []string, promptKeys []string, report func(port.CaptureProgress)) ([]extractionOutcome, error) {
	outcomes := make([]extractionOutcome, len(request.Materials))
	slots := make(chan struct{}, captureConcurrency)
	var wg sync.WaitGroup
	for index, material := range request.Materials {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			report(port.CaptureProgress{Stage: "extract", MaterialID: material.ID, Status: "running"})
			value, err := s.extract(ctx, material, system, screenLabels, promptKeys, request.Model)
			outcomes[index] = extractionOutcome{value: value, err: err}
			event := port.CaptureProgress{Stage: "extract", MaterialID: material.ID, Status: "ok", Events: len(value.Events)}
			switch {
			case err != nil:
				event.Status, event.Message = "error", outcomes[index].reason()
				s.log.LogError(ctx, "capture extraction failed", err, slog.String("material_kind", string(material.Kind)))
			case value.Unreadable:
				event.Status, event.Message = "unreadable", outcomes[index].reason()
			}
			report(event)
		}()
	}
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var firstErr error
	for _, outcome := range outcomes {
		if outcome.err == nil || errors.Is(outcome.err, domain.ErrAIOutput) {
			return outcomes, nil
		}
		if firstErr == nil {
			firstErr = outcome.err
		}
	}
	return nil, firstErr
}

func (s *Service) extract(ctx context.Context, material domain.CaptureMaterial, system string, screenLabels []string, promptKeys []string, model string) (domain.CaptureExtraction, error) {
	user := fmt.Sprintf("Material %s (%s", material.ID, material.Label())
	if name := strings.TrimSpace(material.Name); name != "" {
		user += ", " + s.masker.Mask(name)
	}
	user += ").\n"
	message := domain.AIMessage{Role: domain.AIRoleUser}
	if material.Kind == domain.CaptureImage {
		message.Images = []domain.AIImage{{MediaType: material.MediaType, Data: material.Data}}
		user += "Read every money movement shown in the image."
	} else {
		user += "Read every money movement in this text:\n```\n" + s.masker.Mask(material.Text) + "\n```"
	}
	today := s.today()
	message.Content = user + "\n\n" + domain.CaptureFormatInstruction(today, screenLabels)
	var extraction domain.CaptureExtraction
	_, err := s.complete(ctx, call{
		feature:  "capture-extract",
		class:    classCapture,
		useCase:  domain.AIUseCaseVision,
		model:    model,
		prompts:  promptKeys,
		messages: []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, message},
		json:     true,
		validate: func(content string) error {
			var err error
			extraction, err = domain.ParseCaptureExtraction(content, material, today)
			return err
		},
	})
	return extraction, err
}
