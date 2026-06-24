// 基础按钮——shadcn/ui 风格的最小实现。
// 用 class-variance-authority 管理变体，asChild 用 @radix-ui/react-slot。
//
// 用法：
//   <Button>Click</Button>
//   <Button variant="ghost" size="sm">Cancel</Button>
//   <Button asChild><Link to="/x">Go</Link></Button>

import {Slot} from '@radix-ui/react-slot'
import {cva, type VariantProps} from 'class-variance-authority'
import * as React from 'react'
import {cn} from '../../lib/utils'

const buttonVariants = cva(
  // 基础
  'inline-flex items-center justify-center gap-2 whitespace-nowrap font-mono text-xs uppercase tracking-wider transition-colors focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-tribe disabled:pointer-events-none disabled:opacity-50',
  {
    variants: {
      variant: {
        default:
          'bg-tribe/20 text-tribe border border-tribe/40 hover:bg-tribe/30 hover:border-tribe/60',
        ghost: 'text-ink-dim hover:text-ink hover:bg-white/5',
        outline:
          'border border-ink-faint/40 text-ink-dim hover:border-tribe/60 hover:text-tribe',
        solid: 'bg-tribe text-land-1 hover:bg-tribe/90',
      },
      size: {
        sm: 'h-7 px-2.5',
        md: 'h-8 px-3.5',
        lg: 'h-10 px-5 text-sm',
        icon: 'h-8 w-8 p-0',
      },
    },
    defaultVariants: {
      variant: 'default',
      size: 'md',
    },
  },
)

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement>,
    VariantProps<typeof buttonVariants> {
  asChild?: boolean
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  ({className, variant, size, asChild = false, ...props}, ref) => {
    const Comp = asChild ? Slot : 'button'
    return (
      <Comp
        className={cn(buttonVariants({variant, size, className}))}
        ref={ref}
        {...props}
      />
    )
  },
)
Button.displayName = 'Button'

export {buttonVariants}
