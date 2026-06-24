// 能力矩阵视图——产品视角。
// 横轴：能力（8 个标准 + features 横向展开）
// 纵轴：部落
// 单元：✓ 完整 / ◐ 占位 / ✗ 不支持
//
// 这就是"可扩展架构"最直观的体现——一眼看出哪个 CLI 哪个能力还没做。

import {useAland} from '../stores/alandStore'
import {Dialog, DialogContent, DialogTitle, DialogDescription, Badge} from './ui'
import {Grid3x3, Check, CircleDashed, X} from 'lucide-react'

interface MatrixProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

const STANDARD_CAPS = [
  {key: 'process', label: 'Process'},
  {key: 'launch', label: 'Launch'},
  {key: 'config', label: 'Config'},
  {key: 'configEdit', label: 'Config Edit'},
  {key: 'sessions', label: 'Sessions'},
  {key: 'sessionTail', label: 'Session Tail'},
  {key: 'tokens', label: 'Tokens'},
  {key: 'tokensLive', label: 'Tokens Live'},
] as const

export function CapabilityMatrix({open, onOpenChange}: MatrixProps) {
  const meta = useAland(s => s.meta)
  const caps = useAland(s => s.caps)

  // 收集所有 feature ids（跨部落）
  // 全方位防御：caps 里可能 entry undefined、c.features 也可能 undefined
  const allFeatureIds = Array.from(
    new Set(
      Object.values(caps)
        .filter((c): c is NonNullable<typeof c> => !!c)
        .flatMap(c => (c.features ?? []).map(f => f.id)),
    ),
  )

  const tribes = Object.values(meta)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-4xl">
        <DialogTitle>
          <Grid3x3 className="inline h-4 w-4 mr-2" />
          Capability Matrix · 能力覆盖
        </DialogTitle>
        <DialogDescription>
          一目了然：哪个部落哪个能力做到了什么程度
        </DialogDescription>

        <div className="mt-4 overflow-x-auto">
          <table className="w-full text-xs font-mono">
            <thead>
              <tr className="border-b border-white/10">
                <th className="text-left py-2 px-3 text-ink-faint font-normal sticky left-0 bg-land-2/95">
                  Tribe
                </th>
                {STANDARD_CAPS.map(c => (
                  <th
                    key={c.key}
                    className="text-center py-2 px-2 text-ink-faint font-normal whitespace-nowrap"
                    title={c.label}
                  >
                    {c.label}
                  </th>
                ))}
                {allFeatureIds.map(id => (
                  <th
                    key={id}
                    className="text-center py-2 px-2 text-ink-faint font-normal whitespace-nowrap"
                    title={id}
                  >
                    {id}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {tribes.length === 0 ? (
                <tr>
                  <td colSpan={STANDARD_CAPS.length + 1} className="py-8 text-center text-ink-faint">
                    没有部落
                  </td>
                </tr>
              ) : (
                tribes.map(m => {
                  const c = caps[m.id]
                  return (
                    <tr
                      key={m.id}
                      className="border-b border-white/5 hover:bg-white/[0.02]"
                      style={{['--tribe-theme' as string]: m.themeColor} as React.CSSProperties}
                    >
                      <td className="py-2 px-3 sticky left-0 bg-land-2/95">
                        <div className="flex items-center gap-2">
                          <span
                            className="h-2 w-2 rounded-full"
                            style={{backgroundColor: m.themeColor, boxShadow: `0 0 6px ${m.themeColor}`}}
                          />
                          <span className="text-ink">{m.name}</span>
                          <span className="text-ink-faint text-[10px] uppercase">{m.eco}</span>
                        </div>
                      </td>
                      {STANDARD_CAPS.map(sc => {
                        const v = c?.[sc.key as keyof typeof c]
                        return (
                          <td key={sc.key} className="py-2 px-2 text-center">
                            {typeof v === 'boolean' ? (
                              v ? (
                                <Check className="inline h-3.5 w-3.5 text-forge-green" />
                              ) : (
                                <X className="inline h-3.5 w-3.5 text-ink-faint/30" />
                              )
                            ) : (
                              <CircleDashed className="inline h-3.5 w-3.5 text-ink-faint/30" />
                            )}
                          </td>
                        )
                      })}
                      {allFeatureIds.map(fid => {
                        const feat = c?.features?.find(f => f.id === fid)
                        if (!feat) {
                          return (
                            <td key={fid} className="py-2 px-2 text-center">
                              <X className="inline h-3.5 w-3.5 text-ink-faint/30" />
                            </td>
                          )
                        }
                        return (
                          <td key={fid} className="py-2 px-2 text-center" title={feat.description}>
                            {feat.hasData ? (
                              <Check className="inline h-3.5 w-3.5 text-forge-green" />
                            ) : (
                              <CircleDashed className="inline h-3.5 w-3.5 text-forge-amber" />
                            )}
                          </td>
                        )
                      })}
                    </tr>
                  )
                })
              )}
            </tbody>
          </table>
        </div>

        {/* Legend */}
        <div className="mt-4 pt-3 border-t border-white/5 flex items-center gap-4 text-[10px] font-mono text-ink-faint">
          <span className="flex items-center gap-1.5">
            <Check className="h-3 w-3 text-forge-green" />
            Implemented
          </span>
          <span className="flex items-center gap-1.5">
            <CircleDashed className="h-3 w-3 text-forge-amber" />
            Stub (placeholder)
          </span>
          <span className="flex items-center gap-1.5">
            <X className="h-3 w-3 text-ink-faint/30" />
            Not supported
          </span>
        </div>
      </DialogContent>
    </Dialog>
  )
}
