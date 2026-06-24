// Spotlight —— Cmd+Shift+A 唤起的悬浮地图。
// 设计上像 macOS Spotlight：你从云端俯瞰大陆，键入可搜索，方向键选部落，回车进入。
// M0+1 版本：4 部落 chip 网格 + 点击进入。

import {useEffect} from 'react'
import {useAland} from '../stores/alandStore'
import {onSpotlightToggle} from '../api/events'
import {Dialog, DialogContent, DialogTitle, DialogDescription, Badge} from './ui'
import {Search, MapPin} from 'lucide-react'

export function Spotlight() {
  const open = useAland(s => s.spotlight)
  const setOpen = useAland(s => s.setSpotlight)
  const enterTribe = useAland(s => s.enterTribe)
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)

  // 订阅后端快捷键事件
  useEffect(() => {
    return onSpotlightToggle(e => {
      if (e.action === 'toggle') setOpen(!useAland.getState().spotlight)
      else if (e.action === 'open') setOpen(true)
      else if (e.action === 'close') setOpen(false)
    })
  }, [setOpen])

  const tribeList = Object.values(meta).map(m => ({
    meta: m,
    status: tribes[m.id]?.status ?? 'idle',
  }))

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent
        className="max-w-2xl gap-3 p-0 overflow-hidden"
        onOpenAutoFocus={e => e.preventDefault()}
      >
        {/* 顶部搜索条 */}
        <div className="flex items-center gap-3 px-5 py-4 border-b border-white/5">
          <Search className="h-4 w-4 text-ink-faint shrink-0" />
          <input
            autoFocus
            placeholder="Search tribes…"
            className="flex-1 bg-transparent border-none outline-none text-sm text-ink placeholder:text-ink-faint font-mono"
          />
        </div>

        <div className="px-5 pb-2">
          <DialogTitle className="text-[10px] tracking-widest text-ink-faint">
            From Above
          </DialogTitle>
          <DialogDescription className="text-xs text-ink-faint">
            你的 Agent 数字大陆 · 当前 4 部落
          </DialogDescription>
        </div>

        {/* 部落网格 */}
        <div className="grid grid-cols-2 gap-2 p-3 pt-0">
          {tribeList.length === 0 ? (
            <div className="col-span-2 py-12 text-center text-ink-faint text-xs font-mono">
              <MapPin className="mx-auto mb-2 h-5 w-5 opacity-30" />
              还没有部落登记到大陆
            </div>
          ) : (
            tribeList.map(({meta: m, status}) => (
              <button
                key={m.id}
                onClick={() => {
                  setOpen(false)
                  enterTribe(m.id)
                }}
                className="interactive group flex items-center gap-3 rounded border border-white/5 bg-white/[0.02] p-3 text-left transition-all hover:bg-white/[0.06]"
                style={
                  {
                    ['--tribe-theme' as string]: m.themeColor,
                  } as React.CSSProperties
                }
              >
                <div
                  className="h-3 w-3 rounded-full"
                  style={{backgroundColor: m.themeColor, boxShadow: `0 0 8px ${m.themeColor}`}}
                />
                <div className="flex-1 min-w-0">
                  <div className="text-sm text-ink font-mono truncate">{m.name}</div>
                  <div className="text-[10px] text-ink-faint font-mono uppercase tracking-wider">
                    {m.eco}
                  </div>
                </div>
                <Badge status={status as any} />
              </button>
            ))
          )}
        </div>

        {/* 底部提示 */}
        <div className="flex items-center justify-between border-t border-white/5 bg-land-1/50 px-5 py-2 text-[10px] text-ink-faint font-mono">
          <span>↑↓ navigate</span>
          <span>↵ enter tribe</span>
        </div>
      </DialogContent>
    </Dialog>
  )
}
