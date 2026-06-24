// Insights 面板——Claude 19 项能力的前 5 个 features 的统一入口。
// 用 Tabs 切换：MCP / Skills / Plugins / Plans / FileHistory。
// stats / slash commands / live tail 在 LiveTab 单独展示。

import { useEffect, useState } from 'react'
import { useAland } from '../stores/alandStore'
import {
  listMCPServers,
  listSkills,
  listPlans,
  listFileHistory,
  listPlugins,
  restoreFile,
  listDailyActivity,
  recentSlashCommands,
  listModelTokenUsage,
  type MCPServer,
  type Skill,
  type PlanFile,
  type FileEdit,
  type Plugin,
  type DailyActivity,
  type SlashCommand,
  type ModelTokenUsage,
} from '../api/wails'
import { onSessionEvent } from '../api/events'
import { streamLatestSession, stopLatestSession } from '../api/wails'
import { Card, CardContent, CardHeader, CardTitle, Button, Badge } from './ui'
import { Plug, Sparkles, Wrench, FileText, History, RotateCcw, Activity, Terminal, Play, Square, Coins, Hammer } from 'lucide-react'
import { ToolChain } from './ToolChain'
import { logger } from '../lib/logger'

interface InsightsProps {
  tribeId: string
  caps: {
    sessions: boolean
    sessionTail: boolean
    tokens: boolean
    tokensLive: boolean
  }
}

export function Insights({ tribeId, caps }: InsightsProps) {
  const [tab, setTab] = useState<'mcp' | 'skills' | 'plugins' | 'plans' | 'history' | 'activity' | 'tail' | 'chain'>('mcp')

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Insights
          <span className="text-ink-faint normal-case tracking-wider text-[10px]">
            Claude 19/19
          </span>
        </CardTitle>
        <div className="flex gap-1 mt-2 -mb-2 flex-wrap">
          <TabBtn active={tab === 'mcp'} onClick={() => setTab('mcp')}>
            <Plug className="h-3 w-3" />
            MCP
          </TabBtn>
          <TabBtn active={tab === 'skills'} onClick={() => setTab('skills')}>
            <Sparkles className="h-3 w-3" />
            Skills
          </TabBtn>
          <TabBtn active={tab === 'plugins'} onClick={() => setTab('plugins')}>
            <Wrench className="h-3 w-3" />
            Plugins
          </TabBtn>
          <TabBtn active={tab === 'plans'} onClick={() => setTab('plans')}>
            <FileText className="h-3 w-3" />
            Plans
          </TabBtn>
          <TabBtn active={tab === 'history'} onClick={() => setTab('history')}>
            <History className="h-3 w-3" />
            History
          </TabBtn>
          <TabBtn active={tab === 'activity'} onClick={() => setTab('activity')}>
            <Activity className="h-3 w-3" />
            Activity
          </TabBtn>
          {caps.sessionTail && (
            <TabBtn active={tab === 'tail'} onClick={() => setTab('tail')}>
              <Terminal className="h-3 w-3" />
              Live
            </TabBtn>
          )}
          <TabBtn active={tab === 'chain'} onClick={() => setTab('chain')}>
            <Hammer className="h-3 w-3" />
            Chain
          </TabBtn>
        </div>
      </CardHeader>
      <CardContent className="flex-1 overflow-auto min-h-0">
        {tab === 'mcp' && <MCPTab tribeId={tribeId} />}
        {tab === 'skills' && <SkillsTab tribeId={tribeId} />}
        {tab === 'plugins' && <PluginsTab tribeId={tribeId} />}
        {tab === 'plans' && <PlansTab tribeId={tribeId} />}
        {tab === 'history' && <HistoryTab tribeId={tribeId} />}
        {tab === 'activity' && <ActivityTab tribeId={tribeId} />}
        {tab === 'tail' && <LiveTailTab tribeId={tribeId} />}
        {tab === 'chain' && <ToolChain />}
      </CardContent>
    </Card>
  )
}

function TabBtn({ active, onClick, children }: { active: boolean; onClick: () => void; children: React.ReactNode }) {
  return (
    <button
      onClick={onClick}
      className={`px-2 py-1 rounded text-[10px] font-mono uppercase tracking-wider inline-flex items-center gap-1 transition-colors ${active
          ? 'bg-tribe/20 text-tribe'
          : 'text-ink-faint hover:text-ink hover:bg-white/5'
        }`}
    >
      {children}
    </button>
  )
}

function EmptyState({ text }: { text: string }) {
  return <div className="text-ink-faint text-xs font-mono py-6 text-center">{text}</div>
}

// —— MCP Tab ——

function MCPTab({ tribeId }: { tribeId: string }) {
  const [items, setItems] = useState<MCPServer[]>([])
  useEffect(() => {
    listMCPServers(tribeId).then(setItems).catch(e => logger.error('mcp load failed', e))
  }, [tribeId])

  if (items.length === 0) return <EmptyState text="暂无 MCP server 配置" />
  return (
    <div className="space-y-2">
      {items.map(s => (
        <div key={s.name} className="rounded border border-white/5 bg-white/[0.02] p-2.5">
          <div className="flex items-center gap-2 mb-1">
            <span className="text-sm text-ink font-mono">{s.name}</span>
            <Badge status="running" />
            <span className="ml-auto text-[10px] text-ink-faint font-mono">{s.transport || 'stdio'}</span>
          </div>
          <div className="text-[11px] text-ink-dim font-mono break-all">
            {s.command} {s.args?.join(' ')}
          </div>
          <div className="text-[10px] text-ink-faint font-mono mt-1">from: {s.source}</div>
        </div>
      ))}
    </div>
  )
}

// —— Skills Tab ——

function SkillsTab({ tribeId }: { tribeId: string }) {
  const [items, setItems] = useState<Skill[]>([])
  useEffect(() => {
    listSkills(tribeId).then(setItems).catch(e => logger.error('skills load failed', e))
  }, [tribeId])

  if (items.length === 0) return <EmptyState text="~/.claude/skills/ 为空" />
  return (
    <div className="space-y-2">
      {items.map(s => (
        <div key={s.name} className="rounded border border-white/5 bg-white/[0.02] p-2.5">
          <div className="text-sm text-ink font-mono mb-1">/{s.name}</div>
          {s.description && <div className="text-[11px] text-ink-dim mb-1">{s.description}</div>}
          <div className="text-[10px] text-ink-faint font-mono truncate" title={s.path}>
            {s.path}
          </div>
        </div>
      ))}
    </div>
  )
}

// —— Plugins Tab ——

function PluginsTab({ tribeId }: { tribeId: string }) {
  const [items, setItems] = useState<Plugin[]>([])
  useEffect(() => {
    listPlugins(tribeId).then(setItems).catch(e => logger.error('plugins load failed', e))
  }, [tribeId])

  if (items.length === 0) return <EmptyState text="settings.enabledPlugins 为空" />
  return (
    <div className="space-y-1.5">
      {items.map(p => (
        <div key={p.name} className="flex items-center gap-2 rounded border border-white/5 bg-white/[0.02] px-2.5 py-1.5">
          <Wrench className="h-3 w-3 text-ink-faint" />
          <span className="text-sm text-ink font-mono flex-1">{p.name}</span>
          <Badge status={p.enabled ? 'running' : 'idle'} />
        </div>
      ))}
    </div>
  )
}

// —— Plans Tab ——

function PlansTab({ tribeId }: { tribeId: string }) {
  const [items, setItems] = useState<PlanFile[]>([])
  useEffect(() => {
    listPlans(tribeId).then(setItems).catch(e => logger.error('plans load failed', e))
  }, [tribeId])

  if (items.length === 0) return <EmptyState text="~/.claude/plans/ 为空" />
  return (
    <div className="space-y-2">
      {items.map(p => (
        <div key={p.path} className="rounded border border-white/5 bg-white/[0.02] p-2.5">
          <div className="flex items-center gap-2 mb-1">
            <FileText className="h-3 w-3 text-ink-faint" />
            <span className="text-sm text-ink font-mono">{p.name}</span>
            <span className="ml-auto text-[10px] text-ink-faint font-mono">
              {new Date(p.modifiedAt * 1000).toLocaleString()}
            </span>
          </div>
          <div className="text-[11px] text-ink-dim line-clamp-3 whitespace-pre-wrap">{p.summary}</div>
        </div>
      ))}
    </div>
  )
}

// —— History Tab (with Restore) ——

function HistoryTab({ tribeId }: { tribeId: string }) {
  const [items, setItems] = useState<FileEdit[]>([])
  useEffect(() => {
    listFileHistory(tribeId).then(setItems).catch(e => logger.error('history load failed', e))
  }, [tribeId])

  const onRestore = async (e: FileEdit) => {
    if (!confirm(`Restore ${e.path} to backup? This will overwrite the current file.`)) return
    try {
      await restoreFile(tribeId, e)
      logger.info('file restored', { path: e.path })
    } catch (err) {
      logger.error('restore failed', err)
    }
  }

  if (items.length === 0) return <EmptyState text="~/.claude/file-history/ 为空" />
  return (
    <div className="space-y-1.5">
      {items.map((e, i) => (
        <div
          key={`${e.backupPath}-${i}`}
          className="flex items-center gap-2 rounded border border-white/5 bg-white/[0.02] px-2.5 py-1.5 group"
        >
          <History className="h-3 w-3 text-ink-faint shrink-0" />
          <div className="flex-1 min-w-0">
            <div className="text-sm text-ink font-mono truncate" title={e.path}>
              {e.path}
            </div>
            <div className="text-[10px] text-ink-faint font-mono">
              v{e.version} · {new Date(e.timestamp * 1000).toLocaleString()}
            </div>
          </div>
          <Button size="sm" variant="ghost" onClick={() => onRestore(e)} title="Restore this version">
            <RotateCcw className="h-3 w-3" />
          </Button>
        </div>
      ))}
    </div>
  )
}

// —— Activity Tab (Daily + Model Usage + Slash Commands) ——

function ActivityTab({ tribeId }: { tribeId: string }) {
  const [daily, setDaily] = useState<DailyActivity[]>([])
  const [models, setModels] = useState<ModelTokenUsage[]>([])
  const [slash, setSlash] = useState<SlashCommand[]>([])
  useEffect(() => {
    Promise.all([
      listDailyActivity(tribeId),
      listModelTokenUsage(tribeId),
      recentSlashCommands(tribeId, 20),
    ]).then(([d, m, s]) => {
      setDaily(d ?? [])
      setModels(m ?? [])
      setSlash(s ?? [])
    })
  }, [tribeId])

  return (
    <div className="space-y-4">
      <Section icon={<Activity className="h-3 w-3" />} title="Daily Activity (last 14d)">
        {daily.length === 0 ? (
          <EmptyState text="暂无数据" />
        ) : (
          <div className="space-y-1">
            {daily.slice(0, 14).map(d => (
              <div key={d.date} className="flex items-center gap-2 text-xs font-mono">
                <span className="text-ink-faint w-20">{d.date}</span>
                <span className="text-ink flex-1">
                  {d.messageCount} msg · {d.sessionCount} sess · {d.toolCallCount} tools
                </span>
              </div>
            ))}
          </div>
        )}
      </Section>

      <Section icon={<Coins className="h-3 w-3" />} title="Token by Model (recent)">
        {models.length === 0 ? (
          <EmptyState text="暂无数据" />
        ) : (
          <div className="space-y-1">
            {models.slice(0, 20).map(m => {
              const total = m.inputTokens + m.outputTokens
              return (
                <div key={`${m.model}-${m.date}`} className="flex items-center gap-2 text-xs font-mono">
                  <span className="text-ink-faint w-20">{m.date}</span>
                  <span className="text-ink-dim flex-1 truncate">{m.model}</span>
                  <span className="text-ink">
                    {(total / 1000).toFixed(1)}k ({m.inputTokens / 1000 | 0}k in /{' '}
                    {m.outputTokens / 1000 | 0}k out)
                  </span>
                </div>
              )
            })}
          </div>
        )}
      </Section>

      <Section icon={<Terminal className="h-3 w-3" />} title="Recent Slash Commands">
        {slash.length === 0 ? (
          <EmptyState text="暂无 history.jsonl" />
        ) : (
          <div className="space-y-1">
            {slash.map(c => (
              <div key={c.timestamp} className="flex items-center gap-2 text-xs font-mono">
                <span className="text-ink-faint w-32 shrink-0">
                  {new Date(c.timestamp).toLocaleString()}
                </span>
                <span className="text-ink-dim flex-1 truncate">
                  {c.command} {c.args}
                </span>
              </div>
            ))}
          </div>
        )}
      </Section>
    </div>
  )
}

function Section({ icon, title, children }: { icon: React.ReactNode; title: string; children: React.ReactNode }) {
  return (
    <div>
      <div className="text-[10px] text-ink-faint font-mono uppercase tracking-wider mb-2 flex items-center gap-1.5">
        {icon}
        {title}
      </div>
      {children}
    </div>
  )
}

// —— Live Tail Tab (killer) ——

interface LiveEvent {
  ts: number
  type: string
  text: string
}

function LiveTailTab({ tribeId }: { tribeId: string }) {
  const [streaming, setStreaming] = useState(false)
  const [events, setEvents] = useState<LiveEvent[]>([])
  const [totalTokens, setTotalTokens] = useState({ input: 0, output: 0 })

  useEffect(() => {
    return onSessionEvent(ev => {
      const text = formatEvent(ev)
      if (!text) return
      setEvents(prev => [...prev.slice(-200), { ts: ev.timestamp, type: ev.type, text }])
      if (ev.tokens) {
        setTotalTokens(prev => ({
          input: prev.input + ev.tokens!.input,
          output: prev.output + ev.tokens!.output,
        }))
      }
    })
  }, [])

  const start = async () => {
    setEvents([])
    setTotalTokens({ input: 0, output: 0 })
    try {
      await streamLatestSession(tribeId)
      setStreaming(true)
    } catch (e) {
      logger.error('stream start failed', e)
    }
  }

  const stop = async () => {
    try {
      await stopLatestSession(tribeId)
      setStreaming(false)
    } catch (e) {
      logger.error('stream stop failed', e)
    }
  }

  return (
    <div className="flex flex-col h-full min-h-0">
      <div className="flex items-center gap-2 mb-3">
        {!streaming ? (
          <Button size="sm" variant="default" onClick={start}>
            <Play className="h-3 w-3" />
            Start Live Tail
          </Button>
        ) : (
          <Button size="sm" variant="outline" onClick={stop}>
            <Square className="h-3 w-3" />
            Stop
          </Button>
        )}
        <span className="text-[10px] text-ink-faint font-mono ml-2">
          {streaming ? '● STREAMING' : '○ idle'}
        </span>
        <span className="text-[10px] text-ink-faint font-mono ml-auto">
          tokens: {totalTokens.input} in / {totalTokens.output} out
        </span>
      </div>
      <div className="flex-1 overflow-auto bg-land-1/40 rounded border border-white/5 p-2 space-y-1 font-mono text-[11px]">
        {events.length === 0 ? (
          <div className="text-ink-faint text-center py-8">
            {streaming ? 'waiting for events…' : '点击 Start Live Tail 启动'}
          </div>
        ) : (
          events.map((e, i) => (
            <div key={i} className="flex gap-2">
              <span className="text-ink-faint shrink-0 w-16">
                {e.ts ? new Date(e.ts).toLocaleTimeString() : ''}
              </span>
              <span className="text-tribe shrink-0 w-16">[{e.type}]</span>
              <span className="text-ink-dim break-all flex-1 whitespace-pre-wrap">{e.text}</span>
            </div>
          ))
        )}
      </div>
    </div>
  )
}

function formatEvent(ev: any): string {
  if (ev.content) return ev.content
  if (ev.thinking) return `💭 ${ev.thinking.slice(0, 200)}`
  if (ev.tool) return `🔧 ${ev.tool.name} (${ev.tool.status || 'pending'})`
  if (ev.tokens) return `+${ev.tokens.input} in / +${ev.tokens.output} out`
  if (ev.error) return `❌ ${ev.error}`
  return ''
}
