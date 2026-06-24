// 部落状态徽章：idle / running / busy / error。
// 通过 CVA 暴露变体，业务侧按 status 选。

import {cva, type VariantProps} from 'class-variance-authority'
import * as React from 'react'
import {cn} from '../../lib/utils'
import type {Status} from '../../api/wails'

const badgeVariants = cva(
  'inline-flex items-center gap-1.5 rounded-sm px-1.5 py-0.5 font-mono text-[10px] uppercase tracking-wider',
  {
    variants: {
      status: {
        idle: 'bg-white/5 text-ink-faint',
        running: 'bg-forge-green/15 text-forge-green',
        busy: 'bg-forge-amber/15 text-forge-amber',
        error: 'bg-forge-red/15 text-forge-red',
      },
    },
    defaultVariants: {
      status: 'idle',
    },
  },
)

export interface BadgeProps
  extends React.HTMLAttributes<HTMLSpanElement>,
    VariantProps<typeof badgeVariants> {}

export const Badge = React.forwardRef<HTMLSpanElement, BadgeProps>(
  ({className, status, children, ...props}, ref) => (
    <span ref={ref} className={cn(badgeVariants({status: status as Status, className}))} {...props}>
      <span
        className={cn(
          'h-1.5 w-1.5 rounded-full',
          status === 'running' && 'bg-forge-green',
          status === 'busy' && 'bg-forge-amber',
          status === 'error' && 'bg-forge-red',
          status === 'idle' && 'bg-ink-faint',
        )}
      />
      {children ?? status}
    </span>
  ),
)
Badge.displayName = 'Badge'
