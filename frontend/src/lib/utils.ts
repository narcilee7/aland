// shadcn/ui 风格的 cn() 工具：合并 className，去重 Tailwind 冲突。
// 任何业务组件组合样式都走这里，避免 clsx + twMerge 散落各处。

import {clsx, type ClassValue} from 'clsx'
import {twMerge} from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}
