// 大陆俯瞰——主界面。
// Canvas 渲染等距地图，hover 高亮，点击进入部落。

import {useEffect, useRef, useState} from 'react'
import {useAland} from '../stores/alandStore'
import {hitTest, renderLand, type Camera} from '../canvas/IsometricEngine'
import {DEFAULT_PLACEMENTS} from '../canvas/layouts'

export function Overlook() {
  const tribes = useAland(s => s.tribes)
  const meta = useAland(s => s.meta)
  const enterTribe = useAland(s => s.enterTribe)

  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [hover, setHover] = useState<string | null>(null)
  const cameraRef = useRef<Camera>({x: 0, y: 0, zoom: 1})
  const sizeRef = useRef({w: 0, h: 0})

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
    <div style={{position: 'relative', width: '100%', height: '100%'}}>
      <canvas
        ref={canvasRef}
        style={{width: '100%', height: '100%', display: 'block'}}
        onMouseMove={onMove}
        onMouseLeave={() => setHover(null)}
        onClick={onClick}
      />

      {/* 顶部状态条（地图罗盘） */}
      <div
        className="draggable"
        style={{
          position: 'absolute',
          top: 0,
          left: 0,
          right: 0,
          height: 32,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '0 16px',
          fontFamily: 'var(--font-mono)',
          fontSize: 11,
          color: 'var(--aland-text-dim)',
          letterSpacing: 1,
          pointerEvents: 'none',
        }}
      >
        <span>ALAND · AGENT LAND</span>
        <span>{new Date().toLocaleTimeString()}</span>
      </div>

      {/* hover 提示 */}
      {hover && meta[hover] && (
        <div
          className="interactive"
          style={{
            position: 'absolute',
            bottom: 24,
            left: '50%',
            transform: 'translateX(-50%)',
            padding: '8px 16px',
            background: 'rgba(10, 14, 26, 0.7)',
            border: `1px solid ${meta[hover].themeColor}55`,
            borderRadius: 4,
            fontFamily: 'var(--font-mono)',
            fontSize: 12,
            color: meta[hover].accentColor,
            backdropFilter: 'blur(8px)',
            pointerEvents: 'auto',
          }}
          onClick={onClick}
        >
          {meta[hover].name} · click to enter →
        </div>
      )}

      {/* 底部状态栏 */}
      <div
        className="draggable"
        style={{
          position: 'absolute',
          bottom: 0,
          left: 0,
          right: 0,
          height: 24,
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'center',
          gap: 24,
          fontFamily: 'var(--font-mono)',
          fontSize: 10,
          color: 'var(--aland-text-faint)',
          letterSpacing: 1,
          pointerEvents: 'none',
        }}
      >
        <span>TRIBES: {Object.keys(tribes).length || 0}</span>
        <span>·</span>
        <span>RUNNING: {Object.values(tribes).filter(t => t.status === 'running' || t.status === 'busy').length}</span>
      </div>
    </div>
  )
}
