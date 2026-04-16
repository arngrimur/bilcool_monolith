import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { DEFAULT_BOOKING_COLOR } from '../utils/bookingColors';

interface SettingsState {
  theme: 'light' | 'dark' | 'system';
  language: 'en' | 'sv';
  bookingColor: string;
  setTheme: (t: 'light' | 'dark' | 'system') => void;
  setLanguage: (l: 'en' | 'sv') => void;
  setBookingColor: (c: string) => void;
}

export const useSettingsStore = create<SettingsState>()(
  persist(
    (set) => ({
      theme: 'system',
      language: 'sv',
      bookingColor: DEFAULT_BOOKING_COLOR,
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      setBookingColor: (bookingColor) => set({ bookingColor }),
    }),
    { name: 'bilcool_settings' }
  )
);
