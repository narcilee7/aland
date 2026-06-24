// 部落内部视图——M1 版本。
// 真实数据：ConfigDNA（结构化三层）+ SessionShards（记忆碎片）。
// 完整版（M2+）会有配置双螺旋编辑、记忆墙点击展开。

import {useEffect, useMemo, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {
  listSessions,
  readTribeConfigDNA,
  type ConfigDNA,
  type ConfigItem,
  type SessionShard,
  type Tribe,
} from '../api/wails'
import {ArrowLeft, Layers, ScrollText, Eye, EyeOff} from 'lucide-react'
import {Badge, Button, Card, CardContent, CardHeader, CardTitle, Separator} from './ui'

export function TribeView() {
  const activeTribe = useAland(s => s.activeTribe)
  const returnOverlook = useAland(s => s.returnOverlook)
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)

  const [detail, setDetail] = useState<Tribe | null>(null)
  const [dna, setDna] = useState<ConfigDNA | null>(null)
  const [sessions, setSessions] = useState<SessionShard[]>([])
  const [showSensitive, setShowSensitive] = useState(false)

  useEffect(() => {
    if (!activeTribe) return
    Promise.all([readTribeConfigDNA(activeTribe), listSessions(activeTribe)]).then(([d, s]) => {
      setDna(d)
      setSessions(s ?? [])
    })
    setDetail(tribes[activeTribe] ?? null)
  }, [activeTribe, tribes])

  if (!activeTribe) return null
  const liveTribe = tribes[activeTribe] || detail
  const liveMeta = meta[activeTribe]
  if (!liveMeta) return null

  return (
    <div
      className="absolute inset-0 p-12 flex flex-col gap-6 backdrop-blur-md"
      style={
        {
          background:
            'radial-gradient(ellipse at center, rgba(13,31,21,0.6) 0%, rgba(10,14,26,0.95) 70%)',
          ['--tribe-theme' as string]: liveMeta.themeColor,
          ['--tribe-accent' as string]: liveMeta.accentColor,
        } as React.CSSProperties
      }
    >
      {/* 顶部 */}
      <div className="draggable flex items-center justify-between">
        <Button onClick={returnOverlook} variant="outline" size="md">
          <ArrowLeft className="h-3 w-3" />
          Back to Overlook
        </Button>
        <h1 className="m-0 font-mono text-lg font-medium uppercase tracking-widest text-tribe">
          {liveMeta.name}
        </h1>
        <div className="w-[120px] flex justify-end">
          {liveTribe && <Badge status={liveTribe.status} />}
        </div>
      </div>

      <Separator />

      {/* Vital */}
      <div className="grid grid-cols-4 gap-4 font-mono">
        <VitalCard label="Status" value={liveTribe?.status ?? '—'} />
        <VitalCard label="PID" value={String(liveTribe?.vital.pid ?? '—')} />
        <VitalCard label="CPU" value={`${(liveTribe?.vital.cpu ?? 0).toFixed(1)}%`} highlight={cpuHighlight(liveTribe?.vital.cpu)} />
        <VitalCard
          label="Memory"
          value={`${(liveTribe?.vital.memory ?? 0).toFixed(0)}MB`}
        />
      </div>

      {/* 双栏：Config DNA + Shards */}
      <div className="grid grid-cols-2 gap-4 flex-1 min-h-0">
        {/* Config DNA */}
        <Card className="flex flex-col min-h-0">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="flex items-center gap-2">
              <Layers className="h-3 w-3" />
              Config · Three Layers
            </CardTitle>
            {dna?.middle.some(i => i.sensitive) && (
              <Button
                size="sm"
                variant="ghost"
                onClick={() => setShowSensitive(s => !s)}
                title={showSensitive ? '隐藏敏感字段' : '显示敏感字段'}
              >
                {showSensitive ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
              </Button>
            )}
          </CardHeader>
          <CardContent className="flex-1 overflow-auto space-y-3 text-xs font-mono">
            {dna ? (
              <>
                <ConfigLayer title="Surface · 运行时" items={dna.surface} showSensitive={showSensitive} />
                <ConfigLayer title="Middle · API & 权限" items={dna.middle} showSensitive={showSensitive} />
                <ConfigLayer title="Deep · 元信息" items={dna.deep} showSensitive={showSensitive} />
                <div className="text-[10px] text-ink-faint pt-2 border-t border-white/5">
                  source: {dna.source}
                </div>
              </>
            ) : (
              <div className="text-ink-faint">加载中…</div>
            )}
          </CardContent>
        </Card>

        {/* Shards */}
        <Card className="flex flex-col min-h-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ScrollText className="h-3 w-3" />
              Memory Shards ·{' '}
              <span className="text-tribe">{sessions.length}</span>
            </CardTitle>
          </CardHeader>
          <CardContent className="flex-1 overflow-auto space-y-2">
            {sessions.length === 0 ? (
              <div className="text-ink-faint text-xs font-mono py-4 text-center">
                暂无会话 · 跑一个 CLI 试试
              </div>
            ) : (
              sessions.slice(0, 30).map(s => <ShardRow key={s.id} shard={s} />)
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function cpuHighlight(cpu: number | undefined): boolean {
  return (cpu ?? 0) > 70
}

function VitalCard({
  label,
  value,
  highlight,
}: {
  label: string
  value: string
  highlight?: boolean
}) {
  return (
    <Card>
      <CardContent className="py-4">
        <div className="text-[10px] uppercase tracking-wider text-ink-faint mb-2 font-mono">
          {label}
        </div>
        <div
          className={`text-lg font-medium font-mono ${
            highlight ? 'text-forge-amber' : 'text-tribe'
          }`}
        >
          {value}
        </div>
      </CardContent>
    </Card>
  )
}

function ConfigLayer({
  title,
  items,
  showSensitive,
}: {
  title: string
  items: ConfigItem[]
  showSensitive: boolean
}) {
  if (items.length === 0) return null
  return (
    <div>
      <div className="text-[10px] text-ink-faint uppercase tracking-wider mb-1.5">{title}</div>
      <div className="space-y-1">
        {items.map(item => (
          <div key={item.key} className="flex items-start gap-2 leading-relaxed">
            <span className="text-ink-dim shrink-0">{item.key}</span>
            <span className="text-ink-faint shrink-0">=</span>
            <span className="text-ink break-all">
              {item.sensitive && !showSensitive
                ? maskValue(item.value)
                : JSON.stringify(item.value)}
            </span>
          </div>
        ))}
      </div>
    </div>
  )
}

function maskValue(_v: unknown): string {
  return '•••••••• (hidden)'
}

function ShardRow({shard}: {shard: SessionShard}) {
  const ts = useMemo(() => new Date(shard.timestamp), [shard.timestamp])
  return (
    <div
      className="group rounded border border-white/5 bg-white/[0.02] p-2 hover:bg-white/[0.05] transition-colors cursor-default"
      title={shard.filePath}
    >
      <div className="flex items-center gap-2 mb-1">
        <span className="text-[10px] text-ink-faint font-mono">
          {ts.toLocaleDateString()} {ts.toLocaleTimeString([], {hour: '2-digit', minute: '2-digit'})}
        </span>
        {shard.model && (
          <span className="text-[10px] text-ink-faint font-mono truncate max-w-[120px]">
            {shard.model}
          </span>
        )}
        <span className="ml-auto text-[10px] text-ink-faint font-mono">
          {shard.messageCount}msg · {(shard.tokenCount / 1000).toFixed(1)}k tok
        </span>
      </div>
      <div className="text-xs text-ink-dim line-clamp-2">{shard.summary}</div>
    </div>
  )
}
