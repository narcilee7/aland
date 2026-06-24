// 大陆俯瞰——主界面。
// 地图 + 右侧 TribeDock 双视图，hover 高亮，点击进入部落。

import {useEffect, useRef, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {hitTest, renderLand, type Camera} from '../canvas/IsometricEngine'
import {DEFAULT_PLACEMENTS} from '../canvas/layouts'
import {TribeDock} from './TribeDock'

interface OverlookProps {
  onOpenForge?: () => void
  onOpenMatrix?: () => void
  onOpenSpotlight?: () => void
  onOpenEye?: () => void
}

export function Overlook({onOpenForge, onOpenMatrix, onOpenSpotlight, onOpenEye}: OverlookProps = {}) {
  // 只订阅函数（稳定的引用），数据走 ref——这样 tribe:vital 不会触发 useEffect 重跑
  // 避免 canvas 每秒被 reset 闪烁
  const enterTribe = useAland(s => s.enterTribe)
  const toggleSpotlight = useAland(s => s.toggleSpotlight)

  // 数据 ref：tribe:vital 触发时 ref 刷新，render loop 读 ref 拿最新值
  const tribesRef = useRef(useAland.getState().tribes)
  const metaRef = useRef(useAland.getState().meta)
  useEffect(() => {
    const unsub = useAland.subscribe(state => {
      tribesRef.current = state.tribes
      metaRef.current = state.meta
    })
    return unsub
  }, [])

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [hover, setHover] = useState<string | null>(null)
  const hoverRef = useRef<string | null>(null)
  hoverRef.current = hover
  const cameraRef = useRef<Camera>({x: 0, y: 0, zoom: 1})
  const sizeRef = useRef({w: 0, h: 0})

  // 渲染循环——只在 mount 时跑一次
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    let raf = 0
    const dpr = window.devicePixelRatio || 1

    const resize = () => {
      const rect = canvas.getBoundingClientRect()
      sizeRef.current = {w: rect.width, h: rect.height}
      canvas.width = rect.width * dpr
      canvas.height = rect.height * dpr
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    }
    resize()
    const ro = new ResizeObserver(resize)
    ro.observe(canvas)

    const tick = (t: number) => {
      const {w, h} = sizeRef.current
      renderLand(ctx, {
        placements: DEFAULT_PLACEMENTS,
        tribes: tribesRef.current,
        meta: metaRef.current,
        camera: cameraRef.current,
        hoverId: hoverRef.current,
        width: w,
        height: h,
        time: t,
      })
      raf = requestAnimationFrame(tick)
    }
    raf = requestAnimationFrame(tick)
    return () => {
      cancelAnimationFrame(raf)
      ro.disconnect()
    }
  }, []) // 只 mount 一次

  const onMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const mx = e.clientX - rect.left
    const my = e.clientY - rect.top
    const id = hitTest(mx, my, DEFAULT_PLACEMENTS, metaRef.current, cameraRef.current, rect.width, rect.height)
    setHover(id)
  }

  const onClick = () => {
    if (hover) enterTribe(hover)
  }

  return (
    <div className="relative w-full h-full">
      <canvas
        ref={canvasRef}
        className="block w-full h-full"
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
        onClick={onClick}
      />

      {/* 顶部罗盘 */}
      <div className="draggable absolute top-0 left-0 h-8 flex items-center px-4 font-mono text-[11px] tracking-wider text-ink-dim pointer-events-none">
        <span>Aland · Agent Land</span>
      </div>

      {/* 顶部时间 */}
      <div className="draggable absolute top-0 left-1/2 -translate-x-1/2 h-8 flex items-center px-4 font-mono text-[11px] tracking-wider text-ink-dim pointer-events-none">
        <span>{new Date().toLocaleTimeString()}</span>
      </div>

      {/* 右侧 TribeDock：所有部落 + 工具栏 + Active 提示 */}
      <TribeDock
        onOpenForge={onOpenForge}
        onOpenMatrix={onOpenMatrix}
        onOpenEye={onOpenEye}
        onOpenSpotlight={onOpenSpotlight ?? toggleSpotlight}
      />

      {/* 底部：Cmd+Shift+A 提示 */}
      <div className="draggable absolute bottom-2 left-4 font-mono text-[10px] tracking-wider text-ink-faint/60 pointer-events-none">
        <kbd className="rounded bg-white/5 border border-white/10 px-1.5 py-0.5 mr-1.5">⌘⇧A</kbd>
        Spotlight
      </div>
    </div>
  )
}
