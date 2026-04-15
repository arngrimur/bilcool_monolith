import { useTranslation } from 'react-i18next'
import { useSettingsStore } from '../../stores/settingsStore'
import type { AccentColor } from '../../stores/settingsStore'
import { cn } from '../../lib/utils'

const COLOR_OPTIONS: { value: AccentColor; hex: string }[] = [
  { value: 'default', hex: '#71717a' },
  { value: 'blue',    hex: '#3b82f6' },
  { value: 'green',   hex: '#22c55e' },
  { value: 'purple',  hex: '#a855f7' },
  { value: 'rose',    hex: '#f43f5e' },
]

export default function ColorSelector() {
  const { t } = useTranslation('common')
  const accentColor = useSettingsStore((s) => s.accentColor)
  const setAccentColor = useSettingsStore((s) => s.setAccentColor)

  return (
    <div
      className="flex items-center gap-1.5 px-1"
      role="radiogroup"
      aria-label={t('color.label')}
    >
      {COLOR_OPTIONS.map(({ value, hex }) => (
        <button
          key={value}
          role="radio"
          aria-checked={accentColor === value}
          aria-label={t(`color.${value}`)}
          title={t(`color.${value}`)}
          onClick={() => setAccentColor(value)}
          className={cn(
            'h-5 w-5 rounded-full ring-offset-background transition-all focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring focus-visible:ring-offset-2',
            accentColor === value ? 'ring-2 ring-ring ring-offset-2' : 'opacity-70 hover:opacity-100'
          )}
          style={{ backgroundColor: hex }}
        />
      ))}
    </div>
  )
}
