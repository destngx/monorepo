# 5 Whys - Detailed Guide

## How to Establish the Problem (Phase 1)

Get a clear, specific problem statement. Vague problems lead to vague answers. The problem should be:

- **Specific and observable** - "Customers abandon carts on the payment page" (not "Sales are low")
- **Measurable where possible** - "30% of users click back before entering payment info"
- **Free of proposed solutions** - "Customers struggle with the checkout flow" (not "We need a better UI")

Bad problem statements:

- ❌ "Our app is slow"
- ❌ "Team morale is low"
- ❌ "We're losing customers"

Good problem statements:

- ✅ "Mean page load time increased from 1.2s to 2.8s after the redesign"
- ✅ "Three senior engineers left in the past month"
- ✅ "Churn rate jumped from 3% to 8% for the enterprise segment"

## Asking Why Five Times (Phase 2)

### The Process for Each Why

1. **Record the answer** - Write down the explanation you get, exactly as stated
2. **Dig deeper** - Ask why that answer is true (don't just assume)
3. **Distinguish symptoms from causes** - Is this a surface behavior or a fundamental reason?
4. **Challenge gently** - "And why does that happen?" keeps momentum
5. **Track the chain** - Each why should logically follow from the previous answer

### Complete Example: Checkout Abandonment

```
Problem: 30% of users abandon at the payment page

Why 1: Why do customers abandon at payment?
Answer: The form feels unsafe

Why 2: Why does it feel unsafe?
Answer: There's no trust signal or security badge visible

Why 3: Why is there no trust signal?
Answer: We removed the visual indicators during the redesign

Why 4: Why did you remove them?
Answer: The designer thought they looked outdated

Why 5: Why didn't the team validate this change with users?
Answer: User testing was skipped due to timeline pressure
```

**Root insight**: Not a problem with checkout UX, but a process problem: design decisions weren't validated.

## Synthesizing Findings (Phase 3)

After 5 whys, you might not have reached the ultimate root cause—and that's OK. Often you've hit practical stopping points.

**For each "why" level, ask yourself:**

- Is this actionable? Can we do something about it?
- Is this a pattern we've seen before?
- Does this connect to other problems we're facing?
- What would change if we fixed this?

### Example Synthesis from Checkout Case

| Level | Why                     | Actionable?        | Insight                                  |
| ----- | ----------------------- | ------------------ | ---------------------------------------- |
| 1     | Feels unsafe            | No (symptom)       | Points to level 2                        |
| 2     | No trust signal         | Partially (design) | Could improve visual design              |
| 3     | Removed during redesign | Yes (process)      | Design changes need validation           |
| 4     | Designer judgment       | Yes (process)      | Decisions shouldn't be made in isolation |
| 5     | Timeline pressure       | Yes (system)       | Process encourages shortcuts             |

**Most actionable levels**: 4 and 5 (process and system issues)

## Tips for Better Whys

### Avoid Blame

❌ "The person didn't validate" → ✅ "Our process doesn't include validation"

❌ "They made a bad decision" → ✅ "We didn't have a decision framework"

Focus on systems and processes, not individuals. The person was working within constraints.

### Stay Curious

Each answer should genuinely lead to the next question. If it feels forced, you might have the wrong answer.

- ❌ "Why does X happen?" "I don't know" → Stop here, go sideways
- ✅ "Why does X happen?" "Because of Y" → "Why Y?" makes natural sense

### Distinguish Levels

Different kinds of causes:

1. **Symptoms** - Observable behaviors
2. **Immediate causes** - What directly caused it (changed code, human action)
3. **Systemic causes** - Process or structural issues (no review, no testing)
4. **Foundational causes** - Organizational or market-level causes (no resources, priorities)

A good why chain moves from symptoms through systemic to foundational.

### Go Sideways Sometimes

If one why thread seems exhausted, explore an alternative explanation for the same problem:

```
Problem: High user churn

Why chain 1: Price too high → Product not worth price → Features incomplete → Backlog prioritized wrong
Why chain 2: Product crashes → Error handling poor → Team too small to fix → Resource constraint
Why chain 3: Confusing UX → No user research → Timeline pressure → Org prioritizes speed over quality
```

Three different root causes! Now you understand the real picture.

### Use Data When You Have It

- ✅ "Traffic dropped 40% when we deployed version X"
- ✅ "Conversion rate fell from 8% to 5% after we removed the trust badge"
- ❌ "I think something broke"
- ❌ "Users probably don't like it"

Data grounds your why chain in reality.

### Name Assumptions

If you're guessing at why something happened, say so:

- "I'm assuming the designer wasn't aware of the security requirement..."
- "Best guess: the API change caused this, but I should verify..."
- "We haven't actually asked customers why they churn..."

Naming assumptions creates opportunities to test them.

## When to Stop

You don't always need to complete all 5 whys. Stop when:

- **You reach an external constraint** - "We can't change the payment processor's requirements"
- **You hit a decision made by leadership/customers/the market** - "The CEO decided to compete on price"
- **You've identified a clear, actionable insight** - "Our process doesn't validate design changes; let's fix that"
- **The chain stops making logical sense** - Indicates you've gone off track; backtrack and try a different why

You can also go _beyond_ 5 if a chain is still productive. Most problems stabilize between 3-4 levels deep. After that, you hit diminishing returns or external constraints.

## Output Format

```
## Problem Statement
[Clear, specific description with measurements if possible]

## Why Chain
**Why 1**: [Answer] → Why does that happen?
**Why 2**: [Answer] → Why does that happen?
**Why 3**: [Answer] → Why does that happen?
**Why 4**: [Answer] → Why does that happen?
**Why 5**: [Answer] → This is a practical stopping point because...

## Pattern Recognition
- [Insight 1 - themes that emerged]
- [Insight 2 - systemic issues]
- [Insight 3 - external factors]

## Most Actionable Level
[Which why is actually solvable with available resources?]

## Recommended Next Steps
1. [Specific experiment or investigation to validate this cause]
2. [Test or measurement to confirm]
3. [Action if confirmed]
```

## Common Pitfalls

### Stopping Too Early

❌ Problem: "Sales are down" → Why? "Market downturn" → Stop

✅ Problem: "Sales down 15% while competitors up 5%" → Why? [dig into competitive response, pricing, product fit]

### Accepting Surface Answers

❌ Why does the system crash? "It's buggy" → That's not an answer, it's a restatement

✅ Why does the system crash? "Memory leak in the notification handler when processing >1000 events/sec" → That's a root cause

### Confusing Correlation with Causation

❌ "Sales dropped after we launched feature X, so feature X caused it" → Possible, but confirm

✅ "Sales dropped after launch. Feature X increased page load time from 1s to 3s. Bounce rate increased 40%. Hypothesis: feature X caused churn."

### Staying at the Symptom Level

❌ Just knowing "customers don't like the UI" doesn't help you fix it

✅ "Customers abandon because trust badges were removed → removed during redesign → designer decision not validated → process doesn't include validation"

## When 5 Whys Isn't Enough

Complex problems sometimes need multiple parallel chains:

```
Root Cause: High production bugs

Chain 1: → No automated testing → Resource constraint
Chain 2: → Deployment process doesn't validate → Process issue
Chain 3: → Team pressure to ship faster → Organizational priority
Chain 4: → Engineers burned out → Retention risk

All four are real. Fixing only one won't solve the problem.
```

In cases like this, map all chains and prioritize which to fix first.
