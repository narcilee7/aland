// 与 Wails 后端的类型化 API 包装。
// 后端方法在 backend/app.go 中以大写开头，前端通过 window.go.main.App 调用。
// 这里手写类型是因为 Wails 自动生成的 .d.ts 在 dev 跑之前不存在。

import type {EventsOn} from '../../wailsjs/runtime/runtime'

// —— 与后端 tribes.Tribe / tribes.Meta 对齐 ——

export type Status = 'idle' | 'running' | 'busy' | 'error'

export interface VitalSign {
  pid: number
  cpu: number
  memory: number
  cwd: string
  uptime: number
  model: string
  updatedAt: number
}

export interface TribeMeta {
  id: string
  name: string
  eco: string
  themeColor: string
  accentColor: string
}

export interface Tribe {
  meta: TribeMeta
  status: Status
  vital: VitalSign
}

export interface Forge {
  dailyBudget: number
  todaySpent: number
  byTribe: Record<string, number>
  byModel: Record<string, number>
}

// —— 类型安全的 Wails 桥 ——

declare global {
  interface Window {
    go: {
      main: {
        App: {
          GetLand(): Promise<Record<string, Tribe>>
          GetTribe(id: string): Promise<Tribe>
          LaunchTribe(id: string, cwd: string, args: string[]): Promise<void>
          KillTribe(id: string): Promise<void>
          ReadTribeConfig(id: string): Promise<Record<string, unknown>>
          GetForge(): Promise<Forge>
          GetTribeMeta(id: string): Promise<TribeMeta>
        }
      }
    }
    runtime: {
      EventsOn: typeof EventsOn
      EventsEmit(eventName: string, ...data: unknown[]): void
    }
  }
}

function wailsAvailable(): boolean {
  return typeof window !== 'undefined' && !!window.go?.main?.App
}

export async function getLand(): Promise<Record<string, Tribe>> {
  if (!wailsAvailable()) return {}
  return window.go.main.App.GetLand()
}

export async function getTribe(id: string): Promise<Tribe | null> {
  if (!wailsAvailable()) return null
  try {
    return await window.go.main.App.GetTribe(id)
  } catch {
    return null
  }
}

export async function getTribeMeta(id: string): Promise<TribeMeta | null> {
  if (!wailsAvailable()) return null
  try {
    return await window.go.main.App.GetTribeMeta(id)
  } catch {
    return null
  }
}

export function onTribeVital(cb: (snapshot: Record<string, Tribe>) => void) {
  if (!wailsAvailable()) return () => {}
  return window.runtime.EventsOn('tribe:vital', cb)
}

export function onTribeBorn(cb: (e: {id: string; pid: number; name: string}) => void) {
  if (!wailsAvailable()) return () => {}
  return window.runtime.EventsOn('tribe:born', cb)
}

export function onTribeDeath(cb: (e: {id: string; pid: number; name: string}) => void) {
  if (!wailsAvailable()) return () => {}
  return window.runtime.EventsOn('tribe:death', cb)
}
