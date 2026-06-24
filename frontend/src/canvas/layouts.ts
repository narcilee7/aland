// 部落在地形上的位置。
// v1 是四角分布；v2 改成环形大陆围绕中央圣所。

import type {TribePlacement} from './IsometricEngine'

/** 默认 4 个部落的位置（M0）。 */
export const DEFAULT_PLACEMENTS: TribePlacement[] = [
  {id: 'claude', x: -3, y: -3, z: 1.5}, // 高地（北西）
  // 后续 v1 加入：
  // {id: 'cursor', x: 3, y: -3, z: 0},  // 平原网格（北东）
  // {id: 'trae',   x: -3, y: 3, z: 0},  // 竹林（南西）
  // {id: 'kimi',   x: 3, y: 3, z: -0.5}, // 港湾（南东）
]
