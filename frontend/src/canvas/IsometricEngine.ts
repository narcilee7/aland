// 等距投影 Canvas 2D 引擎（M0 版本）。
//
// 设计取舍：不用 Three.js / WebGL。Canvas 2D + 精灵图足够描绘四个部落。
// 性能策略：背景预渲染，部落精灵每帧绘制，hover/click 命中检测在画布坐标。
//
// 坐标系：
//   - 世界坐标 (worldX, worldY)：等距网格中的"逻辑"位置
//   - 屏幕坐标 (screenX, screenY)：最终绘制到 canvas 上的像素
//
// 投影公式（30° 等距）：
//   screenX = (worldX - worldY) * TILE_W / 2
//   screenY = (worldX + worldY) * TILE_H / 2 - worldZ

import type {Tribe, TribeMeta} from '../api/wails'

export interface TribePlacement {
  id: string
  /** 世界坐标 X（逻辑单位） */
  x: number
  /** 世界坐标 Y */
  y: number
  /** 海拔，影响绘制高度 */
  z: number
}

export interface Camera {
  x: number
  y: number
  zoom: number
}

export const TILE_W = 64
export const TILE_H = 32

/** 把世界坐标投影到屏幕坐标（中心点） */
export function project(x: number, y: number, z = 0, camera: Camera = {x: 0, y: 0, zoom: 1}): {sx: number; sy: number} {
  const sx = ((x - y) * TILE_W) / 2
  const sy = ((x + y) * TILE_H) / 2 - z
  return {sx: sx * camera.zoom + camera.x, sy: sy * camera.zoom + camera.y}
}

/** 屏幕坐标反投影到世界坐标（用于命中检测） */
export function unproject(sx: number, sy: number, camera: Camera = {x: 0, y: 0, zoom: 1}): {x: number; y: number} {
  const ax = (sx - camera.x) / camera.zoom
  const ay = (sy - camera.y) / camera.zoom
  const x = (ax / (TILE_W / 2) + ay / (TILE_H / 2)) / 2
  const y = (ay / (TILE_H / 2) - ax / (TILE_W / 2)) / 2
  return {x, y}
}

/** 把 hex 顶点画到 canvas。
 * cpu: 0-100 真实 CPU 占用，影响呼吸频率和振幅。
 */
function drawIsoBlob(
  ctx: CanvasRenderingContext2D,
  cx: number,
  cy: number,
  radius: number,
  fill: string,
  glow: string,
  time: number,
  pulse: number,
  cpu: number,
) {
  // 光晕半径随 CPU 增大（高负载时发热）
  const glowSize = radius * (2.4 + cpu * 0.01)

  // 光晕
  const grd = ctx.createRadialGradient(cx, cy, 0, cx, cy, glowSize)
  grd.addColorStop(0, glow)
  grd.addColorStop(1, 'rgba(0,0,0,0)')
  ctx.fillStyle = grd
  ctx.beginPath()
  ctx.arc(cx, cy, glowSize, 0, Math.PI * 2)
  ctx.fill()

  // 主体：等距椭圆，呼吸频率随 CPU 加快
  // CPU=0 时周期 4s；CPU=100 时周期 ~1s
  const period = 4000 / (1 + cpu * 0.04)
  const breath = 1 + 0.03 * Math.sin(time / period)
  const r = radius * breath
  ctx.save()
  ctx.translate(cx, cy)
  ctx.scale(1, 0.55) // 等距压扁
  ctx.fillStyle = fill
  ctx.beginPath()
  ctx.arc(0, 0, r, 0, Math.PI * 2)
  ctx.fill()

  // 中心核——运行中时更亮，且 CPU 高时偏白
  if (pulse > 0) {
    const intensity = 0.2 + 0.3 * pulse + cpu * 0.002
    ctx.fillStyle = `rgba(255,255,255,${Math.min(intensity, 0.9)})`
    ctx.beginPath()
    ctx.arc(0, -r * 0.15, r * 0.35, 0, Math.PI * 2)
    ctx.fill()
  }
  ctx.restore()
}

export interface RenderInput {
  placements: TribePlacement[]
  tribes: Record<string, Tribe>
  meta: Record<string, TribeMeta>
  camera: Camera
  hoverId: string | null
  width: number
  height: number
  time: number // ms
}

export function renderLand(ctx: CanvasRenderingContext2D, input: RenderInput) {
  const {placements, tribes, meta, camera, hoverId, width, height, time} = input

  // —— 1. 背景：极深的靛蓝 → 墨绿径向渐变 ——
  const bg = ctx.createRadialGradient(width / 2, height / 2, 0, width / 2, height / 2, Math.max(width, height))
  bg.addColorStop(0, '#0d1f15')
  bg.addColorStop(1, '#0a0e1a')
  ctx.fillStyle = bg
  ctx.fillRect(0, 0, width, height)

  // —— 2. 大陆地形（同心等距环）——
  const cx = width / 2
  const cy = height / 2
  for (let r = 1; r <= 5; r++) {
    ctx.strokeStyle = `rgba(100, 116, 139, ${0.05 + 0.01 * (5 - r)})`
    ctx.lineWidth = 1
    ctx.beginPath()
    // 用四个等距点连成一个菱形
    const angles: [number, number][] = [[1, 0], [0, 1], [-1, 0], [0, -1]]
    const radius = r * 4
    angles.forEach((a, i) => {
      const p = project(a[0] * radius, a[1] * radius, 0, {x: cx, y: cy, zoom: camera.zoom})
      if (i === 0) ctx.moveTo(p.sx, p.sy)
      else ctx.lineTo(p.sx, p.sy)
    })
    ctx.closePath()
    ctx.stroke()
  }

  // —— 3. 部落精灵（按 y+x 排序：远的先画，近的后画）——
  const sorted = [...placements].sort((a, b) => a.x + a.y - (b.x + b.y))

  for (const place of sorted) {
    const t = tribes[place.id]
    const m = meta[place.id]
    if (!m) continue

    const {sx, sy} = project(place.x, place.y, place.z, {x: cx, y: cy, zoom: camera.zoom})
    const isHover = place.id === hoverId
    const isRunning = t?.status === 'running' || t?.status === 'busy'
    const cpu = t?.vital.cpu ?? 0
    const pulse = isRunning ? 0.5 + 0.5 * Math.sin(time / 600) : 0
    // 部落实体放大（基础 32，hover 40），更显眼
    const radius = (isHover ? 40 : 32) * camera.zoom

    drawIsoBlob(ctx, sx, sy, radius, m.themeColor, m.accentColor + '55', time, pulse, cpu)

    // 部落名——永远显示在实体下方，不再依赖 hover
    ctx.fillStyle = isHover ? '#ffffff' : 'rgba(226, 232, 240, 0.95)'
    ctx.font = `${isHover ? '700' : '500'} ${14 * camera.zoom}px 'JetBrains Mono', monospace`
    ctx.textAlign = 'center'
    ctx.textBaseline = 'top'
    // 名字 + ECO 副标签
    ctx.fillText(m.name.toUpperCase(), sx, sy + radius + 8)
    ctx.fillStyle = 'rgba(148, 163, 184, 0.6)'
    ctx.font = `400 ${10 * camera.zoom}px 'JetBrains Mono', monospace`
    ctx.fillText(m.eco, sx, sy + radius + 26)

    // 状态点 + halo：右上角，跑时大并加光晕
    if (t) {
      const statusColor =
        t.status === 'running' ? '#22c55e' : t.status === 'busy' ? '#f59e0b' : t.status === 'error' ? '#ef4444' : '#475569'
      const dotR = isRunning ? 5 : 3
      if (isRunning) {
        // 光晕
        ctx.fillStyle = statusColor + '44'
        ctx.beginPath()
        ctx.arc(sx + radius * 0.85, sy - radius * 0.85, dotR * 3, 0, Math.PI * 2)
        ctx.fill()
      }
      ctx.fillStyle = statusColor
      ctx.beginPath()
      ctx.arc(sx + radius * 0.85, sy - radius * 0.85, dotR * camera.zoom, 0, Math.PI * 2)
      ctx.fill()
    }
  }

  // —— 4. 中央圣所：Forge 液面（绿色基调）——
  const center = project(0, 0, 0, {x: cx, y: cy, zoom: camera.zoom})
  const forgeRadius = 30 * camera.zoom
  // 液面：翠绿径向
  const forgeGrd = ctx.createRadialGradient(
    center.sx,
    center.sy - forgeRadius * 0.2,
    0,
    center.sx,
    center.sy,
    forgeRadius,
  )
  forgeGrd.addColorStop(0, 'rgba(34, 197, 94, 0.25)')
  forgeGrd.addColorStop(0.6, 'rgba(34, 197, 94, 0.08)')
  forgeGrd.addColorStop(1, 'rgba(34, 197, 94, 0)')
  ctx.fillStyle = forgeGrd
  ctx.beginPath()
  ctx.arc(center.sx, center.sy, forgeRadius, 0, Math.PI * 2)
  ctx.fill()
  // 边框
  ctx.strokeStyle = 'rgba(212, 168, 83, 0.35)'
  ctx.lineWidth = 1
  ctx.beginPath()
  ctx.arc(center.sx, center.sy, forgeRadius, 0, Math.PI * 2)
  ctx.stroke()
  // 中央文字
  ctx.fillStyle = 'rgba(212, 168, 83, 0.7)'
  ctx.font = `${9 * camera.zoom}px 'JetBrains Mono', monospace`
  ctx.textAlign = 'center'
  ctx.textBaseline = 'middle'
  ctx.fillText('FORGE', center.sx, center.sy - 4)
  ctx.fillText('○', center.sx, center.sy + 8)
}

/** 根据鼠标位置判断 hover 到哪个部落（轴对齐包围盒近似） */
export function hitTest(
  mx: number,
  my: number,
  placements: TribePlacement[],
  meta: Record<string, TribeMeta>,
  camera: Camera,
  width: number,
  height: number,
): string | null {
  const cx = width / 2
  const cy = height / 2
  let best: {id: string; dist: number} | null = null
  for (const p of placements) {
    if (!meta[p.id]) continue
    const proj = project(p.x, p.y, p.z, {x: cx, y: cy, zoom: camera.zoom})
    const dx = mx - proj.sx
    const dy = my - proj.sy
    const d = Math.sqrt(dx * dx + (dy * 1.8) ** 2) // Y 方向压扁匹配椭圆
    if (d < 40 * camera.zoom && (!best || d < best.dist)) {
      best = {id: p.id, dist: d}
    }
  }
  return best?.id ?? null
}
