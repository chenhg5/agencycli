import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { KeyRound } from 'lucide-react'
import { i18n } from '../i18n'
import type { ThemeMode } from '../theme/ThemeProvider'
import { useTheme } from '../theme/ThemeProvider'
import { useAuth } from '../lib/auth'
import { apiPut } from '../lib/api'

const selectCls =
  'max-w-xs rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm text-neutral-800 outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200'
const inputCls =
  'block w-full max-w-xs rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm text-neutral-800 outline-none transition-colors placeholder:text-neutral-400 focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:placeholder:text-zinc-600'

function ChangePasswordSection() {
  const { t } = useTranslation()
  const [oldPwd, setOldPwd] = useState('')
  const [newPwd, setNewPwd] = useState('')
  const [confirmPwd, setConfirmPwd] = useState('')
  const [saving, setSaving] = useState(false)
  const [msg, setMsg] = useState<{ type: 'ok' | 'err'; text: string } | null>(null)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setMsg(null)
    if (newPwd.length < 6) {
      setMsg({ type: 'err', text: t('auth.pwdTooShort') })
      return
    }
    if (newPwd !== confirmPwd) {
      setMsg({ type: 'err', text: t('auth.pwdMismatch') })
      return
    }
    setSaving(true)
    try {
      await apiPut('/api/v1/auth/password', { oldPassword: oldPwd, newPassword: newPwd })
      setMsg({ type: 'ok', text: t('auth.pwdChanged') })
      setOldPwd('')
      setNewPwd('')
      setConfirmPwd('')
    } catch (err) {
      setMsg({ type: 'err', text: err instanceof Error ? err.message : String(err) })
    } finally {
      setSaving(false)
    }
  }

  return (
    <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
      <div className="flex items-center gap-2 pb-3">
        <KeyRound className="size-4 text-neutral-500 dark:text-zinc-500" strokeWidth={1.8} />
        <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
          {t('auth.changePassword')}
        </h3>
      </div>
      <form onSubmit={handleSubmit} className="space-y-3">
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.oldPassword')}</span>
          <input type="password" autoComplete="current-password" value={oldPwd} onChange={(e) => setOldPwd(e.target.value)} className={inputCls} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.newPassword')}</span>
          <input type="password" autoComplete="new-password" value={newPwd} onChange={(e) => setNewPwd(e.target.value)} className={inputCls} placeholder={t('auth.pwdMinHint')} />
        </label>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('auth.confirmPassword')}</span>
          <input type="password" autoComplete="new-password" value={confirmPwd} onChange={(e) => setConfirmPwd(e.target.value)} className={inputCls} />
        </label>
        {msg && (
          <p className={`rounded-md px-3 py-2 text-sm ${msg.type === 'ok' ? 'bg-emerald-50 text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400' : 'bg-red-50 text-red-600 dark:bg-red-900/20 dark:text-red-400'}`}>
            {msg.text}
          </p>
        )}
        <button
          type="submit"
          disabled={saving || !oldPwd || !newPwd || !confirmPwd}
          className="rounded-lg bg-sky-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-sky-700 disabled:opacity-50"
        >
          {saving ? t('prompt.saving') : t('auth.changePassword')}
        </button>
      </form>
    </section>
  )
}

export default function SettingsPage() {
  const { t } = useTranslation()
  const { theme, setTheme } = useTheme()
  const { user } = useAuth()
  const lang = (() => {
    const l = i18n.language
    if (l.startsWith('zh-TW') || l === 'zh-Hant') return 'zh-TW'
    if (l.startsWith('zh')) return 'zh-CN'
    if (l.startsWith('ja')) return 'ja'
    return 'en'
  })()

  return (
    <div className="animate-fade-in px-8 py-6">
      <div className="pb-5">
        <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">{t('settings.title')}</h1>
        <p className="mt-0.5 text-sm text-neutral-500 dark:text-zinc-500">{t('settings.intro')}</p>
        {user && (
          <p className="mt-1.5 text-sm text-neutral-400 dark:text-zinc-500">
            {t('auth.loggedInAs')} <span className="font-medium text-neutral-700 dark:text-zinc-300">{user.username}</span>
            <span className="ml-2 rounded-full bg-sky-100 px-2 py-0.5 text-xs font-medium text-sky-700 dark:bg-sky-900/30 dark:text-sky-400">{user.role}</span>
          </p>
        )}
      </div>

      <div className="space-y-5">
        {/* Language */}
        <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
            {t('settings.languageSection')}
          </h3>
          <label className="mt-3 flex flex-col gap-1.5">
            <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('language.label')}</span>
            <select
              className={selectCls}
              value={lang}
              onChange={(e) => void i18n.changeLanguage(e.target.value)}
            >
              <option value="en">{t('language.en')}</option>
              <option value="zh-CN">{t('language.zhCN')}</option>
              <option value="zh-TW">{t('language.zhTW')}</option>
              <option value="ja">{t('language.ja')}</option>
            </select>
          </label>
        </section>

        {/* Appearance */}
        <section className="rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
          <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">
            {t('settings.appearanceSection')}
          </h3>
          <label className="mt-3 flex flex-col gap-1.5">
            <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('theme.appearance')}</span>
            <select
              className={selectCls}
              value={theme}
              onChange={(e) => setTheme(e.target.value as ThemeMode)}
            >
              <option value="light">{t('theme.light')}</option>
              <option value="dark">{t('theme.dark')}</option>
              <option value="system">{t('theme.system')}</option>
            </select>
          </label>
          <p className="mt-3 text-sm text-neutral-400 dark:text-zinc-500">{t('settings.themeHint')}</p>
        </section>

        {/* Change Password */}
        <ChangePasswordSection />
      </div>
    </div>
  )
}
