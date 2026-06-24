// Aland 全局状态。
// 视图模式 + 部落数据 + 能力清单 + 选中态 + Spotlight/Forge 开关。
// 用 Zustand 而不是 Redux：够用，且比 Context 更适合高频更新。

import {create} from 'zustand'
import type {Capabilities, Tribe, TribeMeta} from '../api/wails'
import {getAllCapabilities, getLand, getTribeMeta} from '../api/wails'
import {onTribeBorn, onTribeDeath, onTribeVital} from '../api/events'
import {logger} from '../lib/logger'

export type View = 'overlook' | 'tribe'

interface AlandState {
  // 视图
  view: View
  activeTribe: string | null
  spotlight: boolean
  forgeOpen: boolean
  matrixOpen: boolean

  // 大陆数据
  tribes: Record<string, Tribe>
  meta: Record<string, TribeMeta>
  caps: Record<string, Capabilities>

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
}

export const useAland = create<AlandState>((set, get) => ({
  view: 'overlook',
  activeTribe: null,
  spotlight: false,
  forgeOpen: false,
  matrixOpen: false,
  tribes: {},
  meta: {},
  caps: {},
  booted: false,
  booting: false,

  async boot() {
    if (get().booted || get().booting) return
    set({booting: true})
    logger.info('aland booting')
    try {
      // 并发：取大陆 / 全部 meta / 全部 caps
      const [land, metas, caps] = await Promise.all([
        getLand(),
        Promise.resolve().then(async () => {
          const l = (await getLand()) ?? {}
          return Promise.all(Object.keys(l).map(id => getTribeMeta(id)))
        }),
        getAllCapabilities(),
      ])

      const safeLand = land ?? {}
      const ids = Object.keys(safeLand)
      const meta: Record<string, TribeMeta> = {}
      metas.forEach((m, i) => {
        if (m) meta[ids[i]] = m
      })
      set({tribes: safeLand, meta, caps, booted: true, booting: false})
      logger.info('aland booted', {tribes: ids.length})

      // 订阅实时事件
      onTribeVital(snap => set({tribes: snap}))
      onTribeBorn(e => logger.info('tribe born', e))
      onTribeDeath(e => logger.info('tribe death', e))
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
}))
