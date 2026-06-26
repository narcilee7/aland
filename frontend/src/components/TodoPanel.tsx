// Todo 面板——Claude Code 当前 session 的 TodoWrite 快照。
//
// 状态：
//   - pending       ○ 灰色空心
//   - in_progress   ◐ 琥珀脉冲
//   - completed     ✓ 绿色勾
//
// 显示规则：按 status 分组（in_progress 在最上，pending 居中，completed 折叠到底）。

import {useEffect, useState} from 'react'
import {Card, CardContent, CardHeader, CardTitle, Badge} from './ui'
import {listTodos} from '../api/wails'
import type {Todo} from '../api/wails'
import {Check, Circle, CircleDashed, ListChecks, RotateCw} from 'lucide-react'

interface TodoPanelProps {
  tribeId: string
  sessionId: string | null
}

export function TodoPanel({tribeId, sessionId}: TodoPanelProps) {
  const [todos, setTodos] = useState<Todo[]>([])
  const [loading, setLoading] = useState(false)

  const refresh = async () => {
    if (!sessionId) return
    setLoading(true)
    try {
      const t = await listTodos(tribeId, sessionId)
      setTodos(t ?? [])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
    // sessionId 变化时刷新；其他情况不刷新
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [sessionId])

  const grouped = groupTodos(todos)
  const total = todos.length
  const done = todos.filter(t => t.status === 'completed').length

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <ListChecks className="h-4 w-4" />
          Todos
          {total > 0 && (
            <span className="text-ink-faint normal-case tracking-wider text-[10px]">
              {done}/{total} done
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
          <Empty text="选择一个 session 查看 todos" />
        ) : todos.length === 0 ? (
          <Empty text="暂无 todo（Claude 还没用 TodoWrite）" />
        ) : (
          <div className="space-y-4">
            {grouped.inProgress.length > 0 && (
              <Section title="In Progress" tone="amber">
                {grouped.inProgress.map((t, i) => (
                  <TodoRow key={`ip-${i}`} todo={t} />
                ))}
              </Section>
            )}
            {grouped.pending.length > 0 && (
              <Section title="Pending" tone="dim">
                {grouped.pending.map((t, i) => (
                  <TodoRow key={`p-${i}`} todo={t} />
                ))}
              </Section>
            )}
            {grouped.completed.length > 0 && (
              <Section title="Completed" tone="green">
                {grouped.completed.map((t, i) => (
                  <TodoRow key={`c-${i}`} todo={t} />
                ))}
              </Section>
            )}
          </div>
        )}
      </CardContent>
    </Card>
  )
}

function groupTodos(todos: Todo[]) {
  const out = {inProgress: [] as Todo[], pending: [] as Todo[], completed: [] as Todo[]}
  for (const t of todos) {
    switch (t.status) {
      case 'in_progress':
        out.inProgress.push(t)
        break
      case 'pending':
        out.pending.push(t)
        break
      case 'completed':
        out.completed.push(t)
        break
    }
  }
  return out
}

function Section({title, tone, children}: {title: string; tone: 'amber' | 'dim' | 'green'; children: React.ReactNode}) {
  const cls =
    tone === 'amber'
      ? 'text-forge-amber'
      : tone === 'green'
        ? 'text-forge-green'
        : 'text-ink-faint'
  return (
    <div>
      <div className={`text-[10px] font-mono uppercase tracking-wider mb-1.5 ${cls}`}>
        {title}
      </div>
      <div className="space-y-1">{children}</div>
    </div>
  )
}

function TodoRow({todo}: {todo: Todo}) {
  const isDone = todo.status === 'completed'
  const isIP = todo.status === 'in_progress'
  return (
    <div className="flex items-start gap-2 px-2 py-1.5 rounded bg-white/[0.02] border border-white/5">
      {isDone ? (
        <Check className="h-3 w-3 text-forge-green shrink-0 mt-0.5" />
      ) : isIP ? (
        <CircleDashed className="h-3 w-3 text-forge-amber shrink-0 mt-0.5 animate-pulse" />
      ) : (
        <Circle className="h-3 w-3 text-ink-faint shrink-0 mt-0.5" />
      )}
      <div className="flex-1 min-w-0">
        <div
          className={`text-xs ${
            isDone ? 'text-ink-faint line-through' : isIP ? 'text-ink' : 'text-ink-dim'
          }`}
        >
          {todo.content}
        </div>
        {isIP && todo.activeForm && (
          <div className="text-[10px] text-forge-amber font-mono mt-0.5">
            {todo.activeForm}…
          </div>
        )}
      </div>
    </div>
  )
}

function Empty({text}: {text: string}) {
  return (
    <div className="text-xs text-ink-faint font-mono py-6 text-center">{text}</div>
  )
}