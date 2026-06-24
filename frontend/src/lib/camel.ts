// Wails v2 生成的 JSON 是 PascalCase（直接用 Go 字段名），但 JS 习惯 camelCase。
// 这个模块做一次转换，调用方拿到的是 camelCase 数据。
//
// 例：{Features: [...], Process: true, ByModel: {...}}
//  → {features: [...], process: true, byModel: {...}}

// 单 key 转换：PascalCase → camelCase
// 例："Process" → "process"，"ByModel" → "byModel"，"ANTHROPIC" → "aNTHROPIC"（只处理首字母）
function pascalToCamel(key: string): string {
  if (!key) return key
  return key[0].toLowerCase() + key.slice(1)
}

// 递归转换对象
export function toCamel<T = unknown>(input: unknown): T {
  if (input === null || input === undefined) return input as T
  if (Array.isArray(input)) {
    return input.map(item => toCamel(item)) as T
  }
  if (typeof input === 'object') {
    const out: Record<string, unknown> = {}
    for (const [k, v] of Object.entries(input as Record<string, unknown>)) {
      out[pascalToCamel(k)] = toCamel(v)
    }
    return out as T
  }
  return input as T
}
