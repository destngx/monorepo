# Posturify - Coaching Roadmap

Posturify helps people with forward head posture ("turtle neck") by monitoring head position from the webcam, alerting
on sustained drift, and actively rehabilitating it. This roadmap prioritizes features with a Pareto split: the vital
20% of features expected to deliver ~80% of the real-world effect, and the remaining 80% that refine it.

The real-world effect is measured by two things:

1. **Posture actually improves** (the body changes, not just the reading).
2. **The user keeps the app on** (a disabled app helps nobody; alert fatigue is the main uninstall reason).

## Baseline (done)

| Feature                       | Notes                                                                                                                   |
| ----------------------------- | ----------------------------------------------------------------------------------------------------------------------- |
| Relative CVA proxy            | Calibrated forward vector, face lengths, quality gate, face/body fusion                                                 |
| Posture profiles              | Good / Slight / More turtle neck: neutral CVA and alert strictness                                                      |
| Best-upright calibration step | Captures the correction range (habitual neutral to chin-tucked upright)                                                 |
| Progressive rehab target      | Alerts against a target that moves 15% of the range per week, capped at 75%, with adherence-based advance and step-back |

## Group A - The vital 20% (~80% of the effect)

Each feature is independent of the history store, so all ship without it.

**Status: implemented** (A1-A4). Pure state machines live in `Sources/Coaching` (`AlertEscalator`, `AlertSilence`,
`BreakMonitor`, `ChinTuckSession`) and `Sources/Biomechanics/CalibrationHealth.swift`; `PostureCoach` combines them on
the main actor. Tests: `Tests/KinematicsTests/CoachingTests.swift`.
Alert channels (notifications, movement reminders, edge glow) are configured in the menu bar panel as switches whose
options show only while on; all preferences persist as one `AlertSettings` value.

| #   | Feature                                               | Why it carries the effect                                                                                                                                                                    |
| --- | ----------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| A1  | **Escalating ambient alerts + context-aware silence** | Protects retention. Most drifts are fixed by a glance cue; notifications and sound are reserved for drift that persists. Meetings and presentations are never interrupted.                   |
| A2  | **Guided chin tucks with live camera feedback**       | The only feature that changes the body: deep cervical flexor training has the best evidence for forward head posture. Reuses the calibrated correction range to verify depth, hold and form. |
| A3  | **Movement-break prompts**                            | Prolonged static sitting loads the neck regardless of posture quality. Cheap, evidence-backed, and the natural slot for A2.                                                                  |
| A4  | **Calibration staleness detection**                   | Every alert depends on the baseline. A moved camera or chair silently produces wrong alerts and destroys trust.                                                                              |

### A1 - Escalating alerts and context-aware silence

Escalation ladder per drift episode (an episode starts when the alert level leaves 0 and ends on recovery):

| Stage   | Trigger                                                             | Channel                                    |
| ------- | ------------------------------------------------------------------- | ------------------------------------------ |
| Ambient | Alert level >= 1 (already hold-time filtered)                       | Menu bar icon tint + soft screen-edge glow |
| Notify  | Mild held 30 s, severe held 10 s, sustained overextension held 15 s | Silent notification                        |
| Sound   | Severe held 60 s after a notification in the same episode           | Notification sound (once per episode)      |

- Notifications repeat at most once per profile cooldown; sound at most once per episode.
- **Silence contexts**: microphone in use (calls), camera in use by another app (video meetings), a full-screen app in
  front (presentations, video), manual snooze (15 min / 1 h / until tomorrow), and guided exercise in progress.
  While silenced, the screen-edge glow and notifications are suppressed; the menu bar tint remains (private to the user).
- Sound is only played through the notification system so macOS Focus modes suppress it (Focus status itself needs a
  restricted entitlement, so it is not read directly).

### A2 - Guided chin tucks

- Available once calibration captured a correction range (the best-upright step).
- Rep cycle: glide back to at least 60% of the personal correction range, hold 5 s, release to within 25% of the
  range from neutral. 10 reps per set, 3 sets per day suggested.
- Form check: head pitch must stay within 10 degrees of neutral (gliding back, not nodding down). A broken hold restarts.
- Alerts are silenced and adherence is not counted during a set; alert state restarts afterwards.
- A completed set counts as a movement break (A3).
- Safety note shown in the exercise view: stop on pain, numbness or dizziness; not medical advice.

### A3 - Movement-break prompts

- Seated time accumulates while a face is present; absence for 2 min (or no frames for 2 min) ends the seated session.
- Prompt after 45 min seated (configurable 30 / 45 / 60), repeated every 10 min while still seated.
- Prompts are deferred while silenced (A1 contexts) and delivered once the context clears.
- Completing a chin tuck set resets seated time.

### A4 - Calibration staleness detection

Evaluated on frames with a detected face (absence is never evidence):

| Signal                                                                                       | Stale when                                    |
| -------------------------------------------------------------------------------------------- | --------------------------------------------- |
| Position shift: displacement orthogonal to the forward axis (cannot be forward head posture) | 60 s average above 0.3 face lengths for 3 min |
| Unreliable tracking: quality gate failing                                                    | 60 s average failure rate above 70% for 3 min |
| Camera geometry changed: image aspect differs from calibration                               | Persistently for 10 s                         |

- Stale state is sticky until recalibration or dismissal, shown as a banner with a recalibrate action and one notification.

## Group B - The remaining 80% (~20% of the effect)

| Feature                                                              | Why lower                                                | Depends on |
| -------------------------------------------------------------------- | -------------------------------------------------------- | ---------- |
| Posture history store (per-minute aggregates, SQLite)                | Enabler with no direct effect                            | -          |
| Alert-effectiveness measurement (correction within 10 s of an alert) | Tunes A1                                                 | History    |
| Alert budget (max alerts per hour)                                   | Largely covered by A1's ambient stage and cooldowns      | -          |
| Monthly best-upright reassessment                                    | Proves progress, does not cause it; nearly free after A2 | A2         |
| Data-driven profile suggestions                                      | Convenience; manual switching exists                     | History    |
| Daily summary with 1-3 tips                                          | Reflection changes behavior less than live cues          | History    |
| Positive feedback and streaks                                        | Motivation boost                                         | History    |
| Fatigue-drift detection                                              | Refines break timing that A3 approximates                | History    |
| Ergonomic setup tips (screen height, distance)                       | Mostly a one-time fix                                    | History    |
| Objective CVA capture (side-profile photo)                           | Accuracy; relative tracking already works                | -          |

## Sequencing

1. **A1 + A4** - make alerts trustworthy and bearable (protects retention).
2. **A2 + A3** - active rehab and movement.
3. **Group B** - history store first, then the features that depend on it.
