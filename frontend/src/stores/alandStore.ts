// Aland 全局状态。
// 视图模式 + 部落数据 + 能力清单 + 选中态 + Spotlight/Forge/Eye 开关。
// 用 Zustand 而不是 Redux：够用，且比 Context 更适合高频更新。

import {create} from 'zustand'
import type {Capabilities, EyeState, HookPayload, Tribe, TribeMeta} from '../api/wails'
import {
  clearEyeFlashes,
  consumeEyeFlash,
  getAllCapabilities,
  getEyeState,
  getLand,
  getTribeMeta,
  wailsAvailable,
} from '../api/wails'
import {onEyeFlash, onEyeUpdate, onHook, onTribeBorn, onTribeDeath, onTribeVital} from '../api/events'
import {logger} from '../lib/logger'

/**
 * 等 Wails runtime 注入到 window.go。
 * bundled app 启动慢，window.go 可能在 boot 第一次跑时还没 ready。
 */
async function waitForWails(timeoutMs: number): Promise<boolean> {
  const start = Date.now()
  let attempt = 0
  while (Date.now() - start < timeoutMs) {
    if (wailsAvailable()) return true
    attempt++
    if (attempt % 10 === 0) {
      // 每秒 log 一次当前状态（方便 debug 是否在 wails runtime 里）
      const inWails = typeof window !== 'undefined' && !!window.go
      const hasBackend = typeof window !== 'undefined' && !!window.go?.backend
      const goKeys = typeof window !== 'undefined' && window.go ? Object.keys(window.go) : []
      logger.info('waiting for wails runtime', {
        attempt,
        inWails,
        hasBackend,
        goKeys,
        hint: !inWails
          ? '不在 Wails webview 里——是不是在普通浏览器？'
          : !hasBackend
            ? 'window.go 有但没 backend 键——包名匹配？'
            : 'window.go.backend 有但 .App 是 undefined',
      })
    }
    await new Promise(r => setTimeout(r, 100))
  }
  return false
}

export type View = 'overlook' | 'tribe'

interface AlandState {
  // 视图
  view: View
  activeTribe: string | null
  spotlight: boolean
  forgeOpen: boolean
  matrixOpen: boolean
  eyeOpen: boolean

  // 大陆数据
  tribes: Record<string, Tribe>
  meta: Record<string, TribeMeta>
  caps: Record<string, Capabilities>

  // 灵动岛 (Eye)
  eye: EyeState

  // Claude Hooks 工具链
  toolChain: {
    nodes: ToolNode[]
    maxNodes: number
  }

  // 状态
  booted: boolean
  booting: boolean

  // Actions
  boot: () => Promise<void>
  enterTribe: (id: string) => void
  returnOverlook: () => void
  toggleSpotlight: () => void
  setSpotlight: (open: boolean) => void
  setForgeOpen: (open: boolean) => void
  setMatrixOpen: (open: boolean) => void
  setEyeOpen: (open: boolean) => void
  consumeFlash: (id: string) => Promise<void>
  clearFlashes: () => Promise<void>
  addToolNode: (p: HookPayload) => void
  clearToolChain: () => void
}

// ToolNode 是工具链里的一个节点——来自一次 hook 事件。
export interface ToolNode {
  id: string
  event: string // PreToolUse / PostToolUse / Notification / ...
  toolName?: string
  toolInput?: Record<string, unknown>
  toolResponse?: Record<string, unknown>
  message?: string
  userPrompt?: string
  at: number
}

const initialEye: EyeState = {
  mode: 'dormant',
  running: [],
  flashing: [],
  updatedAt: 0,
}

const TOOL_CHAIN_MAX = 100

export const useAland = create<AlandState>((set, get) => ({
  view: 'overlook',
  activeTribe: null,
  spotlight: false,
  forgeOpen: false,
  matrixOpen: false,
  eyeOpen: false,
  tribes: {},
  meta: {},
  caps: {},
  eye: initialEye,
  toolChain: {nodes: [], maxNodes: TOOL_CHAIN_MAX},
  booted: false,
  booting: false,

  async boot() {
    if (get().booted || get().booting) return
    set({booting: true})
    logger.info('aland booting', {
      href: typeof window !== 'undefined' ? window.location.href : '(no window)',
    })

    // 等 Wails runtime 注入（最多 15 秒）
    const wailsReady = await waitForWails(15000)
    if (!wailsReady) {
      const inWails = typeof window !== 'undefined' && !!window.go
      logger.error('wails runtime not ready after 15s', {
        inWails,
        hint: inWails
          ? 'window.go 存在但 window.go.backend.App 找不到——路径错？'
          : 'window.go 完全不存在——你在浏览器里打开的？要在 Aland .app 窗口里看',
      })
      set({booting: false})
      return
    }

    try {
      // 1. 拉大陆
      const land = (await getLand()) ?? {}
      const ids = Object.keys(land)

      // 2. 并发拉每个部落的 meta + caps
      const [metas, caps] = await Promise.all([
        Promise.all(ids.map(id => getTribeMeta(id))),
        getAllCapabilities(),
      ])

      const meta: Record<string, TribeMeta> = {}
      metas.forEach((m, i) => {
        if (m) meta[ids[i]] = m
      })
      set({tribes: land, meta, caps, booted: true, booting: false})
      logger.info('aland booted', {tribes: ids.length})

      // 订阅实时事件
      onTribeVital(snap => set({tribes: snap}))
      onTribeBorn(e => logger.info('tribe born', e))
      onTribeDeath(e => logger.info('tribe death', e))

      // 加载 Eye 初始状态 + 订阅事件
      const eye = await getEyeState()
      set({eye})
      onEyeUpdate(e =>
        set(s => ({eye: {...s.eye, mode: e.mode, running: e.running, updatedAt: e.updatedAt}})),
      )
      onEyeFlash(e =>
        set(s => ({eye: {...s.eye, flashing: [...s.eye.flashing, e.flash].slice(-16)}})),
      )

      // 订阅 Claude Hook 事件 → 入工具链
      onHook(p => get().addToolNode(p))
    } catch (e) {
      logger.error('aland boot failed', e)
      set({booting: false})
    }
  },

  enterTribe(id: string) {
    set({view: 'tribe', activeTribe: id, spotlight: false})
  },

  returnOverlook() {
    set({view: 'overlook', activeTribe: null})
  },

  toggleSpotlight() {
    set(s => ({spotlight: !s.spotlight}))
  },

  setSpotlight(open: boolean) {
    set({spotlight: open})
  },

  setForgeOpen(open: boolean) {
    set({forgeOpen: open})
  },

  setMatrixOpen(open: boolean) {
    set({matrixOpen: open})
  },

  setEyeOpen(open: boolean) {
    set({eyeOpen: open})
  },

  async consumeFlash(id: string) {
    // 本地乐观更新 + 后端确认
    set(s => ({eye: {...s.eye, flashing: s.eye.flashing.filter(f => f.id !== id)}}))
    await consumeEyeFlash(id)
  },

  async clearFlashes() {
    set(s => ({eye: {...s.eye, flashing: []}}))
    await clearEyeFlashes()
  },

  addToolNode(p) {
    const node: ToolNode = {
      id: `${p.hookEventName ?? 'ev'}-${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      event: p.hookEventName ?? 'unknown',
      toolName: p.toolName,
      toolInput: p.toolInput,
      toolResponse: p.toolResponse,
      message: p.message,
      userPrompt: p.userPrompt,
      at: Date.now(),
    }
    set(s => ({
      toolChain: {
        maxNodes: s.toolChain.maxNodes,
        nodes: [...s.toolChain.nodes, node].slice(-s.toolChain.maxNodes),
      },
    }))
  },

  clearToolChain() {
    set(s => ({toolChain: {...s.toolChain, nodes: []}}))
  },
}))
