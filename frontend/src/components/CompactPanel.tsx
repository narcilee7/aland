// Compact 事件面板——session 中上下文压缩的时刻标记。
//
// 设计：把每个 compact 事件渲染成时间线上的一个标记点。
// 显示 trigger (manual/auto) 和压缩前的 token 数。

import {useEffect, useState} from 'react'
import {Card, CardContent, CardHeader, CardTitle} from './ui'
import {listCompactEvents} from '../api/wails'
import type {CompactEvent} from '../api/wails'
import {Zap, Hand, RotateCw, FileText} from 'lucide-react'

interface CompactPanelProps {
  tribeId: string
  sessionId: string | null
}

export function CompactPanel({tribeId, sessionId}: CompactPanelProps) {
  const [events, setEvents] = useState<CompactEvent[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = async () => {
    if (!sessionId) return
    setLoading(true)
    try {
      const e = await listCompactEvents(tribeId, sessionId)
      setEvents(e ?? [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Zap className="h-4 w-4" />
          Compact Events
          {events.length > 0 && (
            <span className="text-ink-faint normal-case tracking-wider text-[10px]">
              {events.length}
            </span>
          )}
          <button
            onClick={refresh}
            disabled={!sessionId}
            className="ml-auto text-ink-faint hover:text-ink-dim disabled:opacity-30"
            title="Refresh"
          >
            <RotateCw className={`h-3 w-3 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-auto min-h-0">
        {!sessionId ? (
          <div className="text-xs text-ink-faint font-mono py-6 text-center">
            选择一个 session 查看 compact 事件
          </div>
        ) : events.length === 0 ? (
          <div className="text-xs text-ink-faint font-mono py-6 text-center">
            <FileText className="inline h-3 w-3 mr-1" />
            暂无 compact（上下文还没被压缩过）
          </div>
        ) : (
          <div className="space-y-2">
            {events.map((e, i) => (
              <CompactRow key={i} event={e} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function CompactRow({event}: {event: CompactEvent}) {
  const isManual = event.trigger === 'manual'
  const Icon = isManual ? Hand : Zap
  const label = isManual ? 'manual' : 'auto'
  const color = isManual ? 'text-forge-amber' : 'text-forge-green'
  return (
    <div className="flex items-start gap-2 px-2 py-1.5 rounded bg-white/[0.02] border border-white/5">
      <Icon className={`h-3 w-3 ${color} shrink-0 mt-0.5`} />
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 text-xs font-mono">
          <span className={`${color} uppercase tracking-wider text-[10px]`}>[{label}]</span>
          <span className="text-ink">
            {event.preTokens > 0 ? `${formatNumber(event.preTokens)} tokens` : 'unknown size'}
          </span>
        </div>
        <div className="text-[10px] text-ink-faint font-mono mt-0.5">
          {event.at > 0 ? new Date(event.at).toLocaleString() : ''}
        </div>
      </div>
    </div>
  )
}

function formatNumber(n: number): string {
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1_000) return `${(n / 1_000).toFixed(1)}k`
  return String(n)
}