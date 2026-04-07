import { useTranslation } from 'react-i18next'
import { useSettingsStore } from '../../stores/settingsStore'
import { Button } from '../ui/button'

export default function LanguageToggle() {
  const { t } = useTranslation('common')
  const language = useSettingsStore((s) => s.language)
  const setLanguage = useSettingsStore((s) => s.setLanguage)

  function toggle() {
    setLanguage(language === 'en' ? 'sv' : 'en')
  }

  return (
    <Button
      variant="ghost"
      size="sm"
      onClick={toggle}
      aria-label={`Language: ${t(`language.${language}`)}. Click to switch.`}
      title={t(`language.${language}`)}
      className="min-h-[44px] px-2 font-medium text-xs uppercase tracking-wide"
    >
      {language}
    </Button>
  )
}
