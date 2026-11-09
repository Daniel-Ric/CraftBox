import type {
  WorkspacePreferences,
  PreferenceKey,
} from "./workspace-preferences";
export type IngameInterfaceSettings = {
  enabled: boolean;
  confirmDisconnect: boolean;
  modelPickFeedback: boolean;
  modelPickFocus: boolean;
  menuColors: boolean;
};
export const interfaceKeys: PreferenceKey[] = [
  "ingameHotbarMessages",
  "ingameConfirmDisconnect",
  "ingameModelPickFeedback",
  "ingameModelPickFocus",
  "ingameMenuColors",
];
export function interfacePreferences(settings: IngameInterfaceSettings) {
  return {
    ingameHotbarMessages: settings.enabled,
    ingameConfirmDisconnect: settings.confirmDisconnect,
    ingameModelPickFeedback: settings.modelPickFeedback,
    ingameModelPickFocus: settings.modelPickFocus,
    ingameMenuColors: settings.menuColors,
  };
}
export function interfaceSettings(
  prefs: WorkspacePreferences,
): IngameInterfaceSettings {
  return {
    enabled: prefs.ingameHotbarMessages,
    confirmDisconnect: prefs.ingameConfirmDisconnect,
    modelPickFeedback: prefs.ingameModelPickFeedback,
    modelPickFocus: prefs.ingameModelPickFocus,
    menuColors: prefs.ingameMenuColors,
  };
}
