import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Link2, Plus, Trash2, X, MessageSquare, Loader2, RefreshCw, AlertCircle, Settings2, FolderOpen, Bot, ChevronRight } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { apiFetch, apiPost, apiDelete, apiUrl } from '../../lib/api'
import { getStoredToken } from '../../lib/auth'
import { cn } from '../../lib/cn'

type Platform = {
  type: string
  status?: string
}

type CCProject = {
  name: string
  platforms?: Platform[]
  sessions_count?: number
}

type CCConfig = { apiUrl: string; hasToken: boolean }

const SCAN_PLATFORMS = ['feishu', 'weixin'] as const
const MANUAL_PLATFORMS = ['telegram', 'discord', 'slack', 'dingtalk', 'wecom'] as const
const ALL_PLATFORMS = [...SCAN_PLATFORMS, ...MANUAL_PLATFORMS] as const

const PLATFORM_META: Record<string, { fields: { key: string; label: string; required?: boolean; secret?: boolean }[] }> = {
  telegram: { fields: [{ key: 'token', label: 'ccconnect.botToken', required: true, secret: true }] },
  discord: { fields: [{ key: 'token', label: 'ccconnect.botToken', required: true, secret: true }] },
  slack: { fields: [{ key: 'token', label: 'ccconnect.botToken', required: true, secret: true }, { key: 'app_token', label: 'App Token', required: true, secret: true }] },
  dingtalk: { fields: [{ key: 'app_key', label: 'App Key', required: true }, { key: 'app_secret', label: 'ccconnect.appSecret', required: true, secret: true }] },
  wecom: { fields: [{ key: 'corp_id', label: 'Corp ID', required: true }, { key: 'agent_id', label: 'Agent ID', required: true }, { key: 'secret', label: 'Secret', required: true, secret: true }] },
}

type QRState =
  | { step: 'idle' }
  | { step: 'loading' }
  | { step: 'scanning'; qrUrl: string; deviceCode?: string; qrKey?: string; interval: number; platform: string }
  | { step: 'scanned' }
  | { step: 'done' }
  | { step: 'error'; message: string }

export function IMConnectionPanel({ project, agentName, model, workDir }: {
  project: string; agentName: string; model: string; workDir?: string
}) {
  const { t } = useTranslation()
  const [ccConfig, setCcConfig] = useState<CCConfig | null>(null)
  const [ccProject, setCcProject] = useState<CCProject | null>(null)
  const [loading, setLoading] = useState(true)
  const [showAdd, setShowAdd] = useState(false)
  const [selectedPlatform, setSelectedPlatform] = useState<string>('')
  const [manualFields, setManualFields] = useState<Record<string, string>>({})
  const [addBusy, setAddBusy] = useState(false)
  const [qrState, setQrState] = useState<QRState>({ step: 'idle' })
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null)
  const feishuBaseUrlRef = useRef('')
  const feishuIntervalRef = useRef(5)
  const [needRestart, setNeedRestart] = useState(false)
  const [restarting, setRestarting] = useState(false)

  // Explicit project creation state
  const [showSetup, setShowSetup] = useState(false)
  const [setupProjectName, setSetupProjectName] = useState('')
  const [setupWorkDir, setSetupWorkDir] = useState('')
  const [setupAgentType, setSetupAgentType] = useState('')

  const defaultProjectName = `agencycli--${project}--${agentName}`

  const ccFetch = useCallback(async (path: string): Promise<Response | null> => {
    try {
      const headers: Record<string, string> = { Accept: 'application/json' }
      const token = getStoredToken()
      if (token) headers.Authorization = `Bearer ${token}`
      return await fetch(apiUrl(path), { headers })
    } catch { return null }
  }, [])

  const loadData = useCallback(async () => {
    try {
      const cfg = await apiFetch<CCConfig>('/api/v1/ccconnect/config')
      setCcConfig(cfg)
      if (cfg.apiUrl && cfg.hasToken) {
        const projRes = await ccFetch(`/api/v1/ccconnect/projects/${encodeURIComponent(defaultProjectName)}`)
        if (projRes?.ok) {
          setCcProject(await projRes.json())
        } else {
          setCcProject(null)
        }
      }
    } catch {
      setCcConfig({ apiUrl: '', hasToken: false })
    } finally { setLoading(false) }
  }, [defaultProjectName, ccFetch])

  useEffect(() => { void loadData() }, [loadData])

  useEffect(() => () => {
    if (pollRef.current) clearInterval(pollRef.current)
  }, [])

  // Initialize setup form when showing
  function openSetupWizard() {
    setSetupProjectName(defaultProjectName)
    setSetupWorkDir(workDir || '')
    setSetupAgentType(model || 'claudecode')
    setShowSetup(true)
    setSelectedPlatform('')
  }

  if (loading) return null
  if (!ccConfig || !ccConfig.apiUrl) {
    return (
      <section>
        <SHeader />
        <div className="mt-3 rounded-lg border border-neutral-200/80 bg-neutral-50/50 p-4 dark:border-zinc-700/60 dark:bg-zinc-900/30">
          <div className="flex items-center gap-2 text-sm text-neutral-400 dark:text-zinc-500">
            <AlertCircle className="size-4" />
            {t('ccconnect.notConfigured')}
          </div>
        </div>
      </section>
    )
  }

  const platforms = ccProject?.platforms ?? []
  const actualProjectName = showSetup ? setupProjectName : defaultProjectName
  const actualWorkDir = showSetup ? setupWorkDir : (workDir || '')
  const actualAgentType = showSetup ? setupAgentType : (model || 'claudecode')

  async function restartAndReload() {
    setNeedRestart(false)
    setRestarting(true)
    try {
      await apiPost('/api/v1/ccconnect/restart', {})
    } catch { /* ignore */ }
    for (let i = 0; i < 10; i++) {
      await new Promise(r => setTimeout(r, 2000))
      const res = await ccFetch(`/api/v1/ccconnect/projects/${encodeURIComponent(actualProjectName)}`)
      if (res?.ok) {
        setCcProject(await res.json())
        break
      }
    }
    setRestarting(false)
    setShowSetup(false)
    void loadData()
  }

  type FeishuPollRes = {
    status: string; base_url?: string; app_id?: string; app_secret?: string
    platform?: string; owner_open_id?: string; slow_down?: boolean; error?: string
  }

  async function startFeishuQR() {
    setQrState({ step: 'loading' })
    feishuBaseUrlRef.current = ''
    try {
      const res = await apiPost<{ device_code: string; qr_url: string; interval: number }>('/api/v1/ccconnect/setup/feishu/begin', {})
      feishuIntervalRef.current = res.interval || 5
      setQrState({ step: 'scanning', qrUrl: res.qr_url, deviceCode: res.device_code, interval: res.interval || 5, platform: 'feishu' })
      startFeishuPoll(res.device_code)
    } catch (e) {
      setQrState({ step: 'error', message: e instanceof Error ? e.message : String(e) })
    }
  }

  function startFeishuPoll(deviceCode: string) {
    if (pollRef.current) clearInterval(pollRef.current)
    const doPoll = async () => {
      try {
        const res = await apiPost<FeishuPollRes>('/api/v1/ccconnect/setup/feishu/poll', {
          device_code: deviceCode,
          base_url: feishuBaseUrlRef.current || undefined,
        })
        if (res.base_url) feishuBaseUrlRef.current = res.base_url
        if (res.slow_down) feishuIntervalRef.current += 5

        if (res.status === 'completed' && res.app_id && res.app_secret) {
          if (pollRef.current) clearInterval(pollRef.current)
          try {
            await apiPost('/api/v1/ccconnect/setup/feishu/save', {
              project: actualProjectName,
              app_id: res.app_id,
              app_secret: res.app_secret,
              platform_type: res.platform || 'feishu',
              owner_open_id: res.owner_open_id,
              work_dir: actualWorkDir,
              agent_type: actualAgentType,
            })
            setQrState({ step: 'done' })
            void restartAndReload()
          } catch (e) {
            setQrState({ step: 'error', message: e instanceof Error ? e.message : String(e) })
          }
        } else if (res.status === 'expired') {
          if (pollRef.current) clearInterval(pollRef.current)
          setQrState({ step: 'error', message: t('ccconnect.setupExpired') })
        } else if (res.status === 'denied') {
          if (pollRef.current) clearInterval(pollRef.current)
          setQrState({ step: 'error', message: t('ccconnect.setupFailed') })
        } else if (res.status === 'error') {
          if (pollRef.current) clearInterval(pollRef.current)
          setQrState({ step: 'error', message: res.error || t('ccconnect.setupFailed') })
        }
      } catch { /* continue polling */ }
    }
    pollRef.current = setInterval(() => void doPoll(), feishuIntervalRef.current * 1000)
  }

  type WeixinPollRes = {
    status: string; bot_token?: string; ilink_bot_id?: string
    base_url?: string; ilink_user_id?: string
  }

  async function startWeixinQR() {
    setQrState({ step: 'loading' })
    try {
      const res = await apiPost<{ qr_key: string; qr_url: string }>('/api/v1/ccconnect/setup/weixin/begin', {})
      setQrState({ step: 'scanning', qrUrl: res.qr_url, qrKey: res.qr_key, interval: 1, platform: 'weixin' })
      startWeixinPoll(res.qr_key)
    } catch (e) {
      setQrState({ step: 'error', message: e instanceof Error ? e.message : String(e) })
    }
  }

  function startWeixinPoll(qrKey: string) {
    if (pollRef.current) clearInterval(pollRef.current)
    let errCount = 0
    pollRef.current = setInterval(async () => {
      try {
        const res = await apiPost<WeixinPollRes>('/api/v1/ccconnect/setup/weixin/poll', { qr_key: qrKey })
        errCount = 0

        if (res.status === 'confirmed' && res.bot_token) {
          if (pollRef.current) clearInterval(pollRef.current)
          try {
            await apiPost('/api/v1/ccconnect/setup/weixin/save', {
              project: actualProjectName,
              token: res.bot_token,
              base_url: res.base_url,
              ilink_bot_id: res.ilink_bot_id,
              ilink_user_id: res.ilink_user_id,
              work_dir: actualWorkDir,
              agent_type: actualAgentType,
            })
            setQrState({ step: 'done' })
            void restartAndReload()
          } catch (e) {
            setQrState({ step: 'error', message: e instanceof Error ? e.message : String(e) })
          }
        } else if (res.status === 'scaned') {
          setQrState(prev => prev.step === 'scanning' ? { ...prev, step: 'scanning' } : prev)
        } else if (res.status === 'expired') {
          if (pollRef.current) clearInterval(pollRef.current)
          setQrState({ step: 'error', message: t('ccconnect.setupExpired') })
        }
      } catch {
        errCount++
        if (errCount >= 5) {
          if (pollRef.current) clearInterval(pollRef.current)
          setQrState({ step: 'error', message: t('ccconnect.setupFailed') })
        }
      }
    }, 500)
  }

  async function addManualPlatform() {
    if (!selectedPlatform) return
    setAddBusy(true)
    try {
      await apiPost(`/api/v1/ccconnect/projects/${encodeURIComponent(actualProjectName)}/add-platform`, {
        type: selectedPlatform,
        options: manualFields,
        work_dir: actualWorkDir,
        agent_type: actualAgentType,
      })
      setShowAdd(false)
      void restartAndReload()
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e))
    } finally { setAddBusy(false) }
  }

  async function handleDisconnect(_platformType: string) {
    if (!confirm(t('ccconnect.confirmDisconnect'))) return
    try {
      await apiDelete(`/api/v1/ccconnect/projects/${encodeURIComponent(defaultProjectName)}`)
      setNeedRestart(true)
      void loadData()
    } catch (e) {
      alert(e instanceof Error ? e.message : String(e))
    }
  }

  const isScan = SCAN_PLATFORMS.includes(selectedPlatform as typeof SCAN_PLATFORMS[number])

  const inputCls = 'block w-full rounded-md border border-neutral-200 bg-white px-3 py-1.5 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-200'

  // ─── Setup wizard (no cc-connect project yet) ─────────────────────────
  function renderSetupWizard() {
    return (
      <div className="mt-3 space-y-3">
        <div className="rounded-lg border border-sky-200/60 bg-sky-50/30 p-4 dark:border-sky-800/40 dark:bg-sky-950/20">
          <div className="flex items-center justify-between pb-3">
            <div className="flex items-center gap-2">
              <Settings2 className="size-4 text-sky-600 dark:text-sky-400" />
              <h4 className="text-sm font-semibold text-neutral-800 dark:text-zinc-200">{t('ccconnect.setupProject')}</h4>
            </div>
            <button type="button" onClick={() => { setShowSetup(false); setQrState({ step: 'idle' }) }}
              className="rounded-md p-1 text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
              <X className="size-4" />
            </button>
          </div>

          <p className="mb-4 text-xs text-neutral-500 dark:text-zinc-500">{t('ccconnect.setupProjectHint')}</p>

          {/* Step 1: Project config */}
          {!selectedPlatform && qrState.step === 'idle' && (
            <div className="space-y-3">
              <label className="flex flex-col gap-1">
                <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('ccconnect.ccProject')}</span>
                <input value={setupProjectName} onChange={e => setSetupProjectName(e.target.value)}
                  className={inputCls + ' max-w-sm font-mono text-xs'} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="flex items-center gap-1.5 text-xs font-medium text-neutral-600 dark:text-zinc-400">
                  <FolderOpen className="size-3" />{t('ccconnect.workDir')}
                </span>
                <input value={setupWorkDir} onChange={e => setSetupWorkDir(e.target.value)}
                  placeholder="/path/to/agent/dir"
                  className={inputCls + ' max-w-md font-mono text-xs'} />
              </label>
              <label className="flex flex-col gap-1">
                <span className="flex items-center gap-1.5 text-xs font-medium text-neutral-600 dark:text-zinc-400">
                  <Bot className="size-3" />{t('ccconnect.agentType')}
                </span>
                <select value={setupAgentType} onChange={e => setSetupAgentType(e.target.value)}
                  className={inputCls + ' max-w-xs'}>
                  {['claudecode', 'codex', 'cursor', 'gemini', 'acp', 'generic-cli'].map(t => (
                    <option key={t} value={t}>{t}</option>
                  ))}
                </select>
              </label>

              <div className="pt-2">
                <p className="mb-2 text-xs font-medium text-neutral-600 dark:text-zinc-400">{t('ccconnect.selectPlatform')}</p>
                <div className="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-7">
                  {ALL_PLATFORMS.map(p => (
                    <button key={p} type="button"
                      onClick={() => { setSelectedPlatform(p); setManualFields({}) }}
                      className={cn(
                        'rounded-lg border px-3 py-2 text-xs font-medium transition-all',
                        'border-neutral-200 bg-white text-neutral-600 hover:border-sky-300 hover:text-sky-600 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:border-sky-600 dark:hover:text-sky-400',
                      )}>
                      {t(`ccconnect.${p}`, { defaultValue: p })}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          )}

          {/* Step 2: Platform config (after selection) */}
          {selectedPlatform && qrState.step === 'idle' && (
            <div className="space-y-3">
              <div className="flex items-center gap-2 rounded-md bg-sky-100/60 px-3 py-2 text-xs dark:bg-sky-900/20">
                <span className="font-medium text-sky-700 dark:text-sky-400">{t(`ccconnect.${selectedPlatform}`, { defaultValue: selectedPlatform })}</span>
                <button type="button" onClick={() => setSelectedPlatform('')}
                  className="ml-auto text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
                  <X className="size-3" />
                </button>
              </div>

              <div className="rounded-md border border-neutral-100 bg-neutral-50/50 px-3 py-2 dark:border-zinc-800 dark:bg-zinc-900/30">
                <div className="space-y-1 text-xs text-neutral-500 dark:text-zinc-500">
                  <p><span className="font-medium text-neutral-600 dark:text-zinc-400">{t('ccconnect.ccProject')}:</span> <span className="font-mono">{setupProjectName}</span></p>
                  <p><span className="font-medium text-neutral-600 dark:text-zinc-400">{t('ccconnect.workDir')}:</span> <span className="font-mono">{setupWorkDir || '—'}</span></p>
                  <p><span className="font-medium text-neutral-600 dark:text-zinc-400">{t('ccconnect.agentType')}:</span> {setupAgentType}</p>
                </div>
              </div>

              {isScan ? (
                <button type="button" disabled={!setupWorkDir.trim()}
                  onClick={() => { selectedPlatform === 'feishu' ? startFeishuQR() : startWeixinQR() }}
                  className="flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">
                  <ChevronRight className="size-3.5" />
                  {t('ccconnect.startSetup')}
                </button>
              ) : PLATFORM_META[selectedPlatform] ? (
                <div className="space-y-2">
                  {PLATFORM_META[selectedPlatform].fields.map(f => (
                    <label key={f.key} className="flex flex-col gap-1">
                      <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">
                        {t(f.label, { defaultValue: f.label })} {f.required && '*'}
                      </span>
                      <input
                        type={f.secret ? 'password' : 'text'}
                        value={manualFields[f.key] ?? ''}
                        onChange={e => setManualFields(prev => ({ ...prev, [f.key]: e.target.value }))}
                        className={inputCls + ' max-w-sm'}
                      />
                    </label>
                  ))}
                  <button type="button" onClick={() => void addManualPlatform()} disabled={addBusy || !setupWorkDir.trim()}
                    className="flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">
                    {addBusy ? <Loader2 className="size-3.5 animate-spin" /> : <Plus className="size-3.5" />}
                    {t('ccconnect.createAndConnect')}
                  </button>
                </div>
              ) : null}
            </div>
          )}

          {/* QR scan in wizard */}
          {renderQRFlow()}
        </div>
      </div>
    )
  }

  // ─── QR scan flow (shared) ────────────────────────────────────────────
  function renderQRFlow() {
    if (qrState.step === 'idle' || qrState.step === 'done') return null
    return (
      <div className="mt-3">
        {qrState.step === 'loading' && (
          <div className="flex items-center justify-center gap-2 py-8">
            <Loader2 className="size-5 animate-spin text-sky-600" />
            <span className="text-sm text-neutral-500">{t('api.loading')}</span>
          </div>
        )}
        {qrState.step === 'scanning' && (
          <div className="flex flex-col items-center gap-4">
            <p className="text-sm font-medium text-neutral-700 dark:text-zinc-300">{t('ccconnect.scanQRHint')}</p>
            <div className="rounded-xl border border-neutral-200 bg-white p-3 dark:border-zinc-600">
              <QRCodeSVG value={qrState.qrUrl} size={200} />
            </div>
            <p className="text-xs text-neutral-400 dark:text-zinc-500">{t('ccconnect.waitingScan')}</p>
            <button type="button" onClick={() => { if (pollRef.current) clearInterval(pollRef.current); setQrState({ step: 'idle' }); setSelectedPlatform('') }}
              className="text-xs text-neutral-500 hover:text-neutral-700 dark:text-zinc-500 dark:hover:text-zinc-300">
              {t('forms.cancel')}
            </button>
          </div>
        )}
        {qrState.step === 'error' && (
          <div className="flex flex-col items-center gap-3 py-4">
            <p className="text-sm text-red-500">{qrState.message}</p>
            <button type="button" onClick={() => { setQrState({ step: 'idle' }); setSelectedPlatform('') }}
              className="rounded-md bg-sky-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-sky-700">
              {t('ccconnect.retry')}
            </button>
          </div>
        )}
      </div>
    )
  }

  // ─── Main render ──────────────────────────────────────────────────────
  return (
    <section>
      <SHeader />

      {/* Setup wizard mode */}
      {showSetup && renderSetupWizard()}

      {/* Normal mode (project exists or idle) */}
      {!showSetup && (
        <div className="mt-3 space-y-3">
          {/* Connected platforms */}
          {platforms.length > 0 && (
            <div className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
              <div className="divide-y divide-neutral-100 dark:divide-zinc-800/40">
                {platforms.map(p => (
                  <div key={p.type} className="flex items-center justify-between px-4 py-3">
                    <div className="flex items-center gap-3">
                      <span className="inline-flex size-8 items-center justify-center rounded-lg bg-sky-100 dark:bg-sky-900/30">
                        <MessageSquare className="size-4 text-sky-600 dark:text-sky-400" strokeWidth={1.8} />
                      </span>
                      <div>
                        <p className="text-sm font-medium text-neutral-800 dark:text-zinc-200">
                          {t(`ccconnect.${p.type}`, { defaultValue: p.type })}
                        </p>
                        <p className="text-xs text-neutral-400 dark:text-zinc-500">
                          {p.status || 'connected'}
                        </p>
                      </div>
                    </div>
                    <button type="button" onClick={() => void handleDisconnect(p.type)}
                      className="rounded-md p-1.5 text-neutral-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20 dark:hover:text-red-400">
                      <Trash2 className="size-3.5" strokeWidth={2} />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* No platforms yet */}
          {platforms.length === 0 && qrState.step === 'idle' && !showAdd && (
            <div className="rounded-lg border border-dashed border-neutral-300 bg-neutral-50/50 py-8 text-center dark:border-zinc-700 dark:bg-zinc-900/20">
              <MessageSquare className="mx-auto mb-2 size-6 text-neutral-300 dark:text-zinc-600" strokeWidth={1.5} />
              <p className="text-sm text-neutral-400 dark:text-zinc-500">{t('ccconnect.noPlatforms')}</p>
            </div>
          )}

          {/* Restart notice */}
          {needRestart && (
            <div className="flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50/50 px-4 py-2.5 dark:border-amber-800/40 dark:bg-amber-950/20">
              <span className="text-sm text-amber-700 dark:text-amber-400">{t('ccconnect.restartRequired')}</span>
              <div className="flex gap-2">
                <button type="button" onClick={() => setNeedRestart(false)}
                  className="rounded-md px-2.5 py-1 text-xs text-neutral-500 hover:bg-neutral-100 dark:text-zinc-500 dark:hover:bg-zinc-800">
                  {t('ccconnect.restartLater')}
                </button>
                <button type="button" onClick={() => void restartAndReload()} disabled={restarting}
                  className="flex items-center gap-1 rounded-md bg-amber-600 px-3 py-1 text-xs font-medium text-white hover:bg-amber-700 disabled:opacity-50">
                  {restarting ? <Loader2 className="size-3 animate-spin" /> : <RefreshCw className="size-3" />}
                  {restarting ? t('ccconnect.restarting') : t('ccconnect.restartNow')}
                </button>
              </div>
            </div>
          )}

          {/* QR flow for adding platform to existing project */}
          {qrState.step !== 'idle' && qrState.step !== 'done' && (
            <div className="rounded-lg border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
              {renderQRFlow()}
            </div>
          )}

          {qrState.step === 'done' && (
            <div className="flex items-center gap-2 rounded-lg bg-emerald-50 px-4 py-3 text-sm font-medium text-emerald-700 dark:bg-emerald-900/20 dark:text-emerald-400">
              {restarting ? <Loader2 className="size-4 animate-spin" /> : <Link2 className="size-4" />}
              {restarting ? t('ccconnect.restarting') : t('ccconnect.setupComplete')}
            </div>
          )}

          {/* Add platform to existing project */}
          {ccProject && showAdd && qrState.step === 'idle' && (
            <div className="rounded-lg border border-sky-200/60 bg-sky-50/30 p-4 dark:border-sky-800/40 dark:bg-sky-950/20">
              <div className="flex items-center justify-between pb-3">
                <h4 className="text-sm font-semibold text-neutral-800 dark:text-zinc-200">{t('ccconnect.addPlatform')}</h4>
                <button type="button" onClick={() => setShowAdd(false)}
                  className="rounded-md p-1 text-neutral-400 hover:text-neutral-600 dark:hover:text-zinc-300">
                  <X className="size-4" />
                </button>
              </div>
              <div className="space-y-3">
                <div className="grid grid-cols-3 gap-2 sm:grid-cols-4 lg:grid-cols-7">
                  {ALL_PLATFORMS.map(p => (
                    <button key={p} type="button"
                      onClick={() => { setSelectedPlatform(p); setManualFields({}) }}
                      className={cn(
                        'rounded-lg border px-3 py-2 text-xs font-medium transition-all',
                        selectedPlatform === p
                          ? 'border-sky-400 bg-sky-50 text-sky-700 dark:border-sky-600 dark:bg-sky-900/30 dark:text-sky-400'
                          : 'border-neutral-200 bg-white text-neutral-600 hover:border-neutral-300 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:border-zinc-600',
                      )}>
                      {t(`ccconnect.${p}`, { defaultValue: p })}
                    </button>
                  ))}
                </div>
                {selectedPlatform && isScan && (
                  <button type="button"
                    onClick={() => { setShowAdd(false); selectedPlatform === 'feishu' ? startFeishuQR() : startWeixinQR() }}
                    className="flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700">
                    {t('ccconnect.startSetup')}
                  </button>
                )}
                {selectedPlatform && !isScan && PLATFORM_META[selectedPlatform] && (
                  <div className="space-y-2">
                    {PLATFORM_META[selectedPlatform].fields.map(f => (
                      <label key={f.key} className="flex flex-col gap-1">
                        <span className="text-xs font-medium text-neutral-600 dark:text-zinc-400">
                          {t(f.label, { defaultValue: f.label })} {f.required && '*'}
                        </span>
                        <input type={f.secret ? 'password' : 'text'} value={manualFields[f.key] ?? ''}
                          onChange={e => setManualFields(prev => ({ ...prev, [f.key]: e.target.value }))}
                          className={inputCls + ' max-w-sm'} />
                      </label>
                    ))}
                    <button type="button" onClick={() => void addManualPlatform()} disabled={addBusy}
                      className="flex items-center gap-1.5 rounded-md bg-sky-600 px-4 py-2 text-sm font-medium text-white hover:bg-sky-700 disabled:opacity-50">
                      {addBusy ? <Loader2 className="size-3.5 animate-spin" /> : <Plus className="size-3.5" />}
                      {t('ccconnect.addPlatform')}
                    </button>
                  </div>
                )}
              </div>
            </div>
          )}

          {/* Connect / Add platform button */}
          {qrState.step === 'idle' && !showAdd && (
            ccProject ? (
              <button type="button" onClick={() => { setShowAdd(true); setSelectedPlatform('') }}
                className="flex items-center gap-1.5 rounded-md border border-dashed border-neutral-300 px-3 py-2 text-xs font-medium text-neutral-500 transition-colors hover:border-neutral-400 hover:text-neutral-700 dark:border-zinc-700 dark:text-zinc-500 dark:hover:border-zinc-600 dark:hover:text-zinc-300">
                <Plus className="size-3.5" strokeWidth={2} />
                {t('ccconnect.addPlatform')}
              </button>
            ) : (
              <button type="button" onClick={openSetupWizard}
                className="flex items-center gap-1.5 rounded-md border border-dashed border-sky-300 bg-sky-50/30 px-3 py-2 text-xs font-medium text-sky-600 transition-colors hover:border-sky-400 hover:bg-sky-50 dark:border-sky-700 dark:bg-sky-950/20 dark:text-sky-400 dark:hover:border-sky-600">
                <Settings2 className="size-3.5" strokeWidth={2} />
                {t('ccconnect.setupAndConnect')}
              </button>
            )
          )}

          {/* cc-connect project info */}
          {ccProject && (
            <div className="flex items-center gap-4 text-xs text-neutral-400 dark:text-zinc-500">
              <span>{t('ccconnect.ccProject')}: <span className="font-mono">{ccProject.name}</span></span>
              {ccProject.sessions_count != null && <span>{t('ccconnect.sessions')}: {ccProject.sessions_count}</span>}
            </div>
          )}
        </div>
      )}
    </section>
  )
}

function SHeader() {
  const { t } = useTranslation()
  return (
    <div className="flex items-center gap-2.5">
      <div className="flex size-7 items-center justify-center rounded-lg bg-neutral-100 dark:bg-zinc-800">
        <Link2 className="size-3.5 text-neutral-500 dark:text-zinc-400" strokeWidth={1.8} />
      </div>
      <h3 className="text-sm font-semibold text-neutral-800 dark:text-zinc-200">{t('ccconnect.imConnections')}</h3>
    </div>
  )
}
