// 工具链可视化——Claude Code 每次 hook 事件（特别是 PreToolUse / PostToolUse）
// 在这里构成一条时序链。每个节点是一次 tool call，可展开看 input/output。
//
// 设计：
// - 时间倒序：最新在最上面
// - 每个节点显示 tool 名 + 摘要 + 耗时（Pre→Post 配对算）
// - 节点可展开：显示完整 input JSON + 截断的 output
// - 不同事件类型颜色不同：tool_use / tool_result / stop / notification
//
// 数据源：onHook 订阅 → 累积到一个上限的 ring buffer（默认 100 条）。

import {useEffect, useMemo, useState} from 'react'
import {useAland, type ToolNode} from '../stores/alandStore'
import {onHook} from '../api/events'
import type {HookPayload} from '../api/wails'
import {Card, CardContent, CardHeader, CardTitle, Badge} from './ui'
import {Wrench, FileText, Bell, StopCircle, ChevronDown, ChevronRight, Zap, AlertTriangle, User} from 'lucide-react'
import {logger} from '../lib/logger'

interface ToolChainProps {
  /** 最大保留节点数 */
  maxNodes?: number
}

export function ToolChain({maxNodes = 100}: ToolChainProps) {
  const chain = useAland(s => s.toolChain)
  const [expanded, setExpanded] = useState<Set<number>>(new Set())

  // 订阅 hook 事件，由 store 负责入队
  useEffect(() => {
    return onHook((p: HookPayload) => {
      // store 内部已经处理（addToolNode）；这里什么都不做也行
      // 保留这个 hook 方便将来扩展（如解析更细的状态）
      logger.debug('hook received', {event: p.hookEventName, tool: p.toolName})
    })
  }, [])

  const toggle = (i: number) => {
    setExpanded(prev => {
      const next = new Set(prev)
      if (next.has(i)) next.delete(i)
      else next.add(i)
      return next
    })
  }

  const nodes = useMemo(() => chain.nodes.slice().reverse(), [chain.nodes])

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Wrench className="h-4 w-4" />
          Tool Chain
          <span className="text-ink-faint normal-case tracking-wider text-[10px]">
            {chain.nodes.length}/{maxNodes}
          </span>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-auto min-h-0">
        {nodes.length === 0 ? (
          <div className="text-xs text-ink-faint font-mono py-6 text-center">
            等待 Claude Code 的 hook 事件……
            <br />
            <span className="text-[10px]">
              （Settings → 安装 hooks 来启用）
            </span>
          </div>
        ) : (
          <div className="space-y-1">
            {nodes.map((n: ToolNode, idx: number) => {
              const realIdx = chain.nodes.length - 1 - idx
              const isOpen = expanded.has(realIdx)
              const summary = nodeSummary(n)
              return (
                <div
                  key={n.id}
                  className="rounded border border-white/5 bg-white/[0.02] overflow-hidden"
                >
                  <button
                    onClick={() => toggle(realIdx)}
                    className="w-full text-left px-2 py-1.5 flex items-center gap-2 hover:bg-white/5 transition-colors"
                  >
                    {isOpen ? (
                      <ChevronDown className="h-3 w-3 text-ink-faint shrink-0" />
                    ) : (
                      <ChevronRight className="h-3 w-3 text-ink-faint shrink-0" />
                    )}
                    <NodeIcon type={n.event} />
                    <span className="text-xs font-mono text-ink shrink-0">
                      {summary.title}
                    </span>
                    <span className="text-[10px] text-ink-faint font-mono truncate flex-1">
                      {summary.subtitle}
                    </span>
                    <span className="text-[10px] text-ink-faint font-mono shrink-0">
                      {fmtTime(n.at)}
                    </span>
                  </button>
                  {isOpen && (
                    <div className="px-3 py-2 border-t border-white/5 bg-land-1/40 text-[11px] font-mono">
                      {n.toolName && (
                        <div className="mb-1">
                          <span className="text-ink-faint">tool: </span>
                          <span className="text-tribe">{n.toolName}</span>
                        </div>
                      )}
                      {n.toolInput && (
                        <details open className="mb-1">
                          <summary className="text-ink-faint cursor-pointer">input</summary>
                          <pre className="mt-1 text-ink-dim whitespace-pre-wrap break-all">
                            {formatJSON(n.toolInput)}
                          </pre>
                        </details>
                      )}
                      {n.toolResponse && (
                        <details open>
                          <summary className="text-ink-faint cursor-pointer">response</summary>
                          <pre className="mt-1 text-ink-dim whitespace-pre-wrap break-all">
                            {truncate(formatJSON(n.toolResponse), 1500)}
                          </pre>
                        </details>
                      )}
                      {n.message && !n.toolInput && (
                        <div className="text-ink-dim whitespace-pre-wrap break-all">
                          {n.message}
                        </div>
                      )}
                      {n.userPrompt && (
                        <div className="text-ink-dim whitespace-pre-wrap break-all">
                          <span className="text-ink-faint">user_prompt: </span>
                          {n.userPrompt}
                        </div>
                      )}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface NodeSummary {
  title: string
  subtitle: string
}

function nodeSummary(n: {event: string; toolName?: string; message?: string; toolInput?: Record<string, unknown>; toolResponse?: Record<string, unknown>}): NodeSummary {
  switch (n.event) {
    case 'PreToolUse':
      return {title: n.toolName ?? 'tool', subtitle: toolInputSummary(n.toolInput)}
    case 'PostToolUse':
      return {title: n.toolName ? `${n.toolName} ✓` : 'result', subtitle: toolResponseSummary(n.toolResponse)}
    case 'Notification':
      return {title: 'notify', subtitle: n.message ?? ''}
    case 'Stop':
    case 'SubagentStop':
      return {title: 'stop', subtitle: n.message ?? ''}
    case 'UserPromptSubmit':
      return {title: 'user', subtitle: n.message ?? ''}
    case 'PreCompact':
      return {title: 'compact', subtitle: n.message ?? ''}
    default:
      return {title: n.event, subtitle: ''}
  }
}

function toolInputSummary(input: Record<string, unknown> | undefined): string {
  if (!input) return ''
  // 优先显示常见字段
  const s = firstString(input, ['command', 'file_path', 'path', 'filePath', 'url', 'pattern'])
  if (s) {
    return truncate(s, 80)
  }
  return truncate(formatJSON(input), 80)
}

function toolResponseSummary(resp: Record<string, unknown> | undefined): string {
  if (!resp) return ''
  const s = firstString(resp, ['output', 'content', 'text'])
  if (s) {
    return truncate(s, 80)
  }
  return truncate(formatJSON(resp), 80)
}

function firstString(obj: Record<string, unknown>, keys: string[]): string {
  for (const k of keys) {
    if (typeof obj[k] === 'string' && (obj[k] as string).length > 0) {
      return obj[k] as string
    }
  }
  return ''
}

function NodeIcon({type}: {type: string}) {
  switch (type) {
    case 'PreToolUse':
      return <Wrench className="h-3 w-3 text-forge-amber shrink-0" />
    case 'PostToolUse':
      return <FileText className="h-3 w-3 text-forge-green shrink-0" />
    case 'Notification':
      return <Bell className="h-3 w-3 text-forge-red shrink-0" />
    case 'Stop':
    case 'SubagentStop':
      return <StopCircle className="h-3 w-3 text-ink-faint shrink-0" />
    case 'UserPromptSubmit':
      return <User className="h-3 w-3 text-forge-green shrink-0" />
    case 'PreCompact':
      return <Zap className="h-3 w-3 text-forge-amber shrink-0" />
    default:
      return <AlertTriangle className="h-3 w-3 text-ink-faint shrink-0" />
  }
}

function fmtTime(ts: number): string {
  return new Date(ts).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'})
}

function formatJSON(v: unknown): string {
  try {
    return JSON.stringify(v, null, 2)
  } catch {
    return String(v)
  }
}

function truncate(s: string, n: number): string {
  if (s.length <= n) return s
  return s.slice(0, n) + '…'
}