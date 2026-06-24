// 与 Wails 后端的类型化 API 包装。
// 后端方法在 backend/app.go 中以大写开头，前端通过 window.go.backend.App 调用。
// 这里手写类型是因为 Wails 自动生成的 .d.ts 在 dev 跑之前不存在。
// 事件订阅统一搬到 ./events.ts。

import type { EventsOn } from '../../wailsjs/runtime/runtime'
import { toCamel } from '../lib/camel'

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

export interface MCPServer {
  name: string
  command: string
  args?: string[]
  env?: Record<string, string>
  transport?: string
  source: string
  enabled: boolean
}

export interface Skill {
  name: string
  description: string
  path: string
  content: string
}

export interface PlanFile {
  name: string
  path: string
  size: number
  modifiedAt: number
  summary: string
}

export interface FileEdit {
  path: string
  backupPath: string
  timestamp: number
  originalHash?: string
  version: number
}

export interface Plugin {
  name: string
  enabled: boolean
  source: string
}

export interface DailyActivity {
  date: string
  messageCount: number
  sessionCount: number
  toolCallCount: number
}

export interface ModelTokenUsage {
  model: string
  date: string
  inputTokens: number
  outputTokens: number
  cacheRead?: number
  cacheWrite?: number
}

export interface SlashCommand {
  command: string
  args?: string
  timestamp: number
  cwd?: string
}

export interface SessionTokenDelta {
  input: number
  output: number
  cache?: number
}

export interface SessionToolUse {
  name: string
  input?: string
  output?: string
  status?: string
}

export interface SessionEvent {
  type: string
  subtype?: string
  timestamp: number
  role?: string
  content?: string
  thinking?: string
  model?: string
  tokens?: SessionTokenDelta
  tool?: SessionToolUse
  error?: string
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

// —— 灵动岛 (Eye) ——

export type EyeMode = 'dormant' | 'active' | 'storm' | 'alert'

export type FlashType =
  | 'complete'
  | 'cost_alert'
  | 'error'
  | 'conflict'
  | 'born'
  | 'death'

export interface Flash {
  id: string
  type: FlashType
  tribe: string
  content: string
  createdAt: number
}

export interface EyeState {
  mode: EyeMode
  running: string[]
  flashing: Flash[]
  updatedAt: number
}

// —— Claude Hooks ——

// Claude Code 通过 stdin 传给 hook 的 JSON。
// 字段是"广覆盖"——所有事件共有 + 各事件特有。
// 参考 backend/hooks/types.go HookPayload。
export interface HookPayload {
  sessionId?: string
  transcriptPath?: string
  cwd?: string
  hookEventName?: string
  toolName?: string
  toolInput?: Record<string, unknown>
  toolResponse?: Record<string, unknown>
  notificationType?: string
  message?: string
  stopReason?: string
  agentId?: string
  agentType?: string
  agentTranscriptPath?: string
  userPrompt?: string
  trigger?: string
  customInstructions?: string
}

export interface HooksInstallResult {
  added: string[]
  skipped: string[]
}

// Permission 规则集合
export interface Permissions {
  allow: string[]
  deny: string[]
  ask: string[]
}

// —— 类型安全的 Wails 桥 ——

declare global {
  interface Window {
    go: {
      backend: {
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
          ReadSession(id: string, sessionId: string): Promise<SessionEvent[]>
          ListMCPServers(id: string): Promise<MCPServer[]>
          ListSkills(id: string): Promise<Skill[]>
          ListPlans(id: string): Promise<PlanFile[]>
          ListFileHistory(id: string): Promise<FileEdit[]>
          RestoreFile(id: string, edit: FileEdit): Promise<void>
          ListDailyActivity(id: string): Promise<DailyActivity[]>
          ListModelTokenUsage(id: string): Promise<ModelTokenUsage[]>
          RecentSlashCommands(id: string, n: number): Promise<SlashCommand[]>
          ListPlugins(id: string): Promise<Plugin[]>
          StreamLatestSession(id: string): Promise<void>
          StopLatestSession(id: string): Promise<void>

          // 灵动岛
          GetEyeState(): Promise<EyeState>
          ConsumeEyeFlash(id: string): Promise<boolean>
          ClearEyeFlashes(): Promise<void>

          // Claude Hooks
          InstallHooks(): Promise<HooksInstallResult>
          UninstallHooks(): Promise<HooksInstallResult>
          IsHooksInstalled(): Promise<boolean>
          HookServerPort(): Promise<number>
          GetPermissions(): Promise<Permissions>
          TogglePermission(category: string, rule: string): Promise<Permissions>
        }
      }
    }
    runtime: {
      EventsOn: typeof EventsOn
      EventsEmit(eventName: string, ...data: unknown[]): void
    }
  }
}

export function wailsAvailable(): boolean {
  return typeof window !== 'undefined' && !!window.go?.backend?.App
}

export async function getLand(): Promise<Record<string, Tribe>> {
  if (!wailsAvailable()) return {}
  return toCamel<Record<string, Tribe>>(await window.go.backend.App.GetLand())
}

export async function getTribe(id: string): Promise<Tribe | null> {
  if (!wailsAvailable()) return null
  try {
    return toCamel<Tribe>(await window.go.backend.App.GetTribe(id))
  } catch {
    return null
  }
}

export async function getTribeMeta(id: string): Promise<TribeMeta | null> {
  if (!wailsAvailable()) return null
  try {
    return await window.go.backend.App.GetTribeMeta(id)
  } catch {
    return null
  }
}

export async function getForge(): Promise<Forge> {
  if (!wailsAvailable()) return { dailyBudget: 0, todaySpent: 0, byTribe: {}, byModel: {} }
  return window.go.backend.App.GetForge()
}

export async function readTribeConfigDNA(id: string): Promise<ConfigDNA | null> {
  if (!wailsAvailable()) return null
  try {
    return toCamel<ConfigDNA>(await window.go.backend.App.ReadTribeConfigDNA(id))
  } catch {
    return null
  }
}

export async function listSessions(id: string): Promise<SessionShard[]> {
  if (!wailsAvailable()) return []
  try {
    return toCamel<SessionShard[]>(await window.go.backend.App.ListSessions(id))
  } catch {
    return []
  }
}

export async function getTribeCapabilities(id: string): Promise<Capabilities | null> {
  if (!wailsAvailable()) return null
  try {
    return toCamel<Capabilities>(await window.go.backend.App.GetTribeCapabilities(id))
  } catch {
    return null
  }
}

export async function getAllCapabilities(): Promise<Record<string, Capabilities>> {
  if (!wailsAvailable()) return {}
  try {
    return toCamel<Record<string, Capabilities>>(await window.go.backend.App.GetAllCapabilities())
  } catch {
    return {}
  }
}

export async function writeTribeConfig(id: string, dna: ConfigDNA): Promise<boolean> {
  if (!wailsAvailable()) return false
  try {
    await window.go.backend.App.WriteTribeConfig(id, dna)
    return true
  } catch {
    return false
  }
}

// 19 项能力对应的 API 包装
// 注意：list 类接口永远返回 []，绝不返回 null（防止下游 items.length crash）
// 同时 camelCase 化（toCamel）以匹配我们手写的 camelCase TS 类型
async function safeList<T>(p: Promise<T[] | null> | undefined): Promise<T[]> {
  if (!wailsAvailable()) return []
  try {
    const r = await p
    return toCamel<T[]>((r ?? []) as T[])
  } catch {
    return []
  }
}

export const listMCPServers = (id: string) =>
  safeList(window.go.backend.App.ListMCPServers(id))
export const listSkills = (id: string) =>
  safeList(window.go.backend.App.ListSkills(id))
export const listPlans = (id: string) =>
  safeList(window.go.backend.App.ListPlans(id))
export const listFileHistory = (id: string) =>
  safeList(window.go.backend.App.ListFileHistory(id))
export const restoreFile = (id: string, edit: FileEdit) =>
  wailsAvailable() ? window.go.backend.App.RestoreFile(id, edit) : Promise.resolve()
export const listDailyActivity = (id: string) =>
  safeList(window.go.backend.App.ListDailyActivity(id))
export const listModelTokenUsage = (id: string) =>
  safeList(window.go.backend.App.ListModelTokenUsage(id))
export const recentSlashCommands = (id: string, n: number) =>
  safeList(window.go.backend.App.RecentSlashCommands(id, n))
export const listPlugins = (id: string) =>
  safeList(window.go.backend.App.ListPlugins(id))
export const readSession = (id: string, sessionId: string) =>
  safeList(window.go.backend.App.ReadSession(id, sessionId))
export const streamLatestSession = (id: string) =>
  wailsAvailable() ? window.go.backend.App.StreamLatestSession(id) : Promise.resolve()
export const stopLatestSession = (id: string) =>
  wailsAvailable() ? window.go.backend.App.StopLatestSession(id) : Promise.resolve()

// —— Eye 灵动岛 API ——

const DEFAULT_EYE: EyeState = {
  mode: 'dormant',
  running: [],
  flashing: [],
  updatedAt: 0,
}

export async function getEyeState(): Promise<EyeState> {
  if (!wailsAvailable()) return DEFAULT_EYE
  try {
    return toCamel<EyeState>(await window.go.backend.App.GetEyeState())
  } catch {
    return DEFAULT_EYE
  }
}

export async function consumeEyeFlash(id: string): Promise<boolean> {
  if (!wailsAvailable()) return false
  try {
    return await window.go.backend.App.ConsumeEyeFlash(id)
  } catch {
    return false
  }
}

export async function clearEyeFlashes(): Promise<void> {
  if (!wailsAvailable()) return
  try {
    await window.go.backend.App.ClearEyeFlashes()
  } catch {
    /* noop */
  }
}

// —— Claude Hooks API ——

export async function installHooks(): Promise<HooksInstallResult | null> {
  if (!wailsAvailable()) return null
  try {
    return toCamel<HooksInstallResult>(await window.go.backend.App.InstallHooks())
  } catch {
    return null
  }
}

export async function uninstallHooks(): Promise<HooksInstallResult | null> {
  if (!wailsAvailable()) return null
  try {
    return toCamel<HooksInstallResult>(await window.go.backend.App.UninstallHooks())
  } catch {
    return null
  }
}

export async function isHooksInstalled(): Promise<boolean> {
  if (!wailsAvailable()) return false
  try {
    return await window.go.backend.App.IsHooksInstalled()
  } catch {
    return false
  }
}

export async function hookServerPort(): Promise<number> {
  if (!wailsAvailable()) return 0
  try {
    return await window.go.backend.App.HookServerPort()
  } catch {
    return 0
  }
}

// —— Permission 规则 API ——

const EMPTY_PERMS: Permissions = {allow: [], deny: [], ask: []}

export async function getPermissions(): Promise<Permissions> {
  if (!wailsAvailable()) return EMPTY_PERMS
  try {
    return toCamel<Permissions>(await window.go.backend.App.GetPermissions())
  } catch {
    return EMPTY_PERMS
  }
}

export async function togglePermission(category: string, rule: string): Promise<Permissions | null> {
  if (!wailsAvailable()) return null
  try {
    return toCamel<Permissions>(await window.go.backend.App.TogglePermission(category, rule))
  } catch {
    return null
  }
}
