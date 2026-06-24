// 大陆俯瞰——主界面。
// Canvas 渲染等距地图，hover 高亮，点击进入部落。

import { useEffect, useRef, useState } from 'react'
import { useAland } from '../stores/alandStore'
import { hitTest, renderLand, type Camera } from '../canvas/IsometricEngine'
import { DEFAULT_PLACEMENTS } from '../canvas/layouts'
import { Badge, Button } from './ui'

export function Overlook() {
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)
  const enterTribe = useAland(s => s.enterTribe)

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

  const runningCount = Object.values(tribes).filter(
    t => t.status === 'running' || t.status === 'busy',
  ).length

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
      <div className="draggable absolute top-0 inset-x-0 h-8 flex items-center justify-between px-4 font-mono text-[11px] tracking-wider text-ink-dim pointer-events-none">
        <span>Aland · Agent Land</span>
        <span>{new Date().toLocaleTimeString()}</span>
      </div>

      {/* hover 提示 */}
      {hover && meta[hover] && (
        <div
          className="interactive absolute bottom-12 left-1/2 -translate-x-1/2"
          style={
            {
              // 动态注入部落色到 CSS 变量
              ['--tribe-theme' as string]: meta[hover].themeColor,
              ['--tribe-accent' as string]: meta[hover].accentColor,
            } as React.CSSProperties
          }
        >
          <Button onClick={onClick} variant="default" size="md">
            {meta[hover].name} · click to enter →
          </Button>
        </div>
      )}

      {/* 底部状态栏 */}
      <div className="draggable absolute bottom-0 inset-x-0 h-6 flex items-center justify-center gap-6 font-mono text-[10px] tracking-wider text-ink-faint pointer-events-none">
        <span>TRIBES: {Object.keys(tribes).length || 0}</span>
        <span className="text-ink-faint/40">·</span>
        <span>RUNNING: {runningCount}</span>
        <span className="text-ink-faint/40">·</span>
        <span>{meta[hover ?? ''] ? `HOVER: ${meta[hover ?? ''].eco.toUpperCase()}` : 'IDLE'}</span>
      </div>
    </div>
  )
}

// 防止未使用的 Badge 引入被 tree-shake 警告
void Badge
