/** Engine settings stored by the engine; they replace the env file. */
export type SettingKind = 'text' | 'secret' | 'integer' | 'url' | 'choice' | 'path';

export interface SettingGroup {
  id: string;
  label: string;
  description: string;
}

export interface EngineSetting {
  key: string;
  group: string;
  label: string;
  description: string;
  kind: SettingKind;
  /** What the engine uses when the setting is empty. */
  default?: string;
  options?: string[];
  /** Secrets are write-only: only isSet is reported, never the value. */
  secret: boolean;
  isSet: boolean;
  value?: string;
}

export interface EngineSettings {
  groups: SettingGroup[];
  settings: EngineSetting[];
  /** Where the engine stores its settings. */
  path: string;
  /** A saved change is not used until the engine restarts. */
  restartRequired: boolean;
}

/** Keys to new values; null clears a setting. */
export type SettingsChanges = Record<string, string | null>;

/** Which refresh token a Google sign-in mints. */
export type GoogleAuthPurpose = 'sheets' | 'test';

export interface GoogleAuthStart {
  /** Google's consent page; open it in a browser. */
  authUrl: string;
  /** Where Google sends the browser back; the OAuth client must allow it. */
  redirectUrl: string;
  purpose: GoogleAuthPurpose;
}

/** A saved refresh token; the token itself is never returned. */
export interface GoogleAuthResult {
  purpose: GoogleAuthPurpose;
  settingKey: string;
  restartRequired: boolean;
}
