// Subagent 树——Claude Code Task 工具派生的子 agent 层级。
//
// 节点状态：
//   - running    蓝色脉动
//   - completed  绿色 ✓
//   - error      红色 ⚠
//   - unknown    灰色
//
// 缩进表示嵌套深度。子 agent 派生的 agent 再递归展开。

import {useEffect, useState} from 'react'
import {Card, CardContent, CardHeader, CardTitle} from './ui'
import {getSubagentTree} from '../api/wails'
import type {AgentNode} from '../api/wails'
import {
  Bot,
  Check,
  CircleDashed,
  AlertTriangle,
  HelpCircle,
  GitBranch,
  ChevronDown,
  ChevronRight,
  RotateCw,
  MessageSquare,
  Wrench,
} from 'lucide-react'

interface SubagentTreeProps {
  tribeId: string
  sessionId: string | null
}

export function SubagentTree({tribeId, sessionId}: SubagentTreeProps) {
  const [tree, setTree] = useState<AgentNode | null>(null)
  const [loading, setLoading] = useState(false)

  const refresh = async () => {
    if (!sessionId) return
    setLoading(true)
    try {
      const t = await getSubagentTree(tribeId, sessionId)
      setTree(t)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  const hasChildren = tree && tree.children && tree.children.length > 0

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <GitBranch className="h-4 w-4" />
          Subagent Tree
          {hasChildren && (
            <span className="text-ink-faint normal-case tracking-wider text-[10px]">
              {countNodes(tree!)}
            </span>
          )}
          <button
            onClick={refresh}
            disabled={!sessionId}
            className="ml-auto text-ink-faint hover:text-ink-dim disabled:opacity-30"
            title="Refresh"
          >
            <RotateCw className={`h-3 w-3 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-auto min-h-0">
        {!sessionId ? (
          <div className="text-xs text-ink-faint font-mono py-6 text-center">
            选择一个 session 查看 subagent 树
          </div>
        ) : !hasChildren ? (
          <div className="text-xs text-ink-faint font-mono py-6 text-center">
            <Bot className="inline h-3 w-3 mr-1" />
            暂无 subagent（Claude 还没用 Task 工具）
          </div>
        ) : (
          <div className="space-y-1">
            {tree!.children!.map((child, i) => (
              <AgentNodeView key={child.id || i} node={child} depth={0} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function countNodes(n: AgentNode): number {
  let c = 1
  for (const ch of n.children ?? []) {
    c += countNodes(ch)
  }
  return c
}

function AgentNodeView({node, depth}: {node: AgentNode; depth: number}) {
  const [open, setOpen] = useState(true)
  const hasChildren = node.children && node.children.length > 0
  const meta = STATUS_META[node.status] ?? STATUS_META.unknown
  const Icon = meta.Icon

  return (
    <div>
      <div
        className="flex items-start gap-2 px-2 py-1.5 rounded bg-white/[0.02] border border-white/5 hover:bg-white/[0.04] transition-colors"
        style={{marginLeft: depth * 16}}
      >
        {hasChildren ? (
          <button
            onClick={() => setOpen(o => !o)}
            className="text-ink-faint shrink-0 mt-0.5"
          >
            {open ? <ChevronDown className="h-3 w-3" /> : <ChevronRight className="h-3 w-3" />}
          </button>
        ) : (
          <span className="w-3 shrink-0" />
        )}
        <Icon className={`h-3 w-3 shrink-0 mt-0.5 ${meta.color}`} />
        <div className="flex-1 min-w-0">
          <div className="flex items-center gap-1.5">
            <span className="text-xs font-mono text-ink truncate">
              {node.description || node.type || node.id}
            </span>
            {node.type && (
              <span className="text-[10px] text-ink-faint font-mono shrink-0">
                [{node.type}]
              </span>
            )}
          </div>
          {node.startedAt > 0 && (
            <div className="text-[10px] text-ink-faint font-mono mt-0.5 flex items-center gap-2">
              <span className="inline-flex items-center gap-0.5">
                <MessageSquare className="h-2.5 w-2.5" />
                {node.messageCount}
              </span>
              <span className="inline-flex items-center gap-0.5">
                <Wrench className="h-2.5 w-2.5" />
                {node.toolUseCount}
              </span>
              {node.startedAt > 0 && (
                <span>· {fmtTime(node.startedAt)}</span>
              )}
              {node.endedAt > 0 && (
                <span>– {fmtTime(node.endedAt)}</span>
              )}
            </div>
          )}
        </div>
      </div>
      {open && hasChildren && (
        <div className="mt-1 space-y-1">
          {node.children.map((c, i) => (
            <AgentNodeView key={c.id || i} node={c} depth={depth + 1} />
          ))}
        </div>
      )}
    </div>
  )
}

const STATUS_META: Record<string, {color: string; Icon: typeof Bot}> = {
  running: {color: 'text-blue-400', Icon: CircleDashed},
  completed: {color: 'text-forge-green', Icon: Check},
  error: {color: 'text-forge-red', Icon: AlertTriangle},
  unknown: {color: 'text-ink-faint', Icon: HelpCircle},
}

function fmtTime(ts: number): string {
  return new Date(ts).toLocaleTimeString([], {hour: '2-digit', minute: '2-digit', second: '2-digit'})
}