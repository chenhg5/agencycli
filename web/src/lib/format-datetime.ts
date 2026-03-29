import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'

/** Formats an ISO / timestamp using the active i18n language (en vs zh-CN). */
export function useFormatDateTime() {
  const { i18n } = useTranslation()

  return useCallback(
    (input: string | number | Date | null | undefined) => {
      if (input == null || input === '') {
        return '—'
      }
      const d = input instanceof Date ? input : new Date(input)
      if (Number.isNaN(d.getTime())) {
        return '—'
      }
      const locale = i18n.language === 'zh-CN' ? 'zh-CN' : 'en'
      return new Intl.DateTimeFormat(locale, {
        dateStyle: 'short',
        timeStyle: 'medium',
      }).format(d)
    },
    [i18n.language],
  )
}
