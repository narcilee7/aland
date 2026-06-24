// 部落内部视图（M0 占位）。
// 推镜转场后展示单个部落的生命体征 + 配置入口。
// 完整版（M2+）会有配置双螺旋、记忆墙、风铃等。

import {useEffect, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {getTribe, getTribeMeta, type Tribe, type TribeMeta} from '../api/wails'
import {ArrowLeft} from 'lucide-react'
import {Badge, Button, Card, CardContent, CardHeader, CardTitle, Separator} from './ui'

export function TribeView() {
  const activeTribe = useAland(s => s.activeTribe)
  const returnOverlook = useAland(s => s.returnOverlook)
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)

  const [detail, setDetail] = useState<Tribe | null>(null)
  const [tribeMeta, setTribeMeta] = useState<TribeMeta | null>(null)
  const [config, setConfig] = useState<Record<string, unknown> | null>(null)

  useEffect(() => {
    if (!activeTribe) return
    Promise.all([getTribe(activeTribe), getTribeMeta(activeTribe)]).then(([t, m]) => {
      setDetail(t)
      setTribeMeta(m)
    })
    if (window.go?.main?.App) {
      window.go.main.App.ReadTribeConfig(activeTribe).then(setConfig).catch(() => setConfig(null))
    }
  }, [activeTribe])

  if (!activeTribe) return null
  const liveTribe = tribes[activeTribe] || detail
  const liveMeta = meta[activeTribe] || tribeMeta
  if (!liveMeta) return null

  return (
    <div
      className="absolute inset-0 p-12 flex flex-col gap-6 backdrop-blur-md"
      style={
        {
          background: 'radial-gradient(ellipse at center, rgba(13,31,21,0.6) 0%, rgba(10,14,26,0.95) 70%)',
          ['--tribe-theme' as string]: liveMeta.themeColor,
          ['--tribe-accent' as string]: liveMeta.accentColor,
        } as React.CSSProperties
      }
    >
      {/* 顶部：返回 + 标题 */}
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

      {/* 生命体征卡片 */}
      <div className="grid grid-cols-4 gap-4 font-mono">
        <VitalCard label="Status" value={liveTribe?.status ?? '—'} />
        <VitalCard label="PID" value={String(liveTribe?.vital.pid ?? '—')} />
        <VitalCard label="CPU" value={`${(liveTribe?.vital.cpu ?? 0).toFixed(1)}%`} />
        <VitalCard label="Memory" value={`${(liveTribe?.vital.memory ?? 0).toFixed(0)}MB`} />
      </div>

      {/* 配置预览（M0 只读，M2+ 才是 Helix 双螺旋） */}
      <Card className="flex-1 overflow-hidden">
        <CardHeader>
          <CardTitle>Config · read-only</CardTitle>
          <span className="text-[10px] text-ink-faint font-mono">M2+ will replace with Helix</span>
        </CardHeader>
        <CardContent className="overflow-auto">
          <pre className="m-0 font-mono text-xs text-ink-dim whitespace-pre-wrap break-all">
            {config ? JSON.stringify(config, null, 2) : '(no config found)'}
          </pre>
        </CardContent>
      </Card>
    </div>
  )
}

function VitalCard({label, value}: {label: string; value: string}) {
  return (
    <Card>
      <CardContent className="py-4">
        <div className="text-[10px] uppercase tracking-wider text-ink-faint mb-2 font-mono">
          {label}
        </div>
        <div className="text-lg text-tribe font-medium font-mono">{value}</div>
      </CardContent>
    </Card>
  )
}
