// Stealth mode (WM-UI-2): money is hidden by default, toggled globally by
// clicking any value or the header eye, and re-masked on every navigation.
// Never persisted, so a refresh always starts masked.

class PrivacyMask {
  revealed = $state(false);

  toggle(): void {
    this.revealed = !this.revealed;
  }

  hide(): void {
    this.revealed = false;
  }
}

export const mask = new PrivacyMask();
