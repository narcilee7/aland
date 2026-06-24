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
}

export function Overlook({onOpenForge, onOpenMatrix, onOpenSpotlight}: OverlookProps = {}) {
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)
  const enterTribe = useAland(s => s.enterTribe)
  const toggleSpotlight = useAland(s => s.toggleSpotlight)

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [hover, setHover] = useState<string | null>(null)
  const cameraRef = useRef<Camera>({ x: 0, y: 0, zoom: 1 })
  const sizeRef = useRef({ w: 0, h: 0 })

  // 渲染循环
  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    const ctx = canvas.getContext('2d')
    if (!ctx) return

    let raf = 0
    const dpr = window.devicePixelRatio || 1

    const resize = () => {
      const rect = canvas.getBoundingClientRect()
      sizeRef.current = { w: rect.width, h: rect.height }
      canvas.width = rect.width * dpr
      canvas.height = rect.height * dpr
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0)
    }
    resize()
    const ro = new ResizeObserver(resize)
    ro.observe(canvas)

    const tick = (t: number) => {
      const { w, h } = sizeRef.current
      renderLand(ctx, {
        placements: DEFAULT_PLACEMENTS,
        tribes,
        meta,
        camera: cameraRef.current,
        hoverId: hover,
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
  }, [tribes, meta, hover])

  const onMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    const rect = e.currentTarget.getBoundingClientRect()
    const mx = e.clientX - rect.left
    const my = e.clientY - rect.top
    const id = hitTest(mx, my, DEFAULT_PLACEMENTS, meta, cameraRef.current, rect.width, rect.height)
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
