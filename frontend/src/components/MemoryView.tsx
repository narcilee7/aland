// CLAUDE.md 可视化编辑器。
//
// 布局：左 = 文件选择器（project / user global）+ 章节树；右 = 章节内容
//      （只读 markdown 渲染 / 文本编辑两种模式切换）
//
// 数据流：
//   1. 组件挂载 → findMemories(cwd) 拿到可用文件列表
//   2. 选中一个文件 → readMemory(path) 拿 MemoryDoc
//   3. 进入某章节 → 渲染 content；按编辑按钮切到 textarea
//   4. 改完按 Save → saveMemory(path, body, frontmatter) → 备份在 Go 侧做
//
// cwd：从 tribe.vital.cwd 拿（如果有的话）；fallback 用当前浏览器 cwd（通常为空）。

import {useEffect, useMemo, useState} from 'react'
import {Card, CardContent, CardHeader, CardTitle, Button, Badge} from './ui'
import {findMemories, readMemory, saveMemory} from '../api/wails'
import type {MemoryDoc, MemorySection, MemorySource} from '../api/wails'
import {Brain, FileText, FolderOpen, Edit3, Eye, Save, RotateCw, ChevronDown, ChevronRight, Check} from 'lucide-react'
import {logger} from '../lib/logger'

interface MemoryViewProps {
  tribeId: string
  cwd?: string
}

export function MemoryView({tribeId, cwd = ''}: MemoryViewProps) {
  const [sources, setSources] = useState<MemorySource[]>([])
  const [activePath, setActivePath] = useState<string | null>(null)
  const [doc, setDoc] = useState<MemoryDoc | null>(null)
  const [loading, setLoading] = useState(false)

  // 编辑状态：每个章节独立的 draft + dirty
  const [editingOrder, setEditingOrder] = useState<number | null>(null)
  const [drafts, setDrafts] = useState<Record<number, string>>({})
  const [dirty, setDirty] = useState<Record<number, boolean>>({})

  const refresh = async () => {
    setLoading(true)
    try {
      const xs = await findMemories(tribeId, cwd)
      setSources(xs)
      if (xs.length > 0 && !activePath) {
        setActivePath(xs[0].path)
      }
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tribeId, cwd])

  useEffect(() => {
    if (!activePath) {
      setDoc(null)
      return
    }
    setLoading(true)
    readMemory(tribeId, activePath).then(d => {
      setDoc(d)
      setDrafts({})
      setDirty({})
      setEditingOrder(null)
      setLoading(false)
    })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [activePath])

  const startEdit = (s: MemorySection) => {
    setEditingOrder(s.order)
    setDrafts(prev => ({...prev, [s.order]: s.content}))
  }

  const cancelEdit = (s: MemorySection) => {
    setEditingOrder(null)
    setDrafts(prev => {
      const {[s.order]: _, ...rest} = prev
      return rest
    })
    setDirty(prev => {
      const {[s.order]: _, ...rest} = prev
      return rest
    })
  }

  const updateDraft = (order: number, value: string) => {
    setDrafts(prev => ({...prev, [order]: value}))
    setDirty(prev => ({...prev, [order]: true}))
  }

  const saveSection = async (s: MemorySection) => {
    if (!doc) return
    // 重建 body：把这一节的内容替换，其他保持
    const newBody = doc.sections
      .map(sec => (sec.order === s.order ? drafts[s.order] : sec.content))
      .join('\n\n')
    const ok = await saveMemory(tribeId, doc.source.path, newBody, doc.frontmatter)
    if (ok) {
      // 重新读取刷新
      const fresh = await readMemory(tribeId, doc.source.path)
      setDoc(fresh)
      setEditingOrder(null)
      setDirty({})
      setDrafts({})
    } else {
      logger.error('save memory failed')
    }
  }

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Brain className="h-4 w-4" />
          CLAUDE.md Memory
          <button
            onClick={refresh}
            disabled={loading}
            className="ml-auto text-ink-faint hover:text-ink-dim disabled:opacity-30"
            title="Refresh"
          >
            <RotateCw className={`h-3 w-3 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-auto min-h-0">
        <div className="grid grid-cols-[200px_1fr] gap-3 h-full">
          {/* 左：文件 + 章节树 */}
          <div className="space-y-3 border-r border-white/5 pr-3 overflow-auto">
            <div>
              <div className="text-[10px] font-mono uppercase tracking-wider text-ink-faint mb-2">
                Files
              </div>
              {sources.length === 0 ? (
                <div className="text-xs text-ink-faint font-mono">
                  暂无 CLAUDE.md
                </div>
              ) : (
                <div className="space-y-1">
                  {sources.map(s => (
                    <button
                      key={s.path}
                      onClick={() => setActivePath(s.path)}
                      className={`w-full text-left px-2 py-1.5 rounded text-[10px] font-mono truncate transition-colors ${
                        s.path === activePath
                          ? 'bg-tribe/20 text-tribe'
                          : 'hover:bg-white/5 text-ink-dim'
                      }`}
                      title={s.path}
                    >
                      <Badge status="running" />
                      <span className="ml-1.5 uppercase tracking-wider">{s.scope}</span>
                      <div className="truncate mt-0.5">{basename(s.path)}</div>
                    </button>
                  ))}
                </div>
              )}
            </div>

            {doc && doc.sections.length > 0 && (
              <div>
                <div className="text-[10px] font-mono uppercase tracking-wider text-ink-faint mb-2">
                  Sections ({doc.sections.length})
                </div>
                <SectionTree
                  sections={doc.sections}
                  dirty={dirty}
                  active={editingOrder}
                  onPick={s => startEdit(s)}
                />
              </div>
            )}

            {doc && doc.imports.length > 0 && (
              <div>
                <div className="text-[10px] font-mono uppercase tracking-wider text-ink-faint mb-2">
                  Imports
                </div>
                <div className="space-y-0.5">
                  {doc.imports.map((imp, i) => (
                    <div
                      key={i}
                      className="text-[10px] font-mono text-ink-faint truncate"
                      title={`${imp.path}:${imp.line}`}
                    >
                      @{imp.path}
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

          {/* 右：内容 */}
          <div className="overflow-auto">
            {!doc ? (
              <div className="text-xs text-ink-faint font-mono py-6 text-center">
                {loading ? 'loading...' : '选择左侧文件查看内容'}
              </div>
            ) : doc.sections.length === 0 ? (
              <EmptyDoc doc={doc} />
            ) : (
              <div className="space-y-4">
                {doc.frontmatter && (
                  <details className="rounded border border-white/5 bg-white/[0.02] p-2">
                    <summary className="text-[10px] font-mono uppercase tracking-wider text-ink-faint cursor-pointer">
                      Frontmatter
                    </summary>
                    <pre className="mt-2 text-[11px] font-mono text-ink-dim whitespace-pre-wrap break-all">
                      {doc.frontmatter}
                    </pre>
                  </details>
                )}
                {doc.sections.map(s => (
                  <SectionView
                    key={s.order}
                    section={s}
                    isEditing={editingOrder === s.order}
                    isDirty={!!dirty[s.order]}
                    draft={drafts[s.order] ?? ''}
                    onStartEdit={() => startEdit(s)}
                    onCancel={() => cancelEdit(s)}
                    onSave={() => saveSection(s)}
                    onChange={v => updateDraft(s.order, v)}
                  />
                ))}
              </div>
            )}
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

function SectionTree({
  sections,
  dirty,
  active,
  onPick,
}: {
  sections: MemorySection[]
  dirty: Record<number, boolean>
  active: number | null
  onPick: (s: MemorySection) => void
}) {
  const [open, setOpen] = useState(true)
  return (
    <div className="space-y-0.5">
      <button
        onClick={() => setOpen(o => !o)}
        className="text-ink-faint text-[10px] font-mono"
      >
        {open ? <ChevronDown className="inline h-2.5 w-2.5" /> : <ChevronRight className="inline h-2.5 w-2.5" />}
      </button>
      {open && (
        <div className="space-y-0.5">
          {sections.map(s => (
            <button
              key={s.order}
              onClick={() => onPick(s)}
              className={`w-full text-left px-1.5 py-1 rounded text-[10px] font-mono truncate transition-colors ${
                active === s.order
                  ? 'bg-forge-green/20 text-forge-green'
                  : dirty[s.order]
                    ? 'text-forge-amber hover:bg-white/5'
                    : 'text-ink-dim hover:bg-white/5'
              }`}
              title={s.title}
              style={{paddingLeft: `${(s.level - 1) * 8 + 6}px`}}
            >
              {dirty[s.order] ? '● ' : ''}
              {'#'.repeat(s.level)} {s.title}
            </button>
          ))}
        </div>
      )}
    </div>
  )
}

function SectionView({
  section,
  isEditing,
  isDirty,
  draft,
  onStartEdit,
  onCancel,
  onSave,
  onChange,
}: {
  section: MemorySection
  isEditing: boolean
  isDirty: boolean
  draft: string
  onStartEdit: () => void
  onCancel: () => void
  onSave: () => void
  onChange: (v: string) => void
}) {
  return (
    <div className="rounded border border-white/5 bg-white/[0.02] p-3">
      <div className="flex items-center gap-2 mb-2">
        <h3 className="text-sm font-mono text-ink m-0">
          {'#'.repeat(section.level)} {section.title}
        </h3>
        <Badge status={isDirty ? 'busy' : 'idle'} />
        <div className="ml-auto flex items-center gap-1">
          {!isEditing && (
            <Button size="sm" variant="ghost" onClick={onStartEdit}>
              <Edit3 className="h-3 w-3" />
              Edit
            </Button>
          )}
          {isEditing && (
            <>
              <Button size="sm" variant="ghost" onClick={onCancel}>
                Cancel
              </Button>
              <Button size="sm" variant="default" onClick={onSave} disabled={!isDirty}>
                <Save className="h-3 w-3" />
                Save
              </Button>
            </>
          )}
        </div>
      </div>
      {isEditing ? (
        <textarea
          value={draft}
          onChange={e => onChange(e.target.value)}
          className="w-full min-h-[120px] bg-land-1/60 border border-white/10 rounded p-2 text-xs font-mono text-ink resize-y focus:outline-none focus:border-tribe"
        />
      ) : (
        <pre className="text-xs text-ink-dim font-mono whitespace-pre-wrap break-all m-0">
          {section.content || <span className="text-ink-faint italic">（空）</span>}
        </pre>
      )}
    </div>
  )
}

function EmptyDoc({doc}: {doc: MemoryDoc}) {
  return (
    <div className="text-xs text-ink-faint font-mono py-6 text-center">
      <FileText className="inline h-3 w-3 mr-1" />
      没有 # 章节
      <br />
      <span className="text-[10px]">
        直接编辑文件：<code className="text-ink-dim">{doc.source.path}</code>
      </span>
    </div>
  )
}

function basename(p: string): string {
  const i = p.lastIndexOf('/')
  return i >= 0 ? p.slice(i + 1) : p
}