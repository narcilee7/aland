// Permission 规则面板——可视化 + 开关 ~/.claude/settings.json 的 permissions。
//
// 三类规则：
//   - allow: 自动允许（绿色）
//   - ask:   需要用户确认（琥珀）
//   - deny:  自动拒绝（红色）
//
// 每条规则是一个 toggle——开启则加入该类，关闭则移除。
// 跨类冲突时（如某规则同时在 allow 和 deny 里）按"显示当前状态 + 单独 toggle"处理。
//
// 数据流：getPermissions → togglePermission → 刷新。

import {useEffect, useState} from 'react'
import {Card, CardContent, CardHeader, CardTitle, Badge} from './ui'
import {getPermissions, togglePermission} from '../api/wails'
import type {Permissions} from '../api/wails'
import {Shield, ShieldCheck, ShieldAlert, RotateCw} from 'lucide-react'

type Category = 'allow' | 'deny' | 'ask'

const CATEGORY_META: Record<Category, {label: string; color: string; Icon: typeof Shield}> = {
  allow: {label: 'Allow', color: 'text-forge-green', Icon: ShieldCheck},
  deny: {label: 'Deny', color: 'text-forge-red', Icon: ShieldAlert},
  ask: {label: 'Ask', color: 'text-forge-amber', Icon: Shield},
}

export function PermissionsPanel() {
  const [perms, setPerms] = useState<Permissions>({allow: [], deny: [], ask: []})
  const [loading, setLoading] = useState(true)

  const refresh = async () => {
    setLoading(true)
    try {
      const p = await getPermissions()
      setPerms(p)
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    refresh()
  }, [])

  const toggle = async (category: Category, rule: string) => {
    const updated = await togglePermission(category, rule)
    if (updated) setPerms(updated)
  }

  // 收集所有出现过的规则（union）
  const allRules = Array.from(
    new Set([...perms.allow, ...perms.deny, ...perms.ask]),
  ).sort()

  return (
    <Card className="flex flex-col min-h-0">
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Shield className="h-4 w-4" />
          Permissions
          <span className="text-ink-faint normal-case tracking-wider text-[10px]">
            ~/.claude/settings.json
          </span>
          <button
            onClick={refresh}
            className="ml-auto text-ink-faint hover:text-ink-dim"
            title="Refresh"
          >
            <RotateCw className={`h-3 w-3 ${loading ? 'animate-spin' : ''}`} />
          </button>
        </CardTitle>
      </CardHeader>
      <CardContent className="flex-1 overflow-auto min-h-0 space-y-4">
        {allRules.length === 0 ? (
          <div className="text-xs text-ink-faint font-mono py-6 text-center">
            暂无 Permission 规则
            <br />
            <span className="text-[10px]">
              （在 Claude Code 里用 /allowed-tools / /permissions 管理）
            </span>
          </div>
        ) : (
          <div className="space-y-2">
            {allRules.map(rule => (
              <RuleRow key={rule} rule={rule} perms={perms} onToggle={toggle} />
            ))}
          </div>
        )}

        {/* 类别计数 */}
        <div className="grid grid-cols-3 gap-2 pt-3 border-t border-white/5">
          {(Object.keys(CATEGORY_META) as Category[]).map(cat => {
            const meta = CATEGORY_META[cat]
            const Icon = meta.Icon
            const count = perms[cat].length
            return (
              <div
                key={cat}
                className="rounded border border-white/5 bg-white/[0.02] px-2 py-1.5 text-center"
              >
                <Icon className={`inline h-3 w-3 ${meta.color} mb-1`} />
                <div className={`text-sm font-mono ${meta.color}`}>{count}</div>
                <div className="text-[9px] font-mono text-ink-faint uppercase tracking-wider">
                  {meta.label}
                </div>
              </div>
            )
          })}
        </div>
      </CardContent>
    </Card>
  )
}

function RuleRow({
  rule,
  perms,
  onToggle,
}: {
  rule: string
  perms: Permissions
  onToggle: (cat: Category, rule: string) => void
}) {
  const inAllow = perms.allow.includes(rule)
  const inDeny = perms.deny.includes(rule)
  const inAsk = perms.ask.includes(rule)

  return (
    <div className="rounded border border-white/5 bg-white/[0.02] p-2.5">
      <div className="text-sm font-mono text-ink break-all mb-2">{rule}</div>
      <div className="flex gap-1.5">
        {(['allow', 'ask', 'deny'] as Category[]).map(cat => {
          const meta = CATEGORY_META[cat]
          const Icon = meta.Icon
          const active = cat === 'allow' ? inAllow : cat === 'ask' ? inAsk : inDeny
          return (
            <button
              key={cat}
              onClick={() => onToggle(cat, rule)}
              className={`flex-1 flex items-center justify-center gap-1 px-2 py-1 rounded text-[10px] font-mono uppercase tracking-wider transition-colors ${
                active
                  ? `${meta.color} bg-white/10 border border-current`
                  : 'text-ink-faint hover:text-ink-dim border border-white/5'
              }`}
            >
              <Icon className="h-3 w-3" />
              {meta.label}
              {active && ' ✓'}
            </button>
          )
        })}
      </div>
    </div>
  )
}