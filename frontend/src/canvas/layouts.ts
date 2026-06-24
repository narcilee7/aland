// 部落在地形上的位置。
// M0：四角分布；M1 起：4 部落撑满 4 角。

import type {TribePlacement} from './IsometricEngine'

/** 默认 4 个部落的位置。 */
export const DEFAULT_PLACEMENTS: TribePlacement[] = [
  {id: 'claude', x: -3, y: -3, z: 1.5}, // 高地（北西）— 古典派
  {id: 'cursor', x: 3, y: -3, z: 0},    // 平原（北东）— 现代派
  {id: 'trae', x: -3, y: 3, z: 0},      // 竹林（南西）— 东方派
  // 港湾（南东）留给 v1 的 Kimi
  // {id: 'kimi', x: 3, y: 3, z: -0.5},
]
