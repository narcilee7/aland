// 集中导出 UI 底座，业务组件从这里 import。
export {Button, buttonVariants, type ButtonProps} from './button'
export {
  Card,
  CardHeader,
  CardTitle,
  CardDescription,
  CardContent,
  CardFooter,
} from './card'
export {Badge, type BadgeProps} from './badge'
export {Separator} from './separator'
export {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
  TooltipProvider,
} from './tooltip'
export {
  Dialog,
  DialogTrigger,
  DialogPortal,
  DialogClose,
  DialogOverlay,
  DialogContent,
  DialogHeader,
  DialogFooter,
  DialogTitle,
  DialogDescription,
} from './dialog'
export {cn} from '../../lib/utils'
