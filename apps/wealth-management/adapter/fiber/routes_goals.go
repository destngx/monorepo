package fiber

// Goal Plan routes (D31, D32). Writes go through the WriteGuard and touch
// only the app-managed Goal Plan tab.

const goalPlanSummary = "The goal plan. Goals are claims on one pooled savings balance, filled by priority " +
	"(emergency fund first; the same priority shares equally). pool = assets (positive balances of ready accounts; " +
	"crypto, archived and slow accounts excluded) - cardDebt - " +
	"spendingBuffer (bufferMonths x average monthly expense of the last 3 completed months), floored at 0; crypto, " +
	"archived and moneyLent are shown but not counted. slow = investment and long holding accounts and the accounts " +
	"marked not ready to use; only goals with useSlowMoney draw on it, before ready money. needs = average fixed + essential expense of the last 6 " +
	"completed months (fromTotalExpense when no category is tagged); an emergency fund with a blank target gets " +
	"emergencyMonths x needs. monthlySavings = average income - expense of the last 3 completed months, floored at 0. " +
	"Per goal: targetAmount, funded (startingAmount + fromPool), fromSlow (the slow money part of fromPool), remaining, percent, monthsLeft, requiredPerMonth, " +
	"projectedFinish (yyyy-mm, savings added in priority order; null when not reachable within 50 years), " +
	"monthsToReach, yearsToReach, longHorizon and pace (done, paused, funded, on_track, behind, no_deadline). " +
	"check: requiredPerMonth of dated goals vs monthlySavings, gap (> 0 is a shortfall), verdict (fits, short, " +
	"no_deadlines) and tradeOffs (lowest priority first: moveTargetDateTo or lowerTargetBy that alone close the gap). " +
	"setupNeeded is true until the Goal Plan tab exists; legacyGoals counts old Goals rows a setup would import. " +
	"issues lists rejected rows."

func goalProperties() map[string]any {
	return map[string]any{
		"name":           schemaString,
		"target":         map[string]any{"type": "number", "description": "VND, positive. Omit only on the emergency fund (name or note mentions emergency) to compute it."},
		"targetDate":     map[string]any{"type": "string", "description": "yyyy-mm; omit for no deadline."},
		"priority":       map[string]any{"type": "integer", "description": "1 is filled first; the same priority shares equally. 0 or omitted: last on create, unchanged on update."},
		"startingAmount": map[string]any{"type": "number", "description": "Money already set aside outside the pool."},
		"status":         map[string]any{"type": "string", "enum": []string{"active", "paused", "done"}, "description": "Default active on create, unchanged on update."},
		"note":           schemaString,
		"useSlowMoney":   map[string]any{"type": "boolean", "description": "Let the goal draw on slow money (investment and long holding accounts, accounts marked not ready to use). Default false; an update without it turns it off."},
	}
}

func registerGoalWriteRoutes(goals RouteRegistrar, h *DatabaseHandler, guard WriteGuard) {
	goals.Post("/setup", guard.Wrap(h.SetupGoalPlan), RouteMeta{
		Summary: "First use: creates the Goal Plan tab with an emergency fund row (priority 1, computed target) and the old Goals rows (name and target, in sheet order, active). The old Goals tab is only read. Returns {created, imported, skipped}; 409 when the Goal Plan tab already has goals.",
	})
	goals.Post("/", guard.Wrap(h.CreateGoal), RouteMeta{
		Summary:     "Adds a goal row to the Goal Plan tab and returns it with its id and etag. 404 before setup.",
		RequestBody: jsonBody(goalProperties(), "name"),
	})
	updateProperties := goalProperties()
	updateProperties["etag"] = map[string]any{"type": "string", "description": "ETag from the goal plan; may be sent as If-Match instead."}
	goals.Patch("/:id", guard.Wrap(h.UpdateGoal), RouteMeta{
		Summary:     "Replaces a goal row (name, target, target date, priority, starting amount, status, note); pause, resume and mark done by status. Returns 409 when the row changed since it was read.",
		PathParams:  []RouteParameter{PathParam("id", "string", "Goal id, goal-N.")},
		RequestBody: jsonBody(updateProperties, "name"),
	})
	goals.Delete("/:id", guard.Wrap(h.DeleteGoal), RouteMeta{
		Summary:     "Clears a goal row. Requires the row's ETag via If-Match or ?etag=; returns 409 when it changed.",
		PathParams:  []RouteParameter{PathParam("id", "string", "Goal id, goal-N.")},
		QueryParams: []RouteParameter{QueryParam("etag", "string", "Row ETag, if not sent as If-Match.")},
	})
	goals.Post("/reorder", guard.Wrap(h.ReorderGoals), RouteMeta{
		Summary:     "Sets priorities 1..n in the order of ids; goals not listed follow in their current order, and an id that is not in the plan is a 409. Returns the goals in the new order with fresh etags.",
		RequestBody: jsonBody(map[string]any{"ids": map[string]any{"type": "array", "items": schemaString}}, "ids"),
	})
}
