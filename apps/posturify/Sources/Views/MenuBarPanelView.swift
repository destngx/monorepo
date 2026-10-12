import SwiftUI
import AppKit

/// Menu bar dropdown panel: posture status, calibration and profile, alert channels and exercise.
///
/// Each alert channel is a switch; its own options appear beneath it only while it is on.
struct MenuBarPanelView: View {
    @ObservedObject var appState: AppState
    @ObservedObject var coach: PostureCoach

    init(appState: AppState) {
        self.appState = appState
        self.coach = appState.coach
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            header
            Divider()
            postureSection
            Divider()
            alertsSection
            Divider()
            exerciseSection
            Divider()
            footer
        }
        .padding(14)
        .frame(width: 310)
        .animation(.easeInOut(duration: 0.18), value: coach.settings)
        .animation(.easeInOut(duration: 0.18), value: coach.snoozedUntil)
    }

    // MARK: - Header

    private var header: some View {
        HStack(spacing: 8) {
            Circle()
                .fill(statusColor)
                .frame(width: 9, height: 9)
                .shadow(color: statusColor.opacity(0.8), radius: 4)
            Text("Posturify")
                .font(.system(size: 13, weight: .bold))
            Spacer()
            Text(statusText)
                .font(.system(size: 11, weight: .medium))
                .foregroundStyle(.secondary)
        }
    }

    private var statusText: String {
        guard appState.isCalibrated else { return "Not calibrated" }
        return appState.kinematics.posture.status.rawValue
    }

    private var statusColor: Color {
        if let alert = coach.ambientAlert {
            return EdgeGlowController.color(coach.settings.edgeGlow.palette, alert)
        }
        return appState.isCalibrated ? .green : .gray
    }

    // MARK: - Posture

    private var postureSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionTitle("Posture")
            HStack {
                Text("Profile")
                Spacer()
                Picker("Profile", selection: $appState.postureProfile) {
                    ForEach(PostureProfile.allCases) { profile in
                        Text(profile.displayName).tag(profile)
                    }
                }
                .labelsHidden()
                .fixedSize()
            }
            if appState.postureProfile.isRehabEligible {
                HStack {
                    Text(rehabStatusText)
                        .font(.system(size: 11))
                        .foregroundStyle(.secondary)
                    Spacer()
                    Button("Reset") { appState.resetRehabProgress() }
                        .controlSize(.small)
                        .disabled(appState.rehabProgram.stage == 0)
                }
            }
            Button(appState.isCalibrated ? "Recalibrate" : "Calibrate Posture") {
                if appState.isCalibrated {
                    appState.resetCalibration()
                } else {
                    appState.triggerCalibration()
                }
                openMainWindow()
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var rehabStatusText: String {
        guard appState.isCalibrated else { return "Rehab target: calibrate first" }
        guard appState.correctionRange > 0 else { return "Rehab target: recalibrate with the chin-tuck step" }
        return "Rehab target: stage \(appState.rehabProgram.stage)/\(RehabProgram.maxStage)"
    }

    // MARK: - Alerts

    private var alertsSection: some View {
        VStack(alignment: .leading, spacing: 10) {
            SectionTitle("Alerts")
            if let reason = coach.silenceReason {
                Label(reason.displayText, systemImage: "moon.fill")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
            }

            SwitchRow(title: "Notifications", icon: "bell.badge", isOn: $coach.settings.notificationsEnabled) {
                OptionRow("Notify after") {
                    Picker("Notify after", selection: $coach.settings.notifyAfter) {
                        ForEach(NotifyDelay.allCases) { Text($0.shortName).tag($0) }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                    .frame(width: 150)
                }
                OptionRow("Sound when severe") {
                    Toggle("Sound when severe", isOn: $coach.settings.severeSoundEnabled)
                        .toggleStyle(.switch)
                        .controlSize(.mini)
                        .labelsHidden()
                }
            }

            SwitchRow(title: "Movement reminders", icon: "figure.walk", isOn: $coach.settings.breakRemindersEnabled) {
                OptionRow("Every") {
                    Picker("Every", selection: $coach.settings.breakInterval) {
                        ForEach(BreakMonitor.intervalOptions, id: \.self) { Text("\(Int($0 / 60))m").tag($0) }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                    .frame(width: 150)
                }
            }

            SwitchRow(title: "Screen edge glow", icon: "rectangle.dashed", isOn: $coach.settings.edgeGlowEnabled) {
                OptionRow("Intensity") {
                    Picker("Intensity", selection: $coach.settings.edgeGlow.intensity) {
                        ForEach(GlowIntensity.allCases) { Text($0.displayName).tag($0) }
                    }
                    .pickerStyle(.segmented)
                    .labelsHidden()
                    .frame(width: 180)
                }
                OptionRow("Color scheme") {
                    PaletteSwatches(selection: $coach.settings.edgeGlow.palette)
                }
                OptionRow("Hint label") {
                    Toggle("Hint label", isOn: $coach.settings.edgeGlow.showsHint)
                        .toggleStyle(.switch)
                        .controlSize(.mini)
                        .labelsHidden()
                }
                HStack {
                    Spacer()
                    Button(coach.isPreviewingGlow ? "Previewing..." : "Preview Glow") { coach.previewGlow() }
                        .controlSize(.small)
                        .disabled(coach.isPreviewingGlow)
                }
            }

            snoozeRow
        }
    }

    private var snoozeRow: some View {
        HStack(spacing: 6) {
            Image(systemName: "zzz")
                .frame(width: 18)
                .foregroundStyle(.secondary)
            if let until = coach.snoozedUntil {
                Text("Snoozed until \(until.formatted(date: .omitted, time: .shortened))")
                Spacer()
                Button("Resume") { coach.cancelSnooze() }
                    .controlSize(.small)
            } else {
                Text("Snooze")
                Spacer()
                ForEach(SnoozeOption.allCases) { option in
                    Button(option.shortName) { coach.snooze(option) }
                        .controlSize(.small)
                        .help(option.displayName)
                }
            }
        }
    }

    // MARK: - Exercise

    private var exerciseSection: some View {
        VStack(alignment: .leading, spacing: 8) {
            SectionTitle("Exercise")
            HStack {
                Button("Start Chin Tucks") {
                    coach.startChinTucks()
                    openMainWindow()
                }
                .disabled(!coach.canStartChinTucks || coach.isExercising)
                Spacer()
                Text("\(coach.chinTuckSetsToday)/\(ChinTuckSession.setsPerDay) sets today")
                    .font(.system(size: 11))
                    .foregroundStyle(.secondary)
            }
            if !coach.canStartChinTucks {
                Text("Calibrate with the chin-tuck step to unlock guided sets.")
                    .font(.system(size: 10.5))
                    .foregroundStyle(.secondary)
            }
        }
    }

    // MARK: - Footer

    private var footer: some View {
        HStack {
            Button("Open Posturify") { openMainWindow() }
                .keyboardShortcut("o")
            Spacer()
            Button("Quit") { NSApp.terminate(nil) }
                .keyboardShortcut("q")
        }
    }

    private func openMainWindow() {
        // The menu bar panel does not close itself when an action opens another window
        if let panel = NSApp.keyWindow as? NSPanel {
            panel.close()
        }
        WindowManager.shared.showWindow(appState: appState)
    }
}

// MARK: - Components

private struct SectionTitle: View {
    let title: String

    init(_ title: String) {
        self.title = title
    }

    var body: some View {
        Text(title.uppercased())
            .font(.system(size: 10, weight: .semibold))
            .foregroundStyle(.secondary)
    }
}

/// A channel switch whose options are revealed beneath it while on.
private struct SwitchRow<Options: View>: View {
    let title: String
    let icon: String
    @Binding var isOn: Bool
    @ViewBuilder let options: () -> Options

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(spacing: 6) {
                Image(systemName: icon)
                    .frame(width: 18)
                    .foregroundStyle(isOn ? Color.accentColor : .secondary)
                Toggle(isOn: $isOn) {
                    Text(title).font(.system(size: 13, weight: .medium))
                }
                .toggleStyle(.switch)
                .controlSize(.small)
            }
            if isOn {
                VStack(alignment: .leading, spacing: 6) {
                    options()
                }
                .padding(.leading, 24)
                .padding(.vertical, 2)
                .transition(.opacity.combined(with: .move(edge: .top)))
            }
        }
    }
}

private struct OptionRow<Control: View>: View {
    let label: String
    @ViewBuilder let control: () -> Control

    init(_ label: String, @ViewBuilder control: @escaping () -> Control) {
        self.label = label
        self.control = control
    }

    var body: some View {
        HStack {
            Text(label)
                .font(.system(size: 11))
                .foregroundStyle(.secondary)
            Spacer()
            control()
        }
        .controlSize(.small)
    }
}

/// Palette picker: each swatch shows the mild, severe and tilt colors of the palette.
private struct PaletteSwatches: View {
    @Binding var selection: GlowPalette

    var body: some View {
        HStack(spacing: 8) {
            ForEach(GlowPalette.allCases) { palette in
                Button {
                    selection = palette
                } label: {
                    Circle()
                        .fill(AngularGradient(
                            colors: [
                                EdgeGlowController.color(palette, .mildForward),
                                EdgeGlowController.color(palette, .severeForward),
                                EdgeGlowController.color(palette, .overextension),
                                EdgeGlowController.color(palette, .mildForward),
                            ],
                            center: .center
                        ))
                        .frame(width: 18, height: 18)
                        .overlay(
                            Circle().stroke(Color.primary.opacity(selection == palette ? 0.9 : 0), lineWidth: 2).padding(-3)
                        )
                }
                .buttonStyle(.plain)
                .help(palette.displayName)
            }
        }
        .padding(.trailing, 3)
    }
}
