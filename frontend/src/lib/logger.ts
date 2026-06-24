// 前端结构化 logger。
// 与后端 core/logger.go 对齐：四个等级 + 字段约定。
//
// 调用示例：
//   logger.info('tribe born', {id: 'claude', pid: 1234})
//   logger.error('fetch failed', err)

type Level = 'debug' | 'info' | 'warn' | 'error'

const LEVEL_RANK: Record<Level, number> = {debug: 0, info: 1, warn: 2, error: 3}

// 开发期：info；生产期可调成 warn
let currentLevel: Level = 'info'

export function setLevel(level: Level) {
  currentLevel = level
}

function shouldLog(level: Level): boolean {
  return LEVEL_RANK[level] >= LEVEL_RANK[currentLevel]
}

function emit(level: Level, msg: string, fields?: object) {
  if (!shouldLog(level)) return
  const line = {
    t: new Date().toISOString(),
    level,
    msg,
    ...(fields ?? {}),
  }
  const out = level === 'error' ? console.error : level === 'warn' ? console.warn : console.log
  out(JSON.stringify(line))
}

export const logger = {
  debug: (msg: string, fields?: object) => emit('debug', msg, fields),
  info: (msg: string, fields?: object) => emit('info', msg, fields),
  warn: (msg: string, fields?: object) => emit('warn', msg, fields),
  error: (msg: string, fieldsOrError?: object | unknown) => {
    if (fieldsOrError instanceof Error) {
      emit('error', msg, {err: fieldsOrError.message, stack: fieldsOrError.stack})
    } else {
      emit('error', msg, fieldsOrError as object)
    }
  },
}
