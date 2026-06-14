import { useMemo, useState } from 'react'
import type { ReactNode } from 'react'
import {
  Search,
  RefreshCcw,
  Loader2,
  Users,
  ShieldCheck,
  ListTree,
  ScrollText,
  SlidersHorizontal,
  Lock,
  Wifi,
  CircleDot,
  AlertTriangle,
} from 'lucide-react'

import { NeonButton } from '@/components/ui/NeonButton'
import { StatusBadge } from '@/components/ui/StatusBadge'
import { formatTime } from '@/lib/format'
import { useUsers, useRoles, useSysConfigsByCategory } from '@core/hooks/api/useSystem'
import { useMenuTree } from '@core/hooks/api/useMenus'
import { useOperationLogs } from '@core/hooks/api/useLogs'
import type {
  User,
  UserStatus,
  Role,
  SysConfigItem,
  OperationResult,
} from '@core/types/system'
import type { Menu } from '@core/types/menu'

type Tab = 'users' | 'roles' | 'menus' | 'audit' | 'config'

const TABS: { key: Tab; label: string; sub: string; icon: ReactNode }[] = [
  { key: 'users', label: '人员清单', sub: 'ACCOUNTS', icon: <Users className="size-3.5" /> },
  { key: 'roles', label: '角色权限', sub: 'RBAC ROLES', icon: <ShieldCheck className="size-3.5" /> },
  { key: 'menus', label: '菜单矩阵', sub: 'MENU TREE', icon: <ListTree className="size-3.5" /> },
  { key: 'audit', label: '审计回放', sub: 'AUDIT LOG', icon: <ScrollText className="size-3.5" /> },
  { key: 'config', label: '系统参数', sub: 'SYS CONFIG', icon: <SlidersHorizontal className="size-3.5" /> },
]

export function SystemPage() {
  const [tab, setTab] = useState<Tab>('users')

  return (
    <div className="warp-in flex h-full w-full flex-col gap-4">
      {/* 标题带 + tab 切换 */}
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div className="flex items-end gap-4">
          <div className="font-display text-4xl font-bold leading-none text-cyan-300/85 text-glow">
            F06
          </div>
          <div>
            <div className="font-display text-lg leading-tight text-cyan-100">
              SYSTEM · 内核管理
            </div>
            <div className="font-mono text-[11px] uppercase tracking-[0.18em] text-cyan-300/55">
              USER · ROLE · MENU · AUDIT · CONFIG
            </div>
          </div>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          {TABS.map((tdef) => (
            <button
              key={tdef.key}
              type="button"
              onClick={() => setTab(tdef.key)}
              className={`chip flex items-center gap-1.5 transition-all ${
                tab === tdef.key
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/55 hover:text-cyan-200'
              }`}
            >
              {tdef.icon}
              <span>{tdef.label}</span>
              <span className="hidden font-mono text-[9px] opacity-60 md:inline">
                {tdef.sub}
              </span>
            </button>
          ))}
        </div>
      </div>

      <div className="flex-1 min-h-0">
        {tab === 'users' && <UsersTab />}
        {tab === 'roles' && <RolesTab />}
        {tab === 'menus' && <MenusTab />}
        {tab === 'audit' && <AuditTab />}
        {tab === 'config' && <ConfigTab />}
      </div>
    </div>
  )
}

/* ============================ 通用三态包裹 ============================ */

function StateBlock({
  isLoading,
  isError,
  error,
  isEmpty,
  emptyLabel,
  children,
}: {
  isLoading: boolean
  isError: boolean
  error?: unknown
  isEmpty: boolean
  emptyLabel: string
  children: ReactNode
}) {
  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 py-16 text-cyan-300/60">
        <Loader2 className="size-4 animate-spin" />
        <span className="font-mono text-xs uppercase tracking-[0.2em]">SYNCING…</span>
      </div>
    )
  }
  if (isError) {
    return (
      <div className="border border-rose-500/40 bg-rose-500/5 px-4 py-6 font-mono text-sm text-rose-300">
        FAILURE · {error instanceof Error ? error.message : '未知错误'}
      </div>
    )
  }
  if (isEmpty) {
    return (
      <div className="flex flex-col items-center justify-center gap-3 py-16 text-cyan-300/45">
        <CircleDot className="size-9 opacity-50" />
        <div className="font-mono text-xs uppercase tracking-[0.2em]">{emptyLabel}</div>
      </div>
    )
  }
  return <>{children}</>
}

/* ============================ 统计卡 ============================ */

function MiniStat({
  label,
  value,
  color,
  icon,
}: {
  label: string
  value: ReactNode
  color: string
  icon?: ReactNode
}) {
  return (
    <div
      className="glass relative overflow-hidden rounded-sm border-l-2 px-4 py-3"
      style={{ borderLeftColor: color }}
    >
      <div className="flex items-center gap-1.5 font-mono text-[10px] uppercase tracking-[0.2em] text-cyan-300/65">
        {icon ? <span style={{ color }}>{icon}</span> : null}
        {label}
      </div>
      <div
        className="font-display text-3xl font-bold leading-tight"
        style={{ color, textShadow: `0 0 8px ${color}` }}
      >
        {value}
      </div>
    </div>
  )
}

/* ============================ USERS ============================ */

const USER_STATUS_MAP: Record<UserStatus, { label: string; badge: string }> = {
  active: { label: '启用', badge: 'active' },
  disabled: { label: '禁用', badge: 'off' },
  inactive: { label: '未激活', badge: 'inactive' },
  locked: { label: '锁定', badge: 'critical' },
}

function UsersTab() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { userName: keyword.trim() } : {}),
    }),
    [page, keyword]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useUsers(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  // 本页内统计（后端未暴露全量聚合，用当前页面快照）
  const onlineCount = useMemo(
    () => rows.filter((u) => u.status === 'active').length,
    [rows]
  )
  const lockedCount = useMemo(
    () => rows.filter((u) => u.status === 'locked' || u.status === 'disabled').length,
    [rows]
  )
  const builtInCount = useMemo(
    () => rows.filter((u) => u.source === 'builtIn').length,
    [rows]
  )

  return (
    <TabFrame
      isFetching={isFetching}
      toolbar={
        <KeywordToolbar
          placeholder="账号 / 用户名"
          value={keyword}
          onChange={(v) => {
            setKeyword(v)
            setPage(1)
          }}
          onRefresh={() => refetch()}
        />
      }
      stats={
        <>
          <MiniStat
            label="本页账号"
            value={total.toLocaleString()}
            color="#00f0ff"
            icon={<Users className="size-3.5" />}
          />
          <MiniStat
            label="启用 · ACTIVE"
            value={onlineCount}
            color="#00ff88"
            icon={<Wifi className="size-3.5" />}
          />
          <MiniStat
            label="禁用/锁定"
            value={lockedCount}
            color="#ff7a1a"
            icon={<Lock className="size-3.5" />}
          />
          <MiniStat
            label="内置 · BUILT-IN"
            value={builtInCount}
            color="#a855f7"
            icon={<ShieldCheck className="size-3.5" />}
          />
        </>
      }
      footer={
        <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyLabel="NO ACCOUNTS · 无账号"
      >
        <div className="space-y-1.5">
          <RowHeader cols="2fr_1.4fr_1fr_1.2fr_1fr_1.4fr">
            <span>用户 · IDENTITY</span>
            <span>账号 · LOGIN</span>
            <span>状态</span>
            <span>角色 · ROLES</span>
            <span>来源</span>
            <span>最后登录</span>
          </RowHeader>
          {rows.map((u: User) => {
            const st = USER_STATUS_MAP[u.status]
            return (
              <div
                key={u.id}
                className="fleet-row grid grid-cols-[2fr_1.4fr_1fr_1.2fr_1fr_1.4fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: '#00f0ff' }}
              >
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {u.displayName || u.username}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {u.email || '—'}
                  </div>
                </div>
                <div className="truncate font-mono text-xs text-cyan-100/85">{u.username}</div>
                <div>
                  <StatusBadge status={st.badge} label={st.label} />
                </div>
                <div className="min-w-0 truncate text-xs text-cyan-100/80">
                  {u.roles && u.roles.length > 0 ? u.roles.join(' · ') : '—'}
                </div>
                <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-cyan-300/65">
                  {u.source === 'builtIn' ? '内置' : u.source === 'LDAP' ? 'LDAP' : '本地'}
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  {u.lastLoginTime ? formatTime(u.lastLoginTime) : '—'}
                </div>
              </div>
            )
          })}
        </div>
      </StateBlock>
    </TabFrame>
  )
}

/* ============================ ROLES ============================ */

function RolesTab() {
  const [page, setPage] = useState(1)
  const pageSize = 20
  const [keyword, setKeyword] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { roleName: keyword.trim() } : {}),
    }),
    [page, keyword]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useRoles(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const builtInCount = useMemo(() => rows.filter((r) => r.builtIn > 0).length, [rows])
  const boundUsers = useMemo(
    () => rows.reduce((acc, r) => acc + (r.userCount ?? 0), 0),
    [rows]
  )
  const noGroupCount = useMemo(
    () =>
      rows.filter(
        (r) => r.builtIn === 0 && (!r.deviceGroupIds || r.deviceGroupIds.length === 0)
      ).length,
    [rows]
  )

  return (
    <TabFrame
      isFetching={isFetching}
      toolbar={
        <KeywordToolbar
          placeholder="角色名 / 编码"
          value={keyword}
          onChange={(v) => {
            setKeyword(v)
            setPage(1)
          }}
          onRefresh={() => refetch()}
        />
      }
      stats={
        <>
          <MiniStat
            label="本页角色"
            value={total.toLocaleString()}
            color="#00f0ff"
            icon={<ShieldCheck className="size-3.5" />}
          />
          <MiniStat label="内置角色" value={builtInCount} color="#a855f7" />
          <MiniStat label="绑定用户合计" value={boundUsers} color="#00ff88" />
          <MiniStat
            label="未绑设备组"
            value={noGroupCount}
            color={noGroupCount > 0 ? '#ff7a1a' : '#525a78'}
            icon={noGroupCount > 0 ? <AlertTriangle className="size-3.5" /> : undefined}
          />
        </>
      }
      footer={
        <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyLabel="NO ROLES · 无角色"
      >
        <div className="space-y-1.5">
          <RowHeader cols="2fr_1.4fr_0.8fr_0.8fr_1fr_1.4fr">
            <span>角色 · ROLE</span>
            <span>编码 · CODE</span>
            <span>用户数</span>
            <span>类型</span>
            <span>设备组</span>
            <span>更新时间</span>
          </RowHeader>
          {rows.map((r: Role) => {
            const noGroup =
              r.builtIn === 0 && (!r.deviceGroupIds || r.deviceGroupIds.length === 0)
            return (
              <div
                key={r.id}
                className="fleet-row grid grid-cols-[2fr_1.4fr_0.8fr_0.8fr_1fr_1.4fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{ ['--row-color' as never]: r.builtIn > 0 ? '#a855f7' : '#00f0ff' }}
              >
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {r.roleName}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {r.description || '—'}
                  </div>
                </div>
                <div className="truncate font-mono text-xs text-cyan-100/85">{r.roleCode}</div>
                <div className="font-display text-sm font-bold text-cyan-200">{r.userCount ?? 0}</div>
                <div>
                  <StatusBadge
                    status={r.builtIn > 0 ? 'inactive' : 'online'}
                    label={r.builtIn > 0 ? '内置' : '自定义'}
                  />
                </div>
                <div className="text-xs">
                  {r.builtIn > 0 ? (
                    <span className="font-mono text-[10px] text-cyan-300/55">全部</span>
                  ) : noGroup ? (
                    <span className="chip text-[#ff7a1a]">未绑定</span>
                  ) : (
                    <span className="font-mono text-[11px] text-cyan-200">
                      {r.deviceGroupIds?.length ?? 0} 组
                    </span>
                  )}
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  {r.updateTime ? formatTime(r.updateTime) : '—'}
                </div>
              </div>
            )
          })}
        </div>
      </StateBlock>
    </TabFrame>
  )
}

/* ============================ MENUS ============================ */

const MENU_TYPE_COLOR: Record<string, string> = {
  directory: '#00f0ff',
  menu: '#00ff88',
  button: '#a855f7',
}

function MenusTab() {
  const { data, isLoading, isError, error, isFetching, refetch } = useMenuTree()
  const tree = data ?? []

  const flat = useMemo(() => flattenMenus(tree), [tree])
  const dirCount = flat.filter((m) => m.type === 'directory').length
  const menuCount = flat.filter((m) => m.type === 'menu').length
  const btnCount = flat.filter((m) => m.type === 'button').length
  const disabledCount = flat.filter((m) => m.status === 'disabled').length

  return (
    <TabFrame
      isFetching={isFetching}
      toolbar={
        <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
          REFRESH
        </NeonButton>
      }
      stats={
        <>
          <MiniStat label="节点总数" value={flat.length} color="#00f0ff" icon={<ListTree className="size-3.5" />} />
          <MiniStat label="目录 · DIR" value={dirCount} color="#00f0ff" />
          <MiniStat label="菜单 · MENU" value={menuCount} color="#00ff88" />
          <MiniStat
            label="按钮 / 禁用"
            value={`${btnCount} / ${disabledCount}`}
            color="#a855f7"
          />
        </>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={tree.length === 0}
        emptyLabel="NO MENUS · 无菜单"
      >
        <div className="space-y-0.5">
          {tree.map((m) => (
            <MenuNode key={m.id} node={m} depth={0} />
          ))}
        </div>
      </StateBlock>
    </TabFrame>
  )
}

function MenuNode({ node, depth }: { node: Menu; depth: number }) {
  const color = MENU_TYPE_COLOR[node.type] ?? '#6b86b6'
  const disabled = node.status === 'disabled'
  return (
    <>
      <div
        className="flex items-center gap-2 border-b border-cyan-500/8 py-1.5 hover:bg-cyan-500/5"
        style={{ paddingLeft: 8 + depth * 18 }}
      >
        <span
          className="size-1.5 shrink-0 rounded-full"
          style={{ background: color, boxShadow: `0 0 6px ${color}` }}
        />
        <span
          className={`font-display text-sm ${disabled ? 'text-cyan-300/35 line-through' : 'text-cyan-100'}`}
        >
          {node.name}
        </span>
        <span className="chip" style={{ color }}>
          {node.type}
        </span>
        {node.routePath ? (
          <span className="truncate font-mono text-[10px] text-cyan-300/45">
            {node.routePath}
          </span>
        ) : null}
        {node.permissionKey ? (
          <span className="ml-auto font-mono text-[10px] text-cyan-300/40">
            {node.permissionKey}
          </span>
        ) : null}
      </div>
      {node.children?.map((c) => (
        <MenuNode key={c.id} node={c} depth={depth + 1} />
      ))}
    </>
  )
}

function flattenMenus(nodes: Menu[]): Menu[] {
  const out: Menu[] = []
  const walk = (list: Menu[]) => {
    for (const n of list) {
      out.push(n)
      if (n.children?.length) walk(n.children)
    }
  }
  walk(nodes)
  return out
}

/* ============================ AUDIT (operation logs) ============================ */

const RESULT_LABEL: Record<OperationResult, string> = {
  success: '成功',
  failure: '失败',
  partial: '部分成功',
}
const RESULT_BADGE: Record<OperationResult, string> = {
  success: 'ok',
  failure: 'critical',
  partial: 'warning',
}

function AuditTab() {
  const [page, setPage] = useState(1)
  const pageSize = 30
  const [keyword, setKeyword] = useState('')
  const [result, setResult] = useState<OperationResult | ''>('')

  const params = useMemo(
    () => ({
      page,
      pageSize,
      ...(keyword.trim() ? { keyword: keyword.trim() } : {}),
      ...(result ? { result } : {}),
    }),
    [page, keyword, result]
  )
  const { data, isLoading, isError, error, isFetching, refetch } = useOperationLogs(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const okCount = useMemo(() => rows.filter((l) => l.result === 'success').length, [rows])
  const failCount = useMemo(() => rows.filter((l) => l.result === 'failure').length, [rows])

  return (
    <TabFrame
      isFetching={isFetching}
      toolbar={
        <>
          <KeywordToolbar
            placeholder="操作人 / 目标 / 内容"
            value={keyword}
            onChange={(v) => {
              setKeyword(v)
              setPage(1)
            }}
          />
          {(['', 'success', 'failure', 'partial'] as const).map((r) => (
            <button
              key={r || 'all'}
              type="button"
              onClick={() => {
                setResult(r)
                setPage(1)
              }}
              className={`chip transition-all ${
                result === r ? 'shadow-[0_0_10px_currentColor]' : 'opacity-60 hover:opacity-100'
              }`}
              style={{
                color: r ? (r === 'failure' ? '#ff2d6f' : r === 'partial' ? '#ffaa00' : '#00ff88') : '#00f0ff',
              }}
            >
              {r ? RESULT_LABEL[r] : 'ALL'}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
      stats={
        <>
          <MiniStat label="本页记录" value={total.toLocaleString()} color="#00f0ff" icon={<ScrollText className="size-3.5" />} />
          <MiniStat label="成功 · OK" value={okCount} color="#00ff88" />
          <MiniStat label="失败 · FAIL" value={failCount} color={failCount > 0 ? '#ff2d6f' : '#525a78'} />
          <MiniStat label="页码" value={`${page}/${totalPages}`} color="#a855f7" />
        </>
      }
      footer={
        <Pager page={page} totalPages={totalPages} total={total} pageSize={pageSize} onPage={setPage} />
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={rows.length === 0}
        emptyLabel="NO AUDIT RECORDS · 无审计记录"
      >
        <div className="space-y-1.5">
          <RowHeader cols="1.5fr_1fr_1fr_2fr_1fr_1.4fr">
            <span>操作人 · IP</span>
            <span>模块</span>
            <span>类型</span>
            <span>目标 / 内容</span>
            <span>结果</span>
            <span>时间</span>
          </RowHeader>
          {rows.map((l) => {
            const badge = RESULT_BADGE[l.result] ?? 'unknown'
            return (
              <div
                key={l.id}
                className="fleet-row grid grid-cols-[1.5fr_1fr_1fr_2fr_1fr_1.4fr] items-center gap-3 rounded-sm px-3 py-2.5"
                style={{
                  ['--row-color' as never]:
                    l.result === 'failure' ? '#ff2d6f' : l.result === 'partial' ? '#ffaa00' : '#00ff88',
                }}
              >
                <div className="min-w-0">
                  <div className="truncate font-display text-sm font-bold text-cyan-100">
                    {l.operator || '—'}
                  </div>
                  <div className="truncate font-mono text-[10px] text-cyan-300/55">
                    {l.clientIp || '—'}
                  </div>
                </div>
                <div className="truncate text-xs text-cyan-100/85">{l.module || '—'}</div>
                <div className="truncate font-mono text-[11px] uppercase text-cyan-300/70">
                  {l.operationType || '—'}
                </div>
                <div className="min-w-0">
                  <div className="truncate text-xs text-cyan-100/85">{l.target || l.content || '—'}</div>
                  {l.message ? (
                    <div className="truncate font-mono text-[10px] text-cyan-300/55">{l.message}</div>
                  ) : null}
                </div>
                <div>
                  <StatusBadge status={badge} label={RESULT_LABEL[l.result] ?? l.result} />
                </div>
                <div className="font-mono text-[11px] text-cyan-300/75">
                  {formatTime(l.operationTime)}
                </div>
              </div>
            )
          })}
        </div>
      </StateBlock>
    </TabFrame>
  )
}

/* ============================ CONFIG (sys_configs KV) ============================ */

const CONFIG_CATEGORIES: { key: string; label: string }[] = [
  { key: 'basic', label: '基础' },
  { key: 'security', label: '安全' },
  { key: 'device', label: '设备' },
  { key: 'notify', label: '通知' },
  { key: 'storage', label: '存储' },
  { key: 'omc', label: 'OMC' },
  { key: 'northbound', label: '北向' },
]

function ConfigTab() {
  const [category, setCategory] = useState<string>('basic')
  const { data, isLoading, isError, error, isFetching, refetch } = useSysConfigsByCategory(category)
  const items = data ?? []

  return (
    <TabFrame
      isFetching={isFetching}
      toolbar={
        <>
          {CONFIG_CATEGORIES.map((c) => (
            <button
              key={c.key}
              type="button"
              onClick={() => setCategory(c.key)}
              className={`chip transition-all ${
                category === c.key
                  ? 'text-cyan-200 shadow-[0_0_10px_currentColor]'
                  : 'text-cyan-300/55 hover:text-cyan-200'
              }`}
            >
              {c.label}
            </button>
          ))}
          <NeonButton icon={<RefreshCcw />} onClick={() => refetch()}>
            REFRESH
          </NeonButton>
        </>
      }
      stats={
        <>
          <MiniStat
            label="当前分类"
            value={CONFIG_CATEGORIES.find((c) => c.key === category)?.label ?? category}
            color="#00f0ff"
            icon={<SlidersHorizontal className="size-3.5" />}
          />
          <MiniStat label="配置项数" value={items.length} color="#00ff88" />
          <MiniStat
            label="公开项"
            value={items.filter((i) => i.isPublic).length}
            color="#a855f7"
          />
          <MiniStat
            label="分类总数"
            value={CONFIG_CATEGORIES.length}
            color="#5b9eff"
          />
        </>
      }
    >
      <StateBlock
        isLoading={isLoading}
        isError={isError}
        error={error}
        isEmpty={items.length === 0}
        emptyLabel="NO CONFIG ITEMS · 该分类无配置"
      >
        <div className="space-y-1.5">
          <RowHeader cols="1.6fr_1.6fr_0.8fr_2fr">
            <span>键 · KEY</span>
            <span>值 · VALUE</span>
            <span>类型</span>
            <span>说明</span>
          </RowHeader>
          {items.map((cfg: SysConfigItem) => (
            <div
              key={cfg.id || `${cfg.category}.${cfg.key}`}
              className="fleet-row grid grid-cols-[1.6fr_1.6fr_0.8fr_2fr] items-center gap-3 rounded-sm px-3 py-2.5"
              style={{ ['--row-color' as never]: '#00f0ff' }}
            >
              <div className="min-w-0">
                <div className="truncate font-mono text-xs text-cyan-100">{cfg.key}</div>
                {cfg.isPublic ? (
                  <span className="chip text-[#00ff88]">PUBLIC</span>
                ) : null}
              </div>
              <div className="min-w-0 truncate font-mono text-xs text-cyan-200">
                {cfg.value || '—'}
              </div>
              <div className="font-mono text-[10px] uppercase tracking-[0.12em] text-cyan-300/65">
                {cfg.valueType || 'string'}
              </div>
              <div className="min-w-0 truncate text-xs text-cyan-100/75">
                {cfg.description || '—'}
              </div>
            </div>
          ))}
        </div>
      </StateBlock>
    </TabFrame>
  )
}

/* ============================ 共享布局 / 控件 ============================ */

function TabFrame({
  toolbar,
  stats,
  footer,
  isFetching,
  children,
}: {
  toolbar?: ReactNode
  stats?: ReactNode
  footer?: ReactNode
  isFetching?: boolean
  children: ReactNode
}) {
  return (
    <div className="flex h-full flex-col gap-3">
      {toolbar ? (
        <div className="flex flex-wrap items-center gap-2">
          {toolbar}
          {isFetching ? (
            <span className="flex items-center gap-1.5 text-[11px] text-cyan-300/60">
              <Loader2 className="size-3 animate-spin" />
              SYNC
            </span>
          ) : null}
        </div>
      ) : null}
      {stats ? <div className="grid grid-cols-2 gap-3 md:grid-cols-4">{stats}</div> : null}
      <div className="glass-strong relative flex-1 min-h-0 overflow-hidden rounded-sm">
        <div className="scanline" />
        <div className="relative h-full overflow-auto p-3">{children}</div>
      </div>
      {footer ? <div>{footer}</div> : null}
    </div>
  )
}

function KeywordToolbar({
  placeholder,
  value,
  onChange,
  onRefresh,
}: {
  placeholder: string
  value: string
  onChange: (v: string) => void
  onRefresh?: () => void
}) {
  return (
    <>
      <div className="relative">
        <Search className="pointer-events-none absolute left-3 top-1/2 size-3.5 -translate-y-1/2 text-cyan-300/50" />
        <input
          className="neon-input w-72 pl-9"
          placeholder={placeholder}
          value={value}
          onChange={(e) => onChange(e.target.value)}
        />
      </div>
      {onRefresh ? (
        <NeonButton icon={<RefreshCcw />} onClick={onRefresh}>
          REFRESH
        </NeonButton>
      ) : null}
    </>
  )
}

function RowHeader({ cols, children }: { cols: string; children: ReactNode }) {
  return (
    <div
      className="grid items-center gap-3 px-3 pb-1 font-mono text-[10px] uppercase tracking-[0.18em] text-cyan-300/45"
      style={{ gridTemplateColumns: cols.split('_').join(' ') }}
    >
      {children}
    </div>
  )
}

function Pager({
  page,
  totalPages,
  total,
  pageSize,
  onPage,
}: {
  page: number
  totalPages: number
  total: number
  pageSize: number
  onPage: (updater: (p: number) => number) => void
}) {
  return (
    <div className="flex items-center justify-between">
      <span className="font-mono text-[11px] text-cyan-300/55">
        PAGE {page} / {totalPages} · {pageSize}/PAGE · TOTAL {total}
      </span>
      <div className="flex gap-2">
        <NeonButton onClick={() => onPage((p) => Math.max(1, p - 1))} disabled={page <= 1}>
          ◂ PREV
        </NeonButton>
        <NeonButton
          onClick={() => onPage((p) => Math.min(totalPages, p + 1))}
          disabled={page >= totalPages}
        >
          NEXT ▸
        </NeonButton>
      </div>
    </div>
  )
}
