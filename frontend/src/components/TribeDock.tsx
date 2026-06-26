// TribeDock 右侧面板——所有部落的快速入口。
// 核心改动：让用户一眼看到 Claude 在哪、能不能进。
// 可折叠：默认展开，看地图更沉浸时收起。

import {useState} from 'react'
import {useAland} from '../stores/alandStore'
import {Badge} from './ui'
import {ArrowRight, ChevronRight, Eye, MapPin} from 'lucide-react'
import type {Tribe} from '../api/wails'

interface TribeDockProps {
  onOpenForge?: () => void
  onOpenMatrix?: () => void
  onOpenSpotlight?: () => void
  onOpenEye?: () => void
}

export function TribeDock({onOpenForge, onOpenMatrix, onOpenSpotlight, onOpenEye}: TribeDockProps) {
  const [collapsed, setCollapsed] = useState(false)
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)
  const enterTribe = useAland(s => s.enterTribe)

  const list = Object.values(meta).map(m => ({
    meta: m,
    tribe: tribes[m.id],
  }))

  // 排序：running 优先，然后按 mtime（这里没有 mtime，按 id 稳定排序）
  list.sort((a, b) => {
    const aRun = a.tribe?.status === 'running' || a.tribe?.status === 'busy' ? 1 : 0
    const bRun = b.tribe?.status === 'running' || b.tribe?.status === 'busy' ? 1 : 0
    return bRun - aRun
  })

  const runningCount = list.filter(l => l.tribe?.status === 'running' || l.tribe?.status === 'busy').length

  return (
    <div className="absolute right-4 top-16 bottom-8 w-72 flex flex-col gap-2 z-20 pointer-events-none">
      {/* 工具栏（折叠时也保留在折叠条里） */}
      <div className="flex items-center gap-1 pointer-events-auto">
        <div className="rounded-lg border border-white/5 bg-land-2/70 backdrop-blur p-2 flex gap-1 flex-1">
          {onOpenSpotlight && (
            <button
              onClick={onOpenSpotlight}
              className="flex-1 px-2 py-1.5 rounded text-[10px] font-mono uppercase tracking-wider text-ink-dim hover:text-ink hover:bg-white/5 transition-colors"
              title="Cmd+Shift+A"
            >
              ⌘⇧A
            </button>
          )}
          {onOpenForge && (
            <button
              onClick={onOpenForge}
              className="flex-1 px-2 py-1.5 rounded text-[10px] font-mono uppercase tracking-wider text-ink-dim hover:text-ink hover:bg-white/5 transition-colors"
            >
              Forge
            </button>
          )}
          {onOpenMatrix && (
            <button
              onClick={onOpenMatrix}
              className="flex-1 px-2 py-1.5 rounded text-[10px] font-mono uppercase tracking-wider text-ink-dim hover:text-ink hover:bg-white/5 transition-colors"
            >
              Matrix
            </button>
          )}
          {onOpenEye && <EyeDockButton onClick={onOpenEye} />}
        </div>
        {/* 折叠按钮 */}
        <button
          onClick={() => setCollapsed(c => !c)}
          className="rounded-lg border border-white/5 bg-land-2/70 backdrop-blur p-2.5 text-ink-dim hover:text-ink hover:bg-white/5 transition-colors"
          title={collapsed ? 'Expand' : 'Collapse'}
        >
          <ChevronRight className={`h-3 w-3 transition-transform ${collapsed ? '' : 'rotate-180'}`} />
        </button>
      </div>

      {/* Now Active 提示 */}
      {!collapsed && runningCount > 0 && (
        <div className="rounded-lg border border-forge-green/30 bg-forge-green/5 backdrop-blur px-3 py-2 pointer-events-auto">
          <div className="flex items-center gap-2 text-[10px] font-mono uppercase tracking-wider text-forge-green">
            <span className="h-1.5 w-1.5 rounded-full bg-forge-green animate-pulse" />
            {runningCount} ACTIVE
          </div>
        </div>
      )}

      {/* 部落列表 */}
      {!collapsed && (
      <div className="flex-1 rounded-lg border border-white/5 bg-land-2/40 backdrop-blur p-2 space-y-1.5 overflow-y-auto pointer-events-auto">
        {list.length === 0 ? (
          <div className="text-ink-faint text-xs font-mono py-4 text-center">
            没有部落
          </div>
        ) : (
          list.map(({meta: m, tribe}) => (
            <TribeCard key={m.id} tribe={tribe} onEnter={() => enterTribe(m.id)} />
          ))
        )}
      </div>
      )}

      {/* 底部提示 */}
      {!collapsed && (
        <div className="text-[9px] font-mono uppercase tracking-wider text-ink-faint/50 text-center pointer-events-auto">
          <MapPin className="inline h-2.5 w-2.5 mr-1" />
          click map or card
        </div>
      )}
    </div>
  )
}

function TribeCard({tribe, onEnter}: {tribe: Tribe | undefined; onEnter: () => void}) {
  const isRunning = tribe?.status === 'running' || tribe?.status === 'busy'
  const cpu = tribe?.vital.cpu ?? 0
  const mem = tribe?.vital.memory ?? 0
  const pid = tribe?.vital.pid ?? 0

  return (
    <button
      onClick={onEnter}
      className={`interactive group w-full text-left rounded-md p-2.5 transition-all border ${
        isRunning
          ? 'border-forge-green/30 bg-forge-green/5 hover:bg-forge-green/10 hover:border-forge-green/50'
          : 'border-white/5 bg-white/[0.02] hover:bg-white/[0.05] hover:border-white/10'
      }`}
      style={
        {
          ['--tribe-theme' as string]: tribe?.meta.themeColor ?? '#666',
        } as React.CSSProperties
      }
    >
      <div className="flex items-center gap-2 mb-1.5">
        <span
          className="h-2 w-2 rounded-full shrink-0"
          style={{
            backgroundColor: tribe?.meta.themeColor ?? '#666',
            boxShadow: isRunning ? `0 0 8px ${tribe?.meta.themeColor ?? '#666'}` : 'none',
          }}
        />
        <span className="text-sm text-ink font-mono font-medium flex-1 truncate">
          {tribe?.meta.name ?? '—'}
        </span>
        {tribe && <Badge status={tribe.status} />}
        <ArrowRight className="h-3 w-3 text-ink-faint opacity-0 group-hover:opacity-100 transition-opacity" />
      </div>
      <div className="flex items-center gap-3 text-[10px] font-mono text-ink-faint">
        <span className="uppercase tracking-wider">{tribe?.meta.eco ?? '—'}</span>
        {pid > 0 && (
          <>
            <span>·</span>
            <span className="text-ink-dim">PID {pid}</span>
          </>
        )}
        {isRunning && (
          <>
            <span>·</span>
            <span className="text-forge-amber">{cpu.toFixed(0)}% CPU</span>
            <span>·</span>
            <span>{mem.toFixed(0)}MB</span>
          </>
        )}
      </div>
    </button>
  )
}

// 灵动岛快捷按钮——mode 颜色 + 通知数小点。
function EyeDockButton({onClick}: {onClick: () => void}) {
  const eye = useAland(s => s.eye)
  const flashes = eye.flashing.length
  const dot =
    eye.mode === 'storm'
      ? 'bg-forge-amber'
      : eye.mode === 'alert'
        ? 'bg-forge-red'
        : eye.mode === 'active'
          ? 'bg-forge-green'
          : 'bg-ink-faint'

  return (
    <button
      onClick={onClick}
      className="flex-1 relative px-2 py-1.5 rounded text-[10px] font-mono uppercase tracking-wider text-ink-dim hover:text-ink hover:bg-white/5 transition-colors inline-flex items-center justify-center gap-1"
      title={`灵动岛 · ${eye.mode}${flashes ? ` · ${flashes} notifications` : ''}`}
    >
      <Eye className="h-3 w-3" />
      Eye
      {flashes > 0 && (
        <span className="absolute -top-1 -right-1 min-w-[14px] h-[14px] px-1 rounded-full bg-forge-red text-[8px] font-mono text-white inline-flex items-center justify-center">
          {flashes}
        </span>
      )}
      <span className={`absolute bottom-1 right-1 h-1 w-1 rounded-full ${dot}`} />
    </button>
  )
}
