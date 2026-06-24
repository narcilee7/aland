// 前端事件层——与后端 events/events.go 严格对齐。
// 改一处记着改另一处（强约束：常量值必须完全一致）。

import type {Tribe} from './wails'

// 事件名常量。
export const EventName = {
  TribeVital: 'tribe:vital',
  TribeBorn: 'tribe:born',
  TribeDeath: 'tribe:death',
  FSChange: 'fs:change',
  SpotlightToggle: 'spotlight:toggle',
} as const

export type EventName = (typeof EventName)[keyof typeof EventName]

// Payload 类型——与后端 events.TribeLifecycleEvent / FSChangeEvent / SpotlightToggleEvent 镜像。
export interface TribeLifecycleEvent {
  id: string
  pid: number
  name: string
}

export interface FSChangeEvent {
  path: string
  op: string
}

export interface SpotlightToggleEvent {
  action: 'open' | 'close' | 'toggle'
}

export type TribeSnapshotMap = Record<string, Tribe>

// 类型安全的订阅函数。
// 闭包捕获 window.runtime 检查，未启动 wails 时返回 no-op。

declare global {
  interface Window {
    runtime: {
      EventsOn(eventName: string, callback: (...data: any[]) => void): () => void
      EventsEmit(eventName: string, ...data: unknown[]): void
    }
  }
}

function on<T>(name: string, cb: (payload: T) => void): () => void {
  if (typeof window === 'undefined' || !window.runtime) return () => {}
  return window.runtime.EventsOn(name, (payload: T) => cb(payload))
}

export const onTribeVital = (cb: (snap: TribeSnapshotMap) => void) =>
  on<TribeSnapshotMap>(EventName.TribeVital, cb)

export const onTribeBorn = (cb: (e: TribeLifecycleEvent) => void) =>
  on<TribeLifecycleEvent>(EventName.TribeBorn, cb)

export const onTribeDeath = (cb: (e: TribeLifecycleEvent) => void) =>
  on<TribeLifecycleEvent>(EventName.TribeDeath, cb)

export const onFSChange = (cb: (e: FSChangeEvent) => void) => on<FSChangeEvent>(EventName.FSChange, cb)

export const onSpotlightToggle = (cb: (e: SpotlightToggleEvent) => void) =>
  on<SpotlightToggleEvent>(EventName.SpotlightToggle, cb)
