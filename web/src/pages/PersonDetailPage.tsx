import { useCallback, useEffect, useState } from 'react'
import { useParams, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { User, FolderKanban, Mail, ListTodo, Plus, X, ChevronDown, ChevronRight, Reply, Send, CheckCircle2, AtSign, Phone } from 'lucide-react'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { apiFetch, apiPost, apiPut } from '../lib/api'
import { cn } from '../lib/cn'
import { Pagination } from '../components/ui/Pagination'

type PersonInfo = {
  username: string; role: string; displayName?: string
  email?: string; avatar?: string; phone?: string; bio?: string
  projects?: { project: string; role: string }[]
  linkedAgents?: string[]; disabled?: boolean; createdAt?: string
}
type ProjectItem = { name: string; description?: string }
type MsgRow = {
  id: string; from: string; to: string; subject?: string; body: string
  sentAt: string; readAt?: string; archivedAt?: string; mailbox: string
}
type TaskRow = {
  id: string; project: string; agent: string; title: string; type?: string
  assignee?: string; prompt?: string; priority: number; status: string
  createdAt: string; updatedAt: string
}

const PAGE_SIZE = 10
const fieldCls = 'w-full rounded-md border border-neutral-200/80 bg-neutral-50/50 px-3 py-2 text-sm outline-none transition-colors focus:border-sky-400 dark:border-zinc-700/60 dark:bg-zinc-800/50 dark:text-zinc-200 dark:[color-scheme:dark]'

export default function PersonDetailPage() {
  const { username } = useParams<{ username: string }>()
  const { t } = useTranslation()
  const [person, setPerson] = useState<PersonInfo | null>(null)
  const [loading, setLoading] = useState(true)
  const [tab, setTab] = useState<'messages' | 'tasks'>('messages')

  const [msgs, setMsgs] = useState<MsgRow[]>([])
  const [tasks, setTasks] = useState<TaskRow[]>([])
  const [msgPage, setMsgPage] = useState(1)
  const [taskPage, setTaskPage] = useState(1)

  const [allProjects, setAllProjects] = useState<ProjectItem[]>([])
  const [assigning, setAssigning] = useState(false)
  const [assignProject, setAssignProject] = useState('')
  const [assignTeam, setAssignTeam] = useState('')
  const [teams, setTeams] = useState<{ path: string; name: string }[]>([])
  const [assignBusy, setAssignBusy] = useState(false)

  const [expandedMsg, setExpandedMsg] = useState<string | null>(null)
  const [replyingTo, setReplyingTo] = useState<string | null>(null)
  const [replyText, setReplyText] = useState('')
  const [replying, setReplying] = useState(false)
  const [expandedTask, setExpandedTask] = useState<string | null>(null)

  const refresh = useCallback(async () => {
    if (!username) return
    try {
      const [p, projects] = await Promise.all([
        apiFetch<PersonInfo>(`/api/v1/users/${encodeURIComponent(username)}`),
        apiFetch<ProjectItem[]>('/api/v1/projects'),
      ])
      setPerson(p)
      setAllProjects(projects ?? [])

      const linked = p.linkedAgents ?? []
      if (linked.length > 0) {
        const allMsgs: MsgRow[] = []
        const allTasks: TaskRow[] = []
        for (const la of linked) {
          const [proj] = la.split('/')
          if (!proj) continue
          try {
            const m = await apiFetch<MsgRow[]>(`/api/v1/projects/${encodeURIComponent(proj)}/messages?mailbox=${encodeURIComponent(la)}&archived=all`)
            allMsgs.push(...(m ?? []))
          } catch { /* ignore */ }
          try {
            const agName = la.split('/')[1] ?? ''
            const tsk = await apiFetch<TaskRow[]>(`/api/v1/projects/${encodeURIComponent(proj)}/tasks?agent=${encodeURIComponent(agName)}&scope=all`)
            allTasks.push(...(tsk ?? []))
          } catch { /* ignore */ }
        }
        allMsgs.sort((a, b) => new Date(b.sentAt).getTime() - new Date(a.sentAt).getTime())
        allTasks.sort((a, b) => new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime())
        setMsgs(allMsgs)
        setTasks(allTasks)
      }
    } catch { /* ignore */ }
    finally { setLoading(false) }
  }, [username])

  useEffect(() => { void refresh() }, [refresh])

  useEffect(() => {
    if (!assigning) return
    apiFetch<{ path: string; name: string }[]>('/api/v1/teams').then(setTeams).catch(() => {})
  }, [assigning])

  async function handleAssign() {
    if (!assignProject || !assignTeam || !username) return
    setAssignBusy(true)
    try {
      await apiPost(`/api/v1/projects/${encodeURIComponent(assignProject)}/hire`, {
        name: username, team: assignTeam, model: 'human',
      })
      setAssigning(false)
      setAssignProject(''); setAssignTeam('')
      await refresh()
    } catch { /* ignore */ }
    finally { setAssignBusy(false) }
  }

  async function markRead(mailbox: string, msgId: string) {
    const [proj] = mailbox.split('/')
    if (!proj) return
    try {
      await apiPost(`/api/v1/projects/${encodeURIComponent(proj)}/messages/${msgId}/read`, { mailbox })
      setMsgs(prev => prev.map(m => m.id === msgId ? { ...m, readAt: new Date().toISOString() } : m))
    } catch { /* ignore */ }
  }

  async function sendReply(msg: MsgRow) {
    if (!replyText.trim()) return
    setReplying(true)
    const [proj] = msg.mailbox.split('/')
    if (!proj) { setReplying(false); return }
    try {
      await apiPost(`/api/v1/projects/${encodeURIComponent(proj)}/messages`, {
        from: msg.to, to: msg.from, subject: `Re: ${msg.subject ?? ''}`,
        body: replyText.trim(), replyTo: msg.id,
      })
      if (!msg.readAt) await markRead(msg.mailbox, msg.id)
      setReplyText(''); setReplyingTo(null)
    } catch { /* ignore */ }
    finally { setReplying(false) }
  }

  if (loading) return <div className="flex items-center justify-center py-20"><p className="text-sm text-neutral-400">{t('forms.loading')}</p></div>
  if (!person) return <div className="px-8 py-6"><p className="text-neutral-500">{t('people.notFound')}</p></div>

  const assignedProjects = person.projects ?? []
  const linkedAgents = person.linkedAgents ?? []
  const unassigned = allProjects.filter(p => !linkedAgents.some(la => la.startsWith(p.name + '/')))

  const pagedMsgs = msgs.slice((msgPage - 1) * PAGE_SIZE, msgPage * PAGE_SIZE)
  const pagedTasks = tasks.slice((taskPage - 1) * PAGE_SIZE, taskPage * PAGE_SIZE)

  return (
    <div className="animate-fade-in px-8 py-6">
      {/* Header */}
      <div className="flex items-start gap-4 pb-6">
        {person.avatar ? (
          <img src={person.avatar} alt="" className="size-14 shrink-0 rounded-full object-cover" />
        ) : (
          <div className="flex size-14 shrink-0 items-center justify-center rounded-full bg-indigo-100 dark:bg-indigo-900/30">
            <User className="size-7 text-indigo-600 dark:text-indigo-400" strokeWidth={1.5} />
          </div>
        )}
        <div className="min-w-0 flex-1">
          <h1 className="text-xl font-semibold text-neutral-900 dark:text-zinc-100">
            {person.displayName || person.username}
          </h1>
          {person.displayName && <p className="text-sm text-neutral-400 dark:text-zinc-500">@{person.username}</p>}
          <div className="mt-1.5 flex flex-wrap items-center gap-3 text-xs text-neutral-400 dark:text-zinc-500">
            {person.email && <span className="flex items-center gap-1"><AtSign className="size-3" />{person.email}</span>}
            {person.phone && <span className="flex items-center gap-1"><Phone className="size-3" />{person.phone}</span>}
            {person.disabled && <span className="rounded-full bg-red-100 px-2 py-0.5 text-[10px] font-medium text-red-600 dark:bg-red-900/30 dark:text-red-400">{t('users.disabled')}</span>}
            {person.createdAt && <span>{t('people.createdAt')}: {new Date(person.createdAt).toLocaleDateString()}</span>}
          </div>
          {person.bio && <p className="mt-1.5 text-sm text-neutral-500 dark:text-zinc-400">{person.bio}</p>}
        </div>
      </div>

      {/* Assigned Projects */}
      <section className="mb-6 rounded-xl border border-neutral-200/80 bg-white p-5 dark:border-zinc-700/60 dark:bg-zinc-900/40">
        <div className="flex items-center justify-between pb-3">
          <div className="flex items-center gap-2">
            <FolderKanban className="size-4 text-neutral-500 dark:text-zinc-500" strokeWidth={1.8} />
            <h3 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('people.assignedProjects')}</h3>
          </div>
          {unassigned.length > 0 && (
            <button type="button" onClick={() => setAssigning(true)}
              className="flex items-center gap-1 rounded-lg bg-sky-600 px-3 py-1.5 text-xs font-medium text-white transition-colors hover:bg-sky-700">
              <Plus className="size-3.5" /> {t('people.assignToProject')}
            </button>
          )}
        </div>
        {linkedAgents.length === 0 ? (
          <p className="py-4 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('people.noProjects')}</p>
        ) : (
          <div className="space-y-2">
            {linkedAgents.map(la => {
              const [proj, agName] = la.split('/')
              const pa = assignedProjects.find(p => p.project === proj)
              return (
                <Link key={la} to={`/projects/${encodeURIComponent(proj!)}/members/${encodeURIComponent(agName!)}`}
                  className="flex items-center justify-between rounded-lg border border-neutral-200/80 bg-neutral-50/30 px-4 py-2.5 transition-colors hover:border-sky-300 dark:border-zinc-700/60 dark:bg-zinc-800/30 dark:hover:border-sky-600/40">
                  <div className="flex flex-col">
                    <span className="text-sm font-medium text-neutral-800 dark:text-zinc-200">{proj}</span>
                    <span className="text-xs text-neutral-400 dark:text-zinc-500">{la}{pa ? ` · ${pa.role}` : ''}</span>
                  </div>
                  <ChevronRight className="size-4 text-neutral-300 dark:text-zinc-600" />
                </Link>
              )
            })}
          </div>
        )}
      </section>

      {/* Tabs: Messages | Tasks */}
      <div className="mb-4 flex gap-1 rounded-lg bg-neutral-100/80 p-1 dark:bg-zinc-800/60">
        {(['messages', 'tasks'] as const).map(t2 => (
          <button key={t2} type="button" onClick={() => { setTab(t2); t2 === 'messages' ? setMsgPage(1) : setTaskPage(1) }}
            className={cn('flex items-center gap-1.5 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
              tab === t2 ? 'bg-white text-neutral-900 shadow-sm dark:bg-zinc-700 dark:text-zinc-100' : 'text-neutral-500 hover:text-neutral-700 dark:text-zinc-400')}>
            {t2 === 'messages' ? <Mail className="size-3.5" /> : <ListTodo className="size-3.5" />}
            {t(`people.tab_${t2}`)} <span className="ml-0.5 text-xs text-neutral-400 dark:text-zinc-500">({t2 === 'messages' ? msgs.length : tasks.length})</span>
          </button>
        ))}
      </div>

      {/* Messages */}
      {tab === 'messages' && (
        <div className="space-y-2">
          {pagedMsgs.length === 0 ? (
            <p className="py-8 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('people.noMessages')}</p>
          ) : pagedMsgs.map(m => {
            const isExpanded = expandedMsg === m.id
            const isReplying = replyingTo === m.id
            return (
              <div key={m.id} className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
                <button type="button" onClick={() => setExpandedMsg(isExpanded ? null : m.id)}
                  className="flex w-full items-center gap-3 px-4 py-3 text-left">
                  <div className={cn('size-2 shrink-0 rounded-full', m.readAt ? 'bg-neutral-300 dark:bg-zinc-600' : 'bg-sky-500')} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium text-neutral-800 dark:text-zinc-200">{m.subject || t('people.noSubject')}</span>
                      <span className="shrink-0 text-xs text-neutral-400 dark:text-zinc-500">{m.mailbox.split('/')[0]}</span>
                    </div>
                    <p className="truncate text-xs text-neutral-400 dark:text-zinc-500">
                      {m.from} → {m.to} · {new Date(m.sentAt).toLocaleString()}
                    </p>
                  </div>
                  <ChevronDown className={cn('size-4 shrink-0 text-neutral-400 transition-transform', isExpanded && 'rotate-180')} />
                </button>
                {isExpanded && (
                  <div className="border-t border-neutral-200/60 px-4 py-3 dark:border-zinc-700/40">
                    <div className="prose prose-sm max-w-none dark:prose-invert">
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>{m.body}</ReactMarkdown>
                    </div>
                    <div className="mt-3 flex items-center gap-2">
                      {!m.readAt && (
                        <button type="button" onClick={() => void markRead(m.mailbox, m.id)}
                          className="flex items-center gap-1 rounded-md bg-neutral-100 px-2.5 py-1 text-xs text-neutral-600 hover:bg-neutral-200 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700">
                          <CheckCircle2 className="size-3" /> {t('forms.markAsRead')}
                        </button>
                      )}
                      <button type="button" onClick={() => { setReplyingTo(isReplying ? null : m.id); setReplyText('') }}
                        className="flex items-center gap-1 rounded-md bg-neutral-100 px-2.5 py-1 text-xs text-neutral-600 hover:bg-neutral-200 dark:bg-zinc-800 dark:text-zinc-400 dark:hover:bg-zinc-700">
                        <Reply className="size-3" /> {t('workbench.replyTo')}
                      </button>
                    </div>
                    {isReplying && (
                      <div className="mt-3 flex gap-2">
                        <textarea value={replyText} onChange={e => setReplyText(e.target.value)} rows={2}
                          className={cn(fieldCls, 'flex-1 resize-none text-xs')} placeholder={t('workbench.replyTo')} autoFocus />
                        <button type="button" onClick={() => void sendReply(m)} disabled={replying || !replyText.trim()}
                          className="shrink-0 rounded-md bg-sky-600 px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50">
                          <Send className="size-3.5" />
                        </button>
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
          {msgs.length > PAGE_SIZE && <Pagination current={msgPage} total={Math.ceil(msgs.length / PAGE_SIZE)} onChange={setMsgPage} />}
        </div>
      )}

      {/* Tasks */}
      {tab === 'tasks' && (
        <div className="space-y-2">
          {pagedTasks.length === 0 ? (
            <p className="py-8 text-center text-sm text-neutral-400 dark:text-zinc-500">{t('people.noTasks')}</p>
          ) : pagedTasks.map(tk => {
            const isExpanded = expandedTask === tk.id
            return (
              <div key={tk.id} className="rounded-lg border border-neutral-200/80 bg-white dark:border-zinc-700/60 dark:bg-zinc-900/40">
                <button type="button" onClick={() => setExpandedTask(isExpanded ? null : tk.id)}
                  className="flex w-full items-center gap-3 px-4 py-3 text-left">
                  <span className={cn('inline-block size-2 shrink-0 rounded-full',
                    tk.status === 'done_success' ? 'bg-emerald-500' :
                    tk.status === 'done_failed' ? 'bg-red-500' :
                    tk.status === 'pending' ? 'bg-amber-500' : 'bg-sky-500'
                  )} />
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="truncate text-sm font-medium text-neutral-800 dark:text-zinc-200">{tk.title}</span>
                      <span className="shrink-0 rounded bg-neutral-100 px-1.5 py-0.5 text-[10px] dark:bg-zinc-800">{tk.status}</span>
                    </div>
                    <p className="truncate text-xs text-neutral-400 dark:text-zinc-500">
                      {tk.project}/{tk.agent} · {new Date(tk.updatedAt).toLocaleString()}
                    </p>
                  </div>
                  <ChevronDown className={cn('size-4 shrink-0 text-neutral-400 transition-transform', isExpanded && 'rotate-180')} />
                </button>
                {isExpanded && tk.prompt && (
                  <div className="border-t border-neutral-200/60 px-4 py-3 dark:border-zinc-700/40">
                    <div className="prose prose-sm max-w-none dark:prose-invert">
                      <ReactMarkdown remarkPlugins={[remarkGfm]}>{tk.prompt}</ReactMarkdown>
                    </div>
                  </div>
                )}
              </div>
            )
          })}
          {tasks.length > PAGE_SIZE && <Pagination current={taskPage} total={Math.ceil(tasks.length / PAGE_SIZE)} onChange={setTaskPage} />}
        </div>
      )}

      {/* Assign to Project Dialog */}
      {assigning && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/45 p-4" onClick={() => !assignBusy && setAssigning(false)}>
          <div className="w-full max-w-md rounded-xl border border-neutral-200 bg-white shadow-lg dark:border-zinc-700 dark:bg-zinc-900 animate-scale-in" onClick={e => e.stopPropagation()}>
            <div className="flex items-center justify-between border-b border-neutral-200 px-5 py-3 dark:border-zinc-700">
              <h2 className="text-base font-semibold text-neutral-900 dark:text-zinc-100">{t('people.assignToProject')}</h2>
              <button type="button" onClick={() => setAssigning(false)} className="rounded-md p-1 text-neutral-400 hover:bg-neutral-100 dark:hover:bg-zinc-800"><X className="size-4" /></button>
            </div>
            <div className="space-y-3 px-5 py-4">
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('people.selectProject')} *</span>
                <select value={assignProject} onChange={e => setAssignProject(e.target.value)} className={fieldCls}>
                  <option value="">{t('people.selectProject')}</option>
                  {unassigned.map(p => <option key={p.name} value={p.name}>{p.name}</option>)}
                </select>
              </label>
              <label className="flex flex-col gap-1">
                <span className="text-sm font-medium text-neutral-600 dark:text-zinc-400">{t('members.team')} *</span>
                <select value={assignTeam} onChange={e => setAssignTeam(e.target.value)} className={fieldCls}>
                  <option value="">{t('members.selectTeam')}</option>
                  {teams.map(te => <option key={te.path} value={te.path}>{te.name} ({te.path})</option>)}
                </select>
              </label>
              <p className="text-xs text-neutral-400 dark:text-zinc-500">{t('people.assignHint')}</p>
              <div className="flex justify-end gap-2 pt-1">
                <button type="button" onClick={() => setAssigning(false)} disabled={assignBusy} className="rounded-lg border border-neutral-300 px-3 py-1.5 text-sm dark:border-zinc-600">{t('forms.cancel')}</button>
                <button type="button" onClick={() => void handleAssign()} disabled={assignBusy || !assignProject || !assignTeam}
                  className="rounded-lg bg-sky-600 px-3 py-1.5 text-sm font-medium text-white disabled:opacity-50">
                  {assignBusy ? t('members.hiring') : t('people.assign')}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
