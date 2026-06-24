// 前端事件层——与后端 events/events.go 严格对齐。
// 改一处记着改另一处（强约束：常量值必须完全一致）。

import type {EyeMode, Flash, HookPayload, SessionEvent, Tribe} from './wails'
import {toCamel} from '../lib/camel'

// 事件名常量。
export const EventName = {
  TribeVital: 'tribe:vital',
  TribeBorn: 'tribe:born',
  TribeDeath: 'tribe:death',
  FSChange: 'fs:change',
  SpotlightToggle: 'spotlight:toggle',
  SessionEvent: 'session:event',
  EyeUpdate: 'eye:update',
  EyeFlash: 'eye:flash',
  HookEvent: 'claude:hook',
} as const

export type EventName = (typeof EventName)[keyof typeof EventName]

// Payload 类型——与后端 events.TribeLifecycleEvent / FSChangeEvent / SpotlightToggleEvent / EyeUpdateEvent / EyeFlashEvent 镜像。
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

export interface EyeUpdateEvent {
  mode: EyeMode
  running: string[]
  updatedAt: number
}

export interface EyeFlashEvent {
  flash: Flash
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
  on<TribeSnapshotMap>(EventName.TribeVital, snap => cb(toCamel<TribeSnapshotMap>(snap)))

export const onTribeBorn = (cb: (e: TribeLifecycleEvent) => void) =>
  on<TribeLifecycleEvent>(EventName.TribeBorn, e => cb(toCamel<TribeLifecycleEvent>(e)))

export const onTribeDeath = (cb: (e: TribeLifecycleEvent) => void) =>
  on<TribeLifecycleEvent>(EventName.TribeDeath, e => cb(toCamel<TribeLifecycleEvent>(e)))

export const onFSChange = (cb: (e: FSChangeEvent) => void) =>
  on<FSChangeEvent>(EventName.FSChange, e => cb(toCamel<FSChangeEvent>(e)))

export const onSpotlightToggle = (cb: (e: SpotlightToggleEvent) => void) =>
  on<SpotlightToggleEvent>(EventName.SpotlightToggle, e => cb(toCamel<SpotlightToggleEvent>(e)))

export const onSessionEvent = (cb: (e: SessionEvent) => void) =>
  on<SessionEvent>(EventName.SessionEvent, e => cb(toCamel<SessionEvent>(e)))

export const onEyeUpdate = (cb: (e: EyeUpdateEvent) => void) =>
  on<EyeUpdateEvent>(EventName.EyeUpdate, e => cb(toCamel<EyeUpdateEvent>(e)))

export const onEyeFlash = (cb: (e: EyeFlashEvent) => void) =>
  on<EyeFlashEvent>(EventName.EyeFlash, e => cb(toCamel<EyeFlashEvent>(e)))

export const onHook = (cb: (p: HookPayload) => void) =>
  on<HookPayload>(EventName.HookEvent, p => cb(toCamel<HookPayload>(p)))
