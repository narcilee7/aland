// 灵动岛 (Eye) 弹窗——展示 mode、活跃 tribe、通知队列。
//
// 设计：
// - 与菜单栏图标语义对齐：mode 颜色 + 名称
// - 通知列表按时间倒序，每条可单独 consume（标记已读）
// - "全部已读" 一键清空
// - 空状态：dormant + 无通知 → 温馨文案

import {useEffect, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {Dialog, DialogContent, DialogTitle, DialogDescription} from './ui'
import {Eye, Sparkles, Trash2, X, Activity, AlertTriangle, Zap, Moon} from 'lucide-react'
import type {FlashType, Flash, EyeMode} from '../api/wails'
import {wailsAvailable} from '../api/wails'

interface EyeDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const MODE_META: Record<EyeMode, {label: string; color: string; bg: string; Icon: typeof Moon}> = {
  dormant: {label: '沉睡守望', color: 'text-ink-dim', bg: 'bg-white/5', Icon: Moon},
  active: {label: '部落活跃', color: 'text-forge-green', bg: 'bg-forge-green/10', Icon: Activity},
  storm: {label: '风暴 · 高负载', color: 'text-forge-amber', bg: 'bg-forge-amber/10', Icon: Zap},
  alert: {label: '告警', color: 'text-forge-red', bg: 'bg-forge-red/10', Icon: AlertTriangle},
}

const FLASH_META: Record<FlashType, {label: string; color: string}> = {
  complete: {label: 'complete', color: 'text-forge-green'},
  cost_alert: {label: 'cost', color: 'text-forge-amber'},
  error: {label: 'error', color: 'text-forge-red'},
  conflict: {label: 'conflict', color: 'text-forge-red'},
  born: {label: 'born', color: 'text-forge-green'},
  death: {label: 'death', color: 'text-ink-dim'},
}

export function EyeDialog({open, onOpenChange}: EyeDialogProps) {
  const eye = useAland(s => s.eye)
  const consumeFlash = useAland(s => s.consumeFlash)
  const clearFlashes = useAland(s => s.clearFlashes)
  const meta = useAland(s => s.meta)

  // 通知按 createdAt 倒序
  const flashes = [...eye.flashing].sort((a, b) => b.createdAt - a.createdAt)
  const modeMeta = MODE_META[eye.mode] ?? MODE_META.dormant
  const Icon = modeMeta.Icon

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogTitle>
          <Eye className="inline h-4 w-4 mr-2" />
          Eye · 灵动岛
        </DialogTitle>
        <DialogDescription>
          守望者模式 · 当前陆上动态
        </DialogDescription>

        {/* Mode 徽章 */}
        <div className={`mt-4 rounded-md border border-white/5 ${modeMeta.bg} p-3 flex items-center gap-3`}>
          <Icon className={`h-5 w-5 ${modeMeta.color}`} />
          <div className="flex-1">
            <div className={`font-mono text-sm uppercase tracking-wider ${modeMeta.color}`}>
              {eye.mode}
            </div>
            <div className="text-[11px] text-ink-faint font-mono">
              {modeMeta.label}
            </div>
          </div>
        </div>

        {/* Running 列表 */}
        <div className="mt-4">
          <div className="text-[10px] text-ink-faint font-mono uppercase tracking-wider mb-2">
            Running Tribes ({eye.running.length})
          </div>
          {eye.running.length === 0 ? (
            <div className="text-xs text-ink-faint font-mono py-3 px-2 rounded bg-white/[0.02] text-center">
              当前无部落活跃
            </div>
          ) : (
            <div className="space-y-1">
              {eye.running.map(id => {
                const m = meta[id]
                return (
                  <div
                    key={id}
                    className="flex items-center gap-2 px-2 py-1.5 rounded bg-white/[0.02] border border-white/5"
                  >
                    <span
                      className="h-2 w-2 rounded-full shrink-0"
                      style={{background: m?.themeColor ?? '#fff'}}
                    />
                    <span className="text-xs font-mono text-ink">{m?.name ?? id}</span>
                    <span className="ml-auto text-[10px] text-ink-faint font-mono">{id}</span>
                  </div>
                )
              })}
            </div>
          )}
        </div>

        {/* Flash 队列 */}
        <div className="mt-4">
          <div className="flex items-center justify-between mb-2">
            <div className="text-[10px] text-ink-faint font-mono uppercase tracking-wider">
              Notifications ({flashes.length})
            </div>
            {flashes.length > 0 && (
              <button
                onClick={() => clearFlashes()}
                className="text-[10px] font-mono text-ink-faint hover:text-ink-dim inline-flex items-center gap-1"
                title="Clear all"
              >
                <Trash2 className="h-3 w-3" />
                clear
              </button>
            )}
          </div>
          {flashes.length === 0 ? (
            <div className="text-xs text-ink-faint font-mono py-6 px-2 rounded bg-white/[0.02] text-center">
              <Sparkles className="inline h-3 w-3 mr-1" />
              暂无通知
            </div>
          ) : (
            <div className="space-y-1 max-h-60 overflow-y-auto">
              {flashes.map(f => (
                <FlashRow key={f.id} flash={f} onConsume={() => consumeFlash(f.id)} />
              ))}
            </div>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}

function FlashRow({flash, onConsume}: {flash: Flash; onConsume: () => void}) {
  const meta = FLASH_META[flash.type] ?? FLASH_META.complete
  return (
    <div className="flex items-start gap-2 px-2 py-1.5 rounded bg-white/[0.02] border border-white/5 group">
      <span className={`text-[9px] font-mono uppercase tracking-wider mt-0.5 shrink-0 ${meta.color}`}>
        [{meta.label}]
      </span>
      <div className="flex-1 min-w-0">
        <div className="text-xs text-ink-dim break-all">{flash.content}</div>
        <div className="text-[9px] text-ink-faint font-mono mt-0.5">
          {new Date(flash.createdAt).toLocaleTimeString()}
          {flash.tribe && ` · ${flash.tribe}`}
        </div>
      </div>
      <button
        onClick={onConsume}
        className="opacity-0 group-hover:opacity-100 text-ink-faint hover:text-ink-dim transition-opacity shrink-0"
        title="Mark as read"
      >
        <X className="h-3 w-3" />
      </button>
    </div>
  )
}

// 微小 helper：检查 Eye 是否可用（用于条件渲染）
// 注意：wails.ts 已静态导入，避免重复 import。
export function useEyeAvailable(): boolean {
  const [avail, setAvail] = useState(false)
  useEffect(() => {
    setAvail(wailsAvailable())
  }, [])
  return avail
}