# 5 Whys - Troubleshooting Guide

## When You Get Stuck

### "I can't seem to get to the real cause—keep circling back"

**Issue**: Your why chain isn't progressing; answers feel circular.

**Diagnostic questions**:

- Are you asking why about the same thing repeatedly? (Symptom of unclear problem statement)
- Is the answer too vague to question further? ("It's just how it is")
- Are you assuming people are being irrational? (Usually not—there's a reason)

**Fix**:

1. Step back. State the problem more specifically. Include measurements.
2. Ask about a _different aspect_ of the problem.
3. Instead of "why", try "what would need to be true for this NOT to happen?"

**Example**:

- ❌ Why are users confused? "The UI is bad" → Why? "It's not intuitive" → This circles
- ✅ Why are users confused? "They're clicking wrong buttons 60% of the time" → Why? "Button labels don't match their mental model" → Why? "We didn't do user research"

### "I've asked why 3 times and feel like I'm at the root cause—do I need to keep going?"

**Issue**: You're uncertain whether to stop.

**Answer**: If you can't answer "why?" to your current why, and it's actionable, you're probably done. You don't need to reach the literal beginning of the universe—you need an insight that changes what you do.

**Check**:

- Can you name the next why? If yes, keep going. If no or it doesn't make sense, stop.
- Is this actionable? Can your team do something about it? If yes, stop. You have your insight.
- Would fixing this solve the original problem? If unsure, you need more whys.

### "The person I'm interviewing doesn't know the answers"

**Issue**: You're asking someone to speculate about causes they don't actually know.

**Fix**:

1. Name it: "I don't think you know the answer to this. Let me find out differently."
2. Pivot to observable facts: "Let's look at the data." "Let's watch a user try to do this."
3. Interview someone else: Different people know different levels of the chain.

**Example**:

- You to engineer: "Why didn't we use async/await?"
- Engineer: "I... don't know?" (They weren't there or don't remember the decision)
- Better: "Who made that architectural decision? Let me ask them."

### "Every answer points to 'time pressure' or 'resource constraints'"

**Issue**: You're hitting real external constraints at level 2-3. This is fine—these _are_ root causes sometimes.

**Clarify**:

- Is this a permanent constraint or changeable? ("We'll never have more resources" vs. "We're understaffed this quarter")
- Is this acceptable as the root cause, or does it make you uncomfortable? (If uncomfortable, dig sideways)
- What would change if you _didn't_ have the constraint? (This informs the solution)

**Example chain**:

- Why are bugs shipping? → No time to test
- Why no time? → Timeline pressure
- Why timeline pressure? → CEO committed to release date
- Is that the real root cause? Maybe not—the real issue might be "process doesn't build in testing time"

Try a sideways chain: Why do we keep hitting timeline pressure? → We estimate badly → We don't use historical data → No one tracks estimates vs. actuals

Now you have an actionable root cause: "Estimation process doesn't use data."

### "I feel like I'm blaming someone"

**Issue**: Your why chain keeps pointing to "person X made a bad decision."

**Shift**:

- Instead of "Why did they decide that?" ask "Why did that decision make sense to them at the time?"
- Then: "What information or constraint were they working with?"
- Then: "How could we change the system so the right decision is the easy decision?"

**Example**:

- ❌ Why did the designer remove the trust badges? "They thought they were outdated" (blaming their taste)
- ✅ Why did the designer remove the trust badges? "There was no design review process to catch it" (systemic)
- ✅ Why was there no design review? "We prioritized speed over process" (organizational choice)

### "The why chain is too long and complex—I have 8 levels"

**Issue**: Either you're asking about multiple problems, or your problem statement was too broad.

**Fix**:

1. **Separate the chains**: You might have 2-3 different root causes. List them separately.
2. **Narrow the problem**: Instead of "product adoption is low", try "free tier users don't convert" or "enterprise customers churn after 6 months"
3. **Stop at practical stopping points**: You don't need to go all the way to organizational philosophy. Stop at "This is something we can change."

### "We've identified the cause, but fixing it seems impossible"

**Issue**: Root cause is real but feels out of reach.

**Reframe**:

- What's the closest thing to this that IS fixable? (If "needs more budget" is the root cause, what can you control within budget?)
- What's the smallest experiment to validate this? (Don't need to fix the whole thing; just test your hypothesis)
- What would happen if you ignored this and fixed a different level instead? (Sometimes moving up one level is more practical)

**Example**:

- Root cause: "We need to hire 10 more engineers" (hard to control)
- Smallest experiment: "Hire 2 engineers and measure if velocity improves" (testable)
- Alternative: "We're inefficient because we have no code review process" (fixable now)

### "Multiple team members have different why chains for the same problem"

**Issue**: Disagreement on root cause.

**This is good data**:

- Different people see different parts of the system
- You probably have multiple valid root causes
- Prioritize by impact and fixability

**Next step**:

- Map all the why chains on one canvas
- Note where they diverge
- Test the key disagreement point with data if possible
- Pick 1-2 chains to address first (highest impact or easiest to fix)

## What to Do With Your Root Cause

Once you've identified the root cause:

### 1. Validate It (Don't Just Assume)

- **Interview**: Talk to someone who experienced the problem directly
- **Observe**: Watch the problem happen (or don't happen) in context
- **Experiment**: Test your hypothesis with a small change
- **Data**: Check if the timeline aligns (did the cause precede the problem?)

### 2. Pick the Most Actionable Level

If your chain has 5 levels, levels 1-2 are usually symptoms. Levels 3-5 might be actionable. Which one can _you_ actually fix?

```
Problem: Customers hate our onboarding

Why 1: They say it's confusing (symptom)
Why 2: Too many steps (design issue) — You can fix this
Why 3: Feature bloat (product issue) — Would require scope discussion
Why 4: Sales promises everything (org issue) — Needs exec alignment
Why 5: Competitive pressure (market issue) — Can't directly control

Most actionable: Level 2 or 3. Start there. See if it helps the symptom.
```

### 3. Propose an Experiment

Don't try to fix the root cause entirely. Test your hypothesis first:

- "If we reduce onboarding from 8 steps to 4, does satisfaction improve?"
- "If we pair new engineers with a mentor for 2 weeks, do onboarding metrics improve?"
- "If we do user research before design changes, do regressions decrease?"

### 4. Measure Success

- **Before**: What was the problem, measured? (30% abandonment at checkout)
- **Change**: What are you testing? (Adding trust badges back)
- **After**: Measure the same thing (Abandonment rate after change)

If abandonment drops, your root cause hypothesis was probably right.

## Red Flags: When 5 Whys Might Not Be The Right Tool

- **Problem is entirely external** ("Market crashed", "Competitor launched", "Regulation changed") → 5 Whys still works, but you'll hit external constraints faster
- **Problem is complex system behavior** (Load balancer failover cascade, multi-component failure) → Consider using Fault Tree Analysis instead
- **Problem is human behavior** ("Why do users hate our product?") → Supplement with user interviews and observation
- **You need to make a quick decision** ("Server is on fire") → 5 Whys takes 30+ minutes; fix now, analyze later
- **Multiple competing root causes** → Likely your problem statement was too broad; split into smaller problems

## Quick Checklist

Before wrapping up your 5 Whys analysis:

- [ ] Problem statement is specific and measurable
- [ ] Why chain shows progression (each why answers the previous one)
- [ ] No level is "I don't know" or "That's just how it is"
- [ ] You can identify which level is actionable
- [ ] You've considered alternative chains (sideways whys)
- [ ] You've named any assumptions you're making
- [ ] You have a next step or experiment to validate
