import { useEffect } from 'react'
import { useSettingsStore } from '../stores/settingsStore'
import type { AccentColor } from '../stores/settingsStore'
import i18n from '../i18n/index'

const ACCENT_COLORS: AccentColor[] = ['blue', 'green', 'purple', 'rose']

export function useThemeSync() {
  const theme = useSettingsStore((s) => s.theme)
  const language = useSettingsStore((s) => s.language)
  const accentColor = useSettingsStore((s) => s.accentColor)

  useEffect(() => {
    const root = document.documentElement

    if (theme === 'dark') {
      root.classList.add('dark')
    } else if (theme === 'light') {
      root.classList.remove('dark')
    } else {
      const mq = window.matchMedia('(prefers-color-scheme: dark)')
      if (mq.matches) {
        root.classList.add('dark')
      } else {
        root.classList.remove('dark')
      }
      const handler = (e: MediaQueryListEvent) => {
        if (e.matches) {
          root.classList.add('dark')
        } else {
          root.classList.remove('dark')
        }
      }
      mq.addEventListener('change', handler)
      return () => mq.removeEventListener('change', handler)
    }
  }, [theme])

  useEffect(() => {
    i18n.changeLanguage(language)
    document.documentElement.setAttribute('lang', language)
  }, [language])

  useEffect(() => {
    const root = document.documentElement
    ACCENT_COLORS.forEach((c) => root.classList.remove(`accent-${c}`))
    if (accentColor !== 'default') {
      root.classList.add(`accent-${accentColor}`)
    }
  }, [accentColor])
}
