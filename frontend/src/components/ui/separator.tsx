// 分隔线。垂直/水平可选。
import * as React from 'react'
import {cn} from '../../lib/utils'

interface SeparatorProps extends React.HTMLAttributes<HTMLDivElement> {
  orientation?: 'horizontal' | 'vertical'
}

export const Separator = React.forwardRef<HTMLDivElement, SeparatorProps>(
  ({className, orientation = 'horizontal', ...props}, ref) => (
    <div
      ref={ref}
      role="separator"
      className={cn(
        'bg-white/5',
        orientation === 'horizontal' ? 'h-px w-full' : 'w-px h-full',
        className,
      )}
      {...props}
    />
  ),
)
Separator.displayName = 'Separator'
