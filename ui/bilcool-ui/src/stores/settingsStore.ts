import { create } from 'zustand';
import { persist } from 'zustand/middleware';

export type AccentColor = 'default' | 'blue' | 'green' | 'purple' | 'rose';

interface SettingsState {
  theme: 'light' | 'dark' | 'system';
  language: 'en' | 'sv';
  accentColor: AccentColor;
  setTheme: (t: 'light' | 'dark' | 'system') => void;
  setLanguage: (l: 'en' | 'sv') => void;
  setAccentColor: (c: AccentColor) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      language: 'sv',
      accentColor: 'default',
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      setAccentColor: (accentColor) => set({ accentColor }),
    }),
    { name: 'bilcool_settings' }
  )
);
