// 部落内部视图（M0 占位）。
// 推镜转场后展示单个部落的生命体征 + 配置入口。
// 完整版（M2+）会有配置双螺旋、记忆墙、风铃等。

import {useEffect, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {getTribe, getTribeMeta, type Tribe, type TribeMeta} from '../api/wails'

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
    // 读取配置（M2+ 才会真正利用）
    if (window.go?.main?.App) {
      window.go.main.App.ReadTribeConfig(activeTribe).then(setConfig).catch(() => setConfig(null))
    }
  }, [activeTribe])

  if (!activeTribe) return null
  const liveTribe = tribes[activeTribe] || detail
  const liveMeta = meta[activeTribe] || tribeMeta
  if (!liveMeta) return null

  const accent = liveMeta.accentColor
  const theme = liveMeta.themeColor

  return (
    <div
      style={{
        position: 'absolute',
        inset: 0,
        background: 'radial-gradient(ellipse at center, rgba(13,31,21,0.6) 0%, rgba(10,14,26,0.95) 70%)',
        backdropFilter: 'blur(12px)',
        padding: '48px 32px 32px',
        display: 'flex',
        flexDirection: 'column',
        gap: 24,
      }}
    >
      {/* 顶部：返回 + 标题 */}
      <div
        className="draggable"
        style={{display: 'flex', alignItems: 'center', justifyContent: 'space-between'}}
      >
        <button
          className="interactive"
          onClick={returnOverlook}
          style={{
            background: 'transparent',
            border: `1px solid ${theme}55`,
            color: theme,
            padding: '6px 12px',
            fontFamily: 'var(--font-mono)',
            fontSize: 11,
            borderRadius: 3,
            cursor: 'pointer',
          }}
        >
          ← BACK TO OVERLOOK
        </button>
        <h1
          style={{
            margin: 0,
            fontFamily: 'var(--font-mono)',
            fontSize: 18,
            fontWeight: 500,
            color: accent,
            letterSpacing: 2,
          }}
        >
          {liveMeta.name.toUpperCase()}
        </h1>
        <span style={{width: 120}} />
      </div>

      {/* 生命体征卡片 */}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(4, 1fr)',
          gap: 16,
          fontFamily: 'var(--font-mono)',
        }}
      >
        <VitalCard label="STATUS" value={liveTribe?.status ?? '—'} color={theme} />
        <VitalCard label="PID" value={String(liveTribe?.vital.pid ?? '—')} color={accent} />
        <VitalCard label="CPU" value={`${(liveTribe?.vital.cpu ?? 0).toFixed(1)}%`} color={accent} />
        <VitalCard label="MEMORY" value={`${(liveTribe?.vital.memory ?? 0).toFixed(0)}MB`} color={accent} />
      </div>

      {/* 配置预览（M0 只读，M2+ 才是 Helix 双螺旋） */}
      <div
        style={{
          flex: 1,
          background: 'rgba(0,0,0,0.3)',
          border: `1px solid ${theme}22`,
          borderRadius: 4,
          padding: 16,
          overflow: 'auto',
          fontFamily: 'var(--font-mono)',
          fontSize: 12,
          color: 'var(--aland-text-dim)',
        }}
      >
        <div style={{color: theme, marginBottom: 8, letterSpacing: 1}}>CONFIG (read-only · M2+ will add Helix)</div>
        <pre style={{margin: 0}}>{config ? JSON.stringify(config, null, 2) : '(no config found)'}</pre>
      </div>
    </div>
  )
}

function VitalCard({label, value, color}: {label: string; value: string; color: string}) {
  return (
    <div
      style={{
        background: 'rgba(0,0,0,0.25)',
        border: `1px solid ${color}22`,
        borderRadius: 4,
        padding: 16,
      }}
    >
      <div style={{fontSize: 10, color: 'var(--aland-text-faint)', letterSpacing: 1, marginBottom: 6}}>
        {label}
      </div>
      <div style={{fontSize: 18, color, fontWeight: 500}}>{value}</div>
    </div>
  )
}
