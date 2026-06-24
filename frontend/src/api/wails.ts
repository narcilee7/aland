// 与 Wails 后端的类型化 API 包装。
// 后端方法在 backend/app.go 中以大写开头，前端通过 window.go.main.App 调用。
// 这里手写类型是因为 Wails 自动生成的 .d.ts 在 dev 跑之前不存在。
// 事件订阅统一搬到 ./events.ts。

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

// —— Config DNA 三层 ——

export interface ConfigItem {
  key: string
  value: unknown
  type: string
  layer: 'surface' | 'middle' | 'deep'
  sensitive?: boolean
}

export interface ConfigField {
  key: string
  label?: string
  description?: string
  type: 'string' | 'number' | 'boolean' | 'enum' | 'secret' | 'json'
  enumValues?: string[]
  sensitive: boolean
  editable: boolean
  default?: unknown
}

export interface ConfigSchema {
  fields: ConfigField[]
}

export interface ConfigDNA {
  source: string
  schema: ConfigSchema
  surface: ConfigItem[]
  middle: ConfigItem[]
  deep: ConfigItem[]
}

// —— 能力清单（核心契约）——

export interface Feature {
  id: string
  label: string
  description: string
  hasData: boolean
}

export interface Capabilities {
  process: boolean
  launch: boolean
  config: boolean
  configEdit: boolean
  sessions: boolean
  sessionTail: boolean
  tokens: boolean
  tokensLive: boolean
  features: Feature[]
}

// —— 记忆碎片 ——

export interface SessionShard {
  id: string
  tribe: string
  timestamp: number
  messageCount: number
  tokenCount: number
  model: string
  cwd: string
  project: string
  filePath: string
  sizeBytes: number
  summary: string
}

// —— Token 熔炉 ——

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
          ReadTribeConfigDNA(id: string): Promise<ConfigDNA>
          WriteTribeConfig(id: string, dna: ConfigDNA): Promise<void>
          GetForge(): Promise<Forge>
          GetTribeMeta(id: string): Promise<TribeMeta>
          GetTribeCapabilities(id: string): Promise<Capabilities>
          GetAllCapabilities(): Promise<Record<string, Capabilities>>
          ListSessions(id: string): Promise<SessionShard[]>
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

export async function getForge(): Promise<Forge> {
  if (!wailsAvailable()) return {dailyBudget: 0, todaySpent: 0, byTribe: {}, byModel: {}}
  return window.go.main.App.GetForge()
}

export async function readTribeConfigDNA(id: string): Promise<ConfigDNA | null> {
  if (!wailsAvailable()) return null
  try {
    return await window.go.main.App.ReadTribeConfigDNA(id)
  } catch {
    return null
  }
}

export async function listSessions(id: string): Promise<SessionShard[]> {
  if (!wailsAvailable()) return []
  try {
    return await window.go.main.App.ListSessions(id)
  } catch {
    return []
  }
}

export async function getTribeCapabilities(id: string): Promise<Capabilities | null> {
  if (!wailsAvailable()) return null
  try {
    return await window.go.main.App.GetTribeCapabilities(id)
  } catch {
    return null
  }
}

export async function getAllCapabilities(): Promise<Record<string, Capabilities>> {
  if (!wailsAvailable()) return {}
  try {
    return await window.go.main.App.GetAllCapabilities()
  } catch {
    return {}
  }
}

export async function writeTribeConfig(id: string, dna: ConfigDNA): Promise<boolean> {
  if (!wailsAvailable()) return false
  try {
    await window.go.main.App.WriteTribeConfig(id, dna)
    return true
  } catch {
    return false
  }
}
