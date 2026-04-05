import { create } from 'zustand';
import { persist } from 'zustand/middleware';

interface SettingsState {
  theme: 'light' | 'dark' | 'system';
  language: 'en' | 'sv';
  setTheme: (t: 'light' | 'dark' | 'system') => void;
  setLanguage: (l: 'en' | 'sv') => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      language: 'sv',
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
    }),
    { name: 'bilcool_settings' }
  )
);
