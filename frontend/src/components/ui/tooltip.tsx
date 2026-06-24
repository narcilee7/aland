// 工具提示。Radix 提供 a11y 与键盘支持；外观用 Tailwind。
//
// 用法：
//   <TooltipProvider>
//     <Tooltip>
//       <TooltipTrigger asChild><Button>Hover me</Button></TooltipTrigger>
//       <TooltipContent side="top">解释</TooltipContent>
//     </Tooltip>
//   </TooltipProvider>

import * as TooltipPrimitive from '@radix-ui/react-tooltip'
import * as React from 'react'
import {cn} from '../../lib/utils'

export const TooltipProvider = TooltipPrimitive.Provider
export const Tooltip = TooltipPrimitive.Root
export const TooltipTrigger = TooltipPrimitive.Trigger

export const TooltipContent = React.forwardRef<
  React.ElementRef<typeof TooltipPrimitive.Content>,
  React.ComponentPropsWithoutRef<typeof TooltipPrimitive.Content>
>(({className, sideOffset = 4, ...props}, ref) => (
  <TooltipPrimitive.Portal>
    <TooltipPrimitive.Content
      ref={ref}
      sideOffset={sideOffset}
      className={cn(
        'z-50 overflow-hidden rounded border border-white/10 bg-land-1/95 px-2.5 py-1.5',
        'text-xs text-ink font-mono shadow-md backdrop-blur',
        'data-[state=delayed-open]:animate-in data-[state=closed]:animate-out',
        'data-[state=closed]:fade-out-0 data-[state=delayed-open]:fade-in-0',
        'data-[side=bottom]:slide-in-from-top-1 data-[side=top]:slide-in-from-bottom-1',
        className,
      )}
      {...props}
    />
  </TooltipPrimitive.Portal>
))
TooltipContent.displayName = TooltipPrimitive.Content.displayName
