import SwiftUI

/// Coaching overlays on the camera view: the guided chin tuck panel, the stale-calibration prompt and the
/// movement-break prompt. At most one is shown, in that priority order.
struct CoachOverlayView: View {
    @ObservedObject var appState: AppState
    @ObservedObject var coach: PostureCoach

    init(appState: AppState) {
        self.appState = appState
        self.coach = appState.coach
    }

    var body: some View {
        Group {
            if let session = coach.chinTuckSession {
                ChinTuckPanel(session: session, setsToday: coach.chinTuckSetsToday) {
                    coach.endChinTucks()
                }
            } else if case .stale(let reason) = coach.calibrationHealth, appState.isCalibrated {
                CoachBanner(
                    icon: "scope",
                    tint: .yellow,
                    title: "Calibration May Be Out of Date",
                    subtitle: reason.displayText,
                    primary: ("Recalibrate", { appState.triggerCalibration() }),
                    secondary: ("Keep", { coach.dismissStaleCalibration() })
                )
            } else if coach.isBreakDue {
                CoachBanner(
                    icon: "figure.walk",
                    tint: .green,
                    title: "Time to Move",
                    subtitle: "Seated for \(Int(coach.seatedSeconds / 60)) min. Stand up for a minute or do a chin tuck set.",
                    primary: coach.canStartChinTucks ? ("Chin Tucks", { coach.startChinTucks() }) : nil,
                    secondary: ("Later", { coach.acknowledgeBreak() })
                )
            }
        }
        .frame(maxWidth: 480)
    }
}

private struct CoachBanner: View {
    let icon: String
    let tint: Color
    let title: String
    let subtitle: String
    let primary: (String, () -> Void)?
    let secondary: (String, () -> Void)

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: icon)
                .font(.system(size: 15, weight: .bold))
                .foregroundColor(tint)

            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundColor(.white)
                Text(subtitle)
                    .font(.system(size: 9.5))
                    .foregroundColor(.white.opacity(0.75))
                    .fixedSize(horizontal: false, vertical: true)
            }

            Spacer()

            if let primary {
                Button(action: primary.1) {
                    Text(primary.0)
                        .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                        .foregroundColor(.black)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(tint)
                        .clipShape(Capsule())
                }
                .buttonStyle(.plain)
            }
            Button(action: secondary.1) {
                Text(secondary.0)
                    .font(.system(size: 10.5, weight: .semibold, design: .monospaced))
                    .foregroundColor(.white.opacity(0.8))
            }
            .buttonStyle(.plain)
        }
        .coachPanelStyle(tint: tint)
    }
}

private struct ChinTuckPanel: View {
    let session: ChinTuckSession
    let setsToday: Int
    let onClose: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 8) {
            HStack {
                Text("Chin Tucks")
                    .font(.system(size: 11, weight: .bold, design: .monospaced))
                    .foregroundColor(.white)
                Spacer()
                Text("Rep \(min(session.completedReps + 1, ChinTuckSession.repsPerSet))/\(ChinTuckSession.repsPerSet)")
                    .font(.system(size: 10.5, weight: .semibold, design: .monospaced))
                    .foregroundColor(.cyan)
                    .opacity(session.isComplete ? 0 : 1)
                Text("Sets today \(setsToday)/\(ChinTuckSession.setsPerDay)")
                    .font(.system(size: 10, design: .monospaced))
                    .foregroundColor(.white.opacity(0.6))
            }

            Text(session.cue.text)
                .font(.system(size: 13, weight: .semibold))
                .foregroundColor(cueColor)

            DepthBar(depth: session.depth, holdProgress: session.holdProgress)

            HStack {
                Text("Stop if you feel pain, numbness or dizziness. Not medical advice.")
                    .font(.system(size: 9))
                    .foregroundColor(.white.opacity(0.55))
                Spacer()
                Button(action: onClose) {
                    Text(session.isComplete ? "Done" : "Stop")
                        .font(.system(size: 10.5, weight: .bold, design: .monospaced))
                        .foregroundColor(.black)
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background(session.isComplete ? Color.green : Color.white.opacity(0.8))
                        .clipShape(Capsule())
                }
                .buttonStyle(.plain)
            }
        }
        .coachPanelStyle(tint: .cyan)
    }

    private var cueColor: Color {
        switch session.cue {
        case .keepHeadLevel, .faceCamera: return .orange
        case .holding, .setComplete: return .green
        case .glideBack, .release: return .white
        }
    }
}

/// Retraction depth from habitual neutral (left) to best upright (right), with the hold target marked.
private struct DepthBar: View {
    let depth: Double
    let holdProgress: Double

    var body: some View {
        GeometryReader { geo in
            let width = geo.size.width
            ZStack(alignment: .leading) {
                Capsule().fill(Color.white.opacity(0.12))
                Capsule()
                    .fill(depth >= ChinTuckSession.targetDepth ? Color.green : Color.cyan)
                    .frame(width: width * min(1, max(0, depth)))
                Rectangle()
                    .fill(Color.white)
                    .frame(width: 2)
                    .offset(x: width * ChinTuckSession.targetDepth - 1)
                if holdProgress > 0 {
                    Capsule()
                        .fill(Color.green.opacity(0.5))
                        .frame(width: width * holdProgress, height: 3)
                        .offset(y: 7)
                }
            }
        }
        .frame(height: 10)
    }
}

private extension View {
    func coachPanelStyle(tint: Color) -> some View {
        padding(.horizontal, 14)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 10)
                    .fill(Color(red: 0.10, green: 0.12, blue: 0.16).opacity(0.92))
                    .shadow(color: Color.black.opacity(0.4), radius: 6, x: 0, y: 3)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 10)
                    .stroke(tint.opacity(0.5), lineWidth: 1.0)
            )
    }
}
