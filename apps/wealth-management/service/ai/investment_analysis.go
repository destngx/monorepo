package ai

import (
	"apps/wealth-management/domain"
	"apps/wealth-management/port"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// thinkTankPhase is one step of the investment analysis with its legacy
// prompt keys (WM-AI-9). The fallback task is optional and smaller; its
// action falls back to the main action.
type thinkTankPhase struct {
	name                         string
	title                        string
	taskKey, actionKey           string
	fallbackTask, fallbackAction string
}

var (
	phaseThinkTank = thinkTankPhase{"think-tank", "Think tank", "market/think-tank", "actions/investment-think-tank", "market/fallback-think-tank", "actions/investment-think-tank-fallback"}
	phaseSynthesis = thinkTankPhase{"synthesis", "Synthesis", "market/synthesis", "actions/investment-synthesis", "market/fallback-synthesis", "actions/investment-synthesis-fallback"}
	phaseActions   = thinkTankPhase{"actions", "Actions", "market/action", "actions/investment-action", "market/fallback-action", "actions/investment-action-fallback"}
)

// thinkTankInputs is the raw context of an investment analysis; it is sent
// to the UI as-is for disclosure (WM-INV-18).
type thinkTankInputs struct {
	Valuation          *domain.PortfolioValuation `json:"valuation,omitempty"`
	ValuationError     string                     `json:"valuationError,omitempty"`
	FundPerformance    *fundPerformanceFacts      `json:"fundPerformance,omitempty"`
	FundPerformanceErr string                     `json:"fundPerformanceError,omitempty"`
	Exposure           *domain.ExposureFacts      `json:"exposure,omitempty"`
	ExposureErr        string                     `json:"exposureError,omitempty"`
	Accounts           []accountContext           `json:"accounts"`
	MoneyLent          moneyLentContext           `json:"moneyLent"`
	RecentTransactions []txContext                `json:"recentTransactions"`
	Budget             budgetLimits               `json:"budget"`
	USDTRate           *domain.ExchangeRate       `json:"usdtRate,omitempty"`
	Searches           []domain.SearchResponse    `json:"searches"`
	SearchErrors       []string                   `json:"searchErrors,omitempty"`
	SearchContext      string                     `json:"searchContext"`
	Pulse              *domain.MarketPulse        `json:"pulse,omitempty"`
	PulseError         string                     `json:"pulseError,omitempty"`
	News               []domain.NewsReport        `json:"news,omitempty"`
	// Focus is what the run analyses: the requested symbols or the holdings.
	Focus domain.ThinkTankFocus `json:"focus"`
}

// budgetLimits are this month's planned income and expenses (legacy input).
type budgetLimits struct {
	Income   float64 `json:"income"`
	Expenses float64 `json:"expenses"`
}

// InvestmentAnalysis streams context → think tank → synthesis → commands.
// Each phase retries with its fallback prompt pair when the main one fails
// (WM-AI-9). Errors after the stream started are sent as an error event.
func (s *Service) InvestmentAnalysis(ctx context.Context, request port.InvestmentAnalysisRequest, emit func(port.InvestmentEvent) error) error {
	if err := errors.Join(s.requireConfigured(), s.requireMarket()); err != nil {
		return err
	}
	if s.portfolio == nil {
		return fmt.Errorf("%w: portfolio valuation is not configured", domain.ErrAIUnavailable)
	}
	symbols, err := domain.NormalizeFocusSymbols(request.Symbols)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, thinkTankTimeout)
	defer cancel()
	snap, err := s.loadSnapshot()
	if err != nil {
		return err
	}
	p, err := s.loadPrompts()
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: "context", Status: "running"}); err != nil {
		return err
	}
	inputs := s.thinkTankInputs(ctx, snap, request.News, symbols)
	if err := emit(port.InvestmentEvent{Type: "context", Context: inputs}); err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: "context", Status: "ok"}); err != nil {
		return err
	}
	vars := s.thinkTankVars(snap, inputs)
	focus := inputs.Focus.PromptBlock()

	debate, model, err := s.runPhase(ctx, p, snap, phaseThinkTank, vars, map[string]string{"totalVND": vars["totalVND"], "totalCrypto": vars["totalCrypto"]}, request.Model, focus, emit)
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "message", Phase: phaseThinkTank.name, Name: phaseThinkTank.title, Content: debate, Model: model}); err != nil {
		return err
	}

	synthesisVars := pick(vars, "totalVND", "totalCrypto", "income", "expenses", "p2pRate", "accountsSummary", "moneyLent", "recentTransactions")
	synthesisVars["expertDebateContext"] = debate
	synthesis, model, err := s.runPhase(ctx, p, snap, phaseSynthesis, synthesisVars, map[string]string{"accountsSummary": vars["accountsSummary"], "debate": debate}, request.Model, focus, emit)
	if err != nil {
		return err
	}
	if err := emit(port.InvestmentEvent{Type: "message", Phase: phaseSynthesis.name, Name: phaseSynthesis.title, Content: synthesis, Model: model}); err != nil {
		return err
	}

	actionVars := pick(vars, "accountsSummary", "cryptoHoldings", "ifcHoldings")
	actionVars["synthesisContext"] = synthesis
	review, previousErr := s.thinkTankReview(ctx)
	contract := focus + "\n\n" + commandsFormatInstruction
	if review.source != "" {
		contract += "\n\n" + domain.RecommendationFormatInstruction + domain.PreviousRecommendationsBlock(review.previous, s.masker.Mask)
	}
	var commands []domain.InvestmentCommand
	var raw domain.InvestmentCommands
	parse := func(content string) error {
		raw = domain.InvestmentCommands{}
		if err := domain.ParseAIJSON(content, &raw); err != nil {
			return err
		}
		normalized, err := raw.Normalize()
		commands = normalized
		return err
	}
	_, model, err = s.runPhaseWith(ctx, p, snap, phaseActions, actionVars, map[string]string{"synthesis": synthesis}, request.Model, contract, true, parse, emit)
	if err != nil {
		return err
	}
	recorded := previousErr
	if recorded == nil {
		recorded = s.record(ctx, review, raw.Recommendations, raw.Outcomes)
	}
	if err := emit(port.InvestmentEvent{Type: "actions", Phase: phaseActions.name, Name: phaseActions.title, Commands: commands, Model: model, Recorded: recorded}); err != nil {
		return err
	}
	return emit(port.InvestmentEvent{Type: "done"})
}

// thinkTankReview is the think tank's place in the recommendations list
// (WM-REC-8): its source, page and earlier items. When the earlier items
// cannot be read the run goes on without the contract, and the returned
// record says why nothing was recorded.
func (s *Service) thinkTankReview(ctx context.Context) (structuredRequest, *domain.RecommendationRecord) {
	request, err := s.withPrevious(ctx, structuredRequest{feature: "think-tank", source: domain.RecommendationSourceThinkTank, page: domain.ThinkTankPage})
	if err != nil {
		s.log.LogError(ctx, "loading earlier recommendations failed", err, slog.String("feature", request.feature))
		request.source = ""
		return request, &domain.RecommendationRecord{Created: []domain.Recommendation{}, Checked: []domain.Recommendation{}, Error: "The recommendations could not be saved: " + err.Error()}
	}
	return request, nil
}

func (s *Service) runPhase(ctx context.Context, p *prompts, snap snapshot, phase thinkTankPhase, vars map[string]string, fallbackVars map[string]string, override string, suffix string, emit func(port.InvestmentEvent) error) (string, string, error) {
	return s.runPhaseWith(ctx, p, snap, phase, vars, fallbackVars, override, suffix, false, func(content string) error {
		if strings.TrimSpace(content) == "" {
			return fmt.Errorf("%w: %s reply is empty", domain.ErrAIOutput, phase.name)
		}
		return nil
	}, emit)
}

// runPhaseWith runs the main prompt pair (with the model fallback) and, if
// that fails, the fallback pair. suffix is engine text appended to the
// action prompt (the focus, and the output contract of a JSON phase). It
// returns the reply and the model used.
func (s *Service) runPhaseWith(ctx context.Context, p *prompts, snap snapshot, phase thinkTankPhase, vars map[string]string, fallbackVars map[string]string, override string, suffix string, json bool, validate func(string) error, emit func(port.InvestmentEvent) error) (string, string, error) {
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "running"}); err != nil {
		return "", "", err
	}
	attempt := func(taskKey, actionKey string, vars map[string]string) (domain.AIResponse, error) {
		task, err := p.render(taskKey, vars)
		if err != nil {
			return domain.AIResponse{}, err
		}
		action, err := p.render(actionKey, nil)
		if err != nil {
			return domain.AIResponse{}, err
		}
		action += suffix
		system, err := s.systemPrompt(p, snap, task)
		if err != nil {
			return domain.AIResponse{}, err
		}
		return s.complete(ctx, call{
			feature:  "think-tank:" + phase.name,
			class:    classReview,
			useCase:  domain.AIUseCaseInsight,
			model:    override,
			prompts:  []string{keyIdentity, taskKey, actionKey},
			messages: []domain.AIMessage{{Role: domain.AIRoleSystem, Content: system}, {Role: domain.AIRoleUser, Content: action}},
			json:     json,
			validate: validate,
		})
	}
	response, err := attempt(phase.taskKey, phase.actionKey, vars)
	if err != nil && ctx.Err() == nil {
		if _, ok := p.lookup(phase.fallbackTask, nil); ok {
			if emitErr := emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "fallback", Message: phaseMessage(err)}); emitErr != nil {
				return "", "", emitErr
			}
			actionKey := phase.fallbackAction
			if _, ok := p.lookup(actionKey, nil); !ok {
				actionKey = phase.actionKey
			}
			if fallback, fallbackErr := attempt(phase.fallbackTask, actionKey, fallbackVars); fallbackErr == nil {
				response, err = fallback, nil
			} else {
				err = errors.Join(err, fallbackErr)
			}
		}
	}
	if err != nil {
		_ = emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "error", Message: phaseMessage(err)})
		return "", "", fmt.Errorf("%s phase: %w", phase.name, err)
	}
	if err := emit(port.InvestmentEvent{Type: "phase", Phase: phase.name, Status: "ok"}); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(response.Content), firstNonEmpty(response.Model, s.models(domain.AIUseCaseInsight, override)[0]), nil
}

func phaseMessage(err error) string {
	if IsUserFacing(err) {
		return err.Error()
	}
	return "the model call failed"
}
