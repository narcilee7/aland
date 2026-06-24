// Token 熔炉面板——M1 版本。
// 展示今日总消耗 + 各部落贡献 + 模型分布 + 颜色编码（绿/琥珀/红）。
// 通过 Dialog 形式打开（z 比 Overlook 高）。

import {useEffect, useState} from 'react'
import {getForge, type Forge as ForgeData} from '../api/wails'
import {onTribeVital} from '../api/events'
import {Dialog, DialogContent, DialogTitle, DialogDescription, Badge} from './ui'
import {Flame, TrendingUp} from 'lucide-react'
import {logger} from '../lib/logger'

interface ForgeProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function Forge({open, onOpenChange}: ForgeProps) {
  const [data, setData] = useState<ForgeData>({
    dailyBudget: 0,
    todaySpent: 0,
    byTribe: {},
    byModel: {},
  })

  useEffect(() => {
    if (!open) return
    getForge().then(setData).catch(e => logger.error('forge fetch failed', e))
  }, [open])

  // 实时：每次 tribe:vital 来时刷新
  useEffect(() => {
    return onTribeVital(() => {
      if (open) getForge().then(setData).catch(() => {})
    })
  }, [open])

  const total = data.todaySpent
  const budget = data.dailyBudget
  const ratio = budget > 0 ? total / budget : 0
  const color = ratio > 1 ? 'red' : ratio > 0.7 ? 'amber' : 'green'
  const colorClass =
    color === 'red' ? 'text-forge-red' : color === 'amber' ? 'text-forge-amber' : 'text-forge-green'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogTitle>
          <Flame className="inline h-4 w-4 mr-2" />
          Forge · 今日 Token
        </DialogTitle>
        <DialogDescription>Token 熔炉 · 各部落贡献流</DialogDescription>

        {/* 总数大数字 */}
        <div className="my-6 text-center">
          <div className={`text-4xl font-mono font-medium ${colorClass}`}>
            {formatNumber(total)}
          </div>
          <div className="mt-1 text-xs text-ink-faint font-mono uppercase tracking-wider">
            tokens spent
          </div>
          {budget > 0 && (
            <div className="mt-3 text-xs text-ink-dim font-mono">
              / {formatNumber(budget)} budget · {Math.round(ratio * 100)}%
            </div>
          )}
        </div>

        {/* 进度条 */}
        {budget > 0 && (
          <div className="h-1 w-full bg-white/5 rounded overflow-hidden mb-6">
            <div
              className={`h-full transition-all duration-500 ${
                color === 'red' ? 'bg-forge-red' : color === 'amber' ? 'bg-forge-amber' : 'bg-forge-green'
              }`}
              style={{width: `${Math.min(ratio, 1) * 100}%`}}
            />
          </div>
        )}

        {/* 按部落分布 */}
        <div className="mb-4">
          <div className="text-[10px] text-ink-faint font-mono uppercase tracking-wider mb-2 flex items-center gap-1">
            <TrendingUp className="h-3 w-3" />
            By Tribe
          </div>
          <div className="space-y-1.5">
            {Object.keys(data.byTribe).length === 0 ? (
              <div className="text-xs text-ink-faint font-mono py-3 text-center">
                暂无数据 · 跑一个 CLI 就会出数字
              </div>
            ) : (
              Object.entries(data.byTribe)
                .sort((a, b) => b[1] - a[1])
                .map(([id, tokens]) => {
                  const pct = total > 0 ? (tokens / total) * 100 : 0
                  return (
                    <div key={id} className="flex items-center gap-2 text-xs font-mono">
                      <span className="w-16 text-ink-dim">{id}</span>
                      <div className="flex-1 h-1.5 bg-white/5 rounded overflow-hidden">
                        <div
                          className="h-full bg-tribe/60"
                          style={{width: `${pct}%`}}
                        />
                      </div>
                      <span className="w-20 text-right text-ink">{formatNumber(tokens)}</span>
                    </div>
                  )
                })
            )}
          </div>
        </div>

        {/* 按模型分布 */}
        {Object.keys(data.byModel).length > 0 && (
          <div>
            <div className="text-[10px] text-ink-faint font-mono uppercase tracking-wider mb-2">
              By Model
            </div>
            <div className="space-y-1.5">
              {Object.entries(data.byModel)
                .sort((a, b) => b[1] - a[1])
                .map(([model, tokens]) => (
                  <div key={model} className="flex items-center gap-2 text-xs font-mono">
                    <span className="flex-1 text-ink-dim truncate">{model}</span>
                    <span className="text-ink">{formatNumber(tokens)}</span>
                  </div>
                ))}
            </div>
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}

function formatNumber(n: number): string {
  if (n < 1000) return n.toString()
  if (n < 1_000_000) return `${(n / 1000).toFixed(1)}k`
  return `${(n / 1_000_000).toFixed(2)}M`
}
