import { Sun, Moon, Monitor } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useSettingsStore } from '../../stores/settingsStore'
import { Button } from '../ui/button'

const themes = ['light', 'dark', 'system'] as const

export default function ThemeToggle() {
  const { t } = useTranslation('common')
  const theme = useSettingsStore((s) => s.theme)
  const setTheme = useSettingsStore((s) => s.setTheme)

  function cycleTheme() {
    const idx = themes.indexOf(theme)
    setTheme(themes[(idx + 1) % themes.length])
  }

  const icon =
    theme === 'light' ? (
      <Sun className="h-4 w-4" />
    ) : theme === 'dark' ? (
      <Moon className="h-4 w-4" />
    ) : (
      <Monitor className="h-4 w-4" />
    )

  const label = t(`theme.${theme}`)

  return (
    <Button
      variant="ghost"
      size="icon"
      onClick={cycleTheme}
      aria-label={`Theme: ${label}. Click to change.`}
      title={label}
      className="min-h-[44px] min-w-[44px]"
    >
      {icon}
    </Button>
  )
}
