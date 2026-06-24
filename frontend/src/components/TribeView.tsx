// 部落内部视图——能力感知版本。
//
// 渲染规则：
//   - eco === "ide" → workspace 面板（Process + Workspace + Config）
//   - 否则按 capabilities 决定面板：
//     - Process vital 永远在顶部
//     - Config：cap.config
//     - Sessions：cap.sessions
//     - Tokens：cap.tokens
//     - Features：cap.features 里有 hasData=true 的
//   - 不支持的能力**完全隐藏**，不显示"即将推出"占位

import {useEffect, useMemo, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {
  listSessions,
  readTribeConfigDNA,
  writeTribeConfig,
  type Capabilities,
  type ConfigDNA,
  type ConfigItem,
  type SessionShard,
  type Tribe,
} from '../api/wails'
import {ArrowLeft, Layers, ScrollText, Eye, EyeOff, Save, Check, Cpu} from 'lucide-react'
import {Badge, Button, Card, CardContent, CardHeader, CardTitle, Separator} from './ui'
import {Insights} from './Insights'

export function TribeView() {
  const activeTribe = useAland(s => s.activeTribe)
  const returnOverlook = useAland(s => s.returnOverlook)
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)
  const capsMap = useAland(s => s.caps)

  const [dna, setDna] = useState<ConfigDNA | null>(null)
  const [dnaDraft, setDnaDraft] = useState<ConfigDNA | null>(null)
  const [sessions, setSessions] = useState<SessionShard[]>([])
  const [showSensitive, setShowSensitive] = useState(false)
  const [saved, setSaved] = useState(false)

  useEffect(() => {
    if (!activeTribe) return
    if (capsMap[activeTribe]?.config) {
      readTribeConfigDNA(activeTribe).then(d => {
        setDna(d)
        setDnaDraft(d)
      })
    }
    if (capsMap[activeTribe]?.sessions) {
      listSessions(activeTribe).then(s => setSessions(s ?? []))
    }
    setSaved(false)
  }, [activeTribe, capsMap])

  if (!activeTribe) return null
  const liveTribe = tribes[activeTribe]
  const liveMeta = meta[activeTribe]
  const caps = capsMap[activeTribe]
  if (!liveMeta || !caps) return null

  const isIDE = liveMeta.eco === 'ide'

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
          {isIDE && (
            <span className="ml-3 text-[10px] text-ink-faint normal-case tracking-wider">
              · IDE workspace
            </span>
          )}
        </h1>
        <div className="w-[120px] flex justify-end">
          {liveTribe && <Badge status={liveTribe.status} />}
        </div>
      </div>

      <Separator />

      {/* Vital 永远在顶部 */}
      {caps.process && (
        <div className="grid grid-cols-4 gap-4 font-mono">
          <VitalCard label="Status" value={liveTribe?.status ?? '—'} />
          <VitalCard label="PID" value={String(liveTribe?.vital.pid ?? '—')} />
          <VitalCard
            label="CPU"
            value={`${(liveTribe?.vital.cpu ?? 0).toFixed(1)}%`}
            highlight={(liveTribe?.vital.cpu ?? 0) > 70}
          />
          <VitalCard label="Memory" value={`${(liveTribe?.vital.memory ?? 0).toFixed(0)}MB`} />
        </div>
      )}

      {/* 内容区：根据 capabilities + eco 分支 */}
      {isIDE ? (
        <IDEWorkspace caps={caps} tribe={liveTribe} />
      ) : (
        <div className="grid grid-cols-2 gap-4 flex-1 min-h-0">
          {/* 左：Config + Sessions（原 CapabilityDrivenPanels 内容） */}
          <CapabilityDrivenPanels
            caps={caps}
            dna={dna}
            dnaDraft={dnaDraft}
            setDnaDraft={setDnaDraft}
            sessions={sessions}
            showSensitive={showSensitive}
            setShowSensitive={setShowSensitive}
            saved={saved}
            onSave={async () => {
              if (!activeTribe || !dnaDraft) return
              const ok = await writeTribeConfig(activeTribe, dnaDraft)
              if (ok) {
                setDna(dnaDraft)
                setSaved(true)
                setTimeout(() => setSaved(false), 2000)
              }
            }}
          />
          {/* 右：Insights（MCP/Skills/Plugins/Plans/History/Activity/Live） */}
          <Insights
            tribeId={activeTribe}
            caps={{
              sessions: caps.sessions,
              sessionTail: caps.sessionTail,
              tokens: caps.tokens,
              tokensLive: caps.tokensLive,
            }}
          />
        </div>
      )}
    </div>
  )
}

function IDEWorkspace({caps, tribe}: {caps: Capabilities; tribe: Tribe | undefined}) {
  return (
    <div className="grid grid-cols-2 gap-4 flex-1 min-h-0">
      <Card>
        <CardHeader>
          <CardTitle>Workspace</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3 text-xs font-mono">
          <Row label="CWD" value={tribe?.vital.cwd || '—'} />
          <Row label="PID" value={String(tribe?.vital.pid || '—')} />
          <Row label="Status" value={tribe?.status || '—'} />
          <div className="pt-3 border-t border-white/5 text-ink-faint text-[10px] uppercase tracking-wider">
            IDE 部落不暴露 Session / Token（无对应数据）
          </div>
        </CardContent>
      </Card>
      {caps.config && (
        <Card>
          <CardHeader>
            <CardTitle>
              <Layers className="inline h-3 w-3 mr-2" />
              Config (read-only)
            </CardTitle>
          </CardHeader>
          <CardContent className="text-[10px] text-ink-faint font-mono">
            <IDENoConfigSupport />
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function IDENoConfigSupport() {
  // 注意：Cursor / Trae 实际上有 ConfigParser，但为了清晰，IDE 面板只显示 workspace
  return <span>使用 Read 模式查看原始 JSON（M2 启用 IDE 适配器的配置编辑）</span>
}

function CapabilityDrivenPanels({
  caps,
  dna,
  dnaDraft,
  setDnaDraft,
  sessions,
  showSensitive,
  setShowSensitive,
  saved,
  onSave,
}: {
  caps: Capabilities
  dna: ConfigDNA | null
  dnaDraft: ConfigDNA | null
  setDnaDraft: (d: ConfigDNA | null) => void
  sessions: SessionShard[]
  showSensitive: boolean
  setShowSensitive: React.Dispatch<React.SetStateAction<boolean>>
  saved: boolean
  onSave: () => void
}) {
  // 决定要渲染的面板
  const showConfig = caps.config
  const showSessions = caps.sessions

  // 没有面板可显示
  if (!showConfig && !showSessions) {
    return (
      <Card>
        <CardContent className="py-12 text-center text-ink-faint text-xs font-mono">
          这个部落声明的能力都还没有 UI（M2 完善）
        </CardContent>
      </Card>
    )
  }

  return (
    <div className={`grid ${showConfig && showSessions ? 'grid-cols-2' : 'grid-cols-1'} gap-4 flex-1 min-h-0`}>
      {showConfig && dnaDraft && (
        <Card className="flex flex-col min-h-0">
          <CardHeader className="flex flex-row items-center justify-between space-y-0">
            <CardTitle className="flex items-center gap-2">
              <Layers className="h-3 w-3" />
              Config · Three Layers
            </CardTitle>
            <div className="flex items-center gap-2">
              {dnaDraft.middle.some(i => i.sensitive) && (
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={() => setShowSensitive(s => !s)}
                  title={showSensitive ? '隐藏敏感字段' : '显示敏感字段'}
                >
                  {showSensitive ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                </Button>
              )}
              {caps.configEdit && (
                <Button size="sm" variant={saved ? 'default' : 'outline'} onClick={onSave}>
                  {saved ? (
                    <>
                      <Check className="h-3 w-3" />
                      Saved
                    </>
                  ) : (
                    <>
                      <Save className="h-3 w-3" />
                      Save
                    </>
                  )}
                </Button>
              )}
            </div>
          </CardHeader>
          <CardContent className="flex-1 overflow-auto space-y-3 text-xs font-mono">
            {dna && dnaDraft ? (
              <>
                <ConfigLayer
                  title="Surface · 运行时"
                  items={dnaDraft.surface}
                  showSensitive={showSensitive}
                  editable={caps.configEdit}
                  onChange={(key, value) => updateItem(dnaDraft, setDnaDraft, 'surface', key, value)}
                  schema={dna.schema.fields}
                />
                <ConfigLayer
                  title="Middle · API & 权限"
                  items={dnaDraft.middle}
                  showSensitive={showSensitive}
                  editable={caps.configEdit}
                  onChange={(key, value) => updateItem(dnaDraft, setDnaDraft, 'middle', key, value)}
                  schema={dna.schema.fields}
                />
                <ConfigLayer
                  title="Deep · 元信息"
                  items={dnaDraft.deep}
                  showSensitive={showSensitive}
                  editable={caps.configEdit}
                  onChange={(key, value) => updateItem(dnaDraft, setDnaDraft, 'deep', key, value)}
                  schema={dna.schema.fields}
                />
                <div className="text-[10px] text-ink-faint pt-2 border-t border-white/5">
                  source: {dna.source}
                </div>
              </>
            ) : (
              <div className="text-ink-faint">加载中…</div>
            )}
          </CardContent>
        </Card>
      )}

      {showSessions && (
        <Card className="flex flex-col min-h-0">
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <ScrollText className="h-3 w-3" />
              Memory Shards · <span className="text-tribe">{sessions.length}</span>
              {!caps.sessionTail && (
                <span className="text-[10px] text-ink-faint normal-case tracking-wider ml-2">
                  (live tail M2)
                </span>
              )}
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
      )}
    </div>
  )
}

function updateItem(
  dna: ConfigDNA,
  setDna: (d: ConfigDNA | null) => void,
  layer: 'surface' | 'middle' | 'deep',
  key: string,
  value: unknown,
) {
  const list = dna[layer].map(item => (item.key === key ? {...item, value} : item))
  setDna({...dna, [layer]: list})
}

function Row({label, value}: {label: string; value: string}) {
  return (
    <div className="flex items-start gap-3">
      <span className="text-ink-faint shrink-0 w-16">{label}</span>
      <span className="text-ink break-all flex-1">{value}</span>
    </div>
  )
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
  editable,
  onChange,
  schema,
}: {
  title: string
  items: ConfigItem[]
  showSensitive: boolean
  editable: boolean
  onChange: (key: string, value: unknown) => void
  schema: {key: string; type: string; label?: string; description?: string}[]
}) {
  if (items.length === 0) return null
  const fieldMap = new Map(schema.map(f => [f.key, f]))
  return (
    <div>
      <div className="text-[10px] text-ink-faint uppercase tracking-wider mb-1.5">{title}</div>
      <div className="space-y-1.5">
        {items.map(item => {
          const field = fieldMap.get(item.key)
          const showValue = item.sensitive && !showSensitive ? '••••••••' : JSON.stringify(item.value)
          return (
            <div key={item.key} className="leading-relaxed">
              <div className="flex items-start gap-2">
                <span className="text-ink-dim shrink-0 text-[11px]">{item.key}</span>
                {field?.label && (
                  <span className="text-ink-faint text-[10px] shrink-0">· {field.label}</span>
                )}
              </div>
              {editable ? (
                <input
                  defaultValue={item.sensitive && !showSensitive ? '' : JSON.stringify(item.value)}
                  placeholder={item.sensitive && !showSensitive ? '••••••••' : ''}
                  onChange={e => onChange(item.key, tryParse(e.target.value))}
                  className="w-full mt-1 px-2 py-1 bg-land-1/50 border border-white/5 rounded text-[11px] text-ink font-mono focus:outline-none focus:border-tribe/50"
                />
              ) : (
                <span className="text-ink break-all text-[11px]">{showValue}</span>
              )}
            </div>
          )
        })}
      </div>
    </div>
  )
}

function tryParse(s: string): unknown {
  try {
    return JSON.parse(s)
  } catch {
    return s
  }
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
