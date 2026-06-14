import { useMemo, useState } from 'react'
import {
  Boxes,
  Database,
  HardDrive,
  Layers,
  RefreshCcw,
  Search,
  Server,
  ShieldCheck,
  Users as UsersIcon,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import {
  EmptyRow,
  ErrorRow,
  LoadingRow,
  PageShell,
  Pagination,
  TableCard,
  formatTime,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useGroups,
  useRoles,
  useSystemInfo,
  useUsers,
} from '@core/hooks/api/useSystem'
import { useMenuTree } from '@core/hooks/api/useAdmin'
import type { MenuItem, UserStatus } from '@core/types/system'

// ============================================================
// 系统管理（system）— v2 皮肤
// v1 业务语义源：webcode/src/pages/system/{UserManagement,RolePermission,
//   GroupManagement,MenuManagement,...}。本页把 v1 拆成多个子菜单页的
//   用户 / 角色 / 用户组 / 菜单 整合为单页多视图（Tab），并叠加系统信息概览。
// 深度下钻（创建/编辑 Drawer、角色菜单授权树、批量锁定/强制下线等写操作）未覆盖，
// 详见返回 notes。
// ============================================================

type ViewKey = 'users' | 'roles' | 'groups' | 'menus'

const TABS: { key: ViewKey; label: string }[] = [
  { key: 'users', label: '用户' },
  { key: 'roles', label: '角色' },
  { key: 'groups', label: '用户组' },
  { key: 'menus', label: '菜单' },
]

const PAGE_SIZE = 20

const USER_STATUS_META: Record<
  UserStatus,
  { label: string; variant: 'success' | 'muted' | 'warning' | 'destructive' }
> = {
  active: { label: '启用', variant: 'success' },
  disabled: { label: '禁用', variant: 'muted' },
  inactive: { label: '未激活', variant: 'muted' },
  locked: { label: '已锁定', variant: 'destructive' },
}

const USER_SOURCE_LABEL: Record<string, string> = {
  builtIn: '内置',
  admin: '本地',
  LDAP: 'LDAP',
}

function StatusDot({ ok }: { ok: boolean }) {
  return (
    <span
      className={cn(
        'mr-1.5 inline-block size-2 rounded-full',
        ok ? 'bg-emerald-500' : 'bg-destructive'
      )}
    />
  )
}

// ---- 系统信息概览卡 ----
function SystemOverview() {
  const { data, isLoading, isError } = useSystemInfo()

  const cards: {
    icon: typeof Server
    label: string
    value: string
    badge?: { text: string; ok: boolean }
  }[] = useMemo(() => {
    if (!data) return []
    const dbOk = (data.dbStatus ?? '').toLowerCase() === 'ok' ||
      (data.dbStatus ?? '').toLowerCase() === 'up' ||
      (data.dbStatus ?? '').toLowerCase() === 'healthy'
    const cacheOk = (data.cacheStatus ?? '').toLowerCase() === 'ok' ||
      (data.cacheStatus ?? '').toLowerCase() === 'up' ||
      (data.cacheStatus ?? '').toLowerCase() === 'healthy'
    return [
      { icon: Server, label: '版本', value: data.version || '—' },
      {
        icon: HardDrive,
        label: '运行时长',
        value:
          data.uptimeHours != null
            ? `${data.uptimeHours.toFixed(1)} 小时`
            : '—',
      },
      {
        icon: Database,
        label: '数据库',
        value: data.dbStatus || '—',
        badge: { text: dbOk ? '正常' : '异常', ok: dbOk },
      },
      {
        icon: Boxes,
        label: '缓存',
        value: data.cacheStatus || '—',
        badge: { text: cacheOk ? '正常' : '异常', ok: cacheOk },
      },
    ]
  }, [data])

  if (isLoading) {
    return (
      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <div
            key={i}
            className="h-[78px] animate-pulse rounded-lg border bg-card"
          />
        ))}
      </div>
    )
  }

  if (isError || !data) {
    return (
      <div className="mb-6 rounded-lg border bg-card px-4 py-3 text-sm text-muted-foreground">
        系统信息暂不可用
      </div>
    )
  }

  return (
    <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-4">
      {cards.map((c) => {
        const Icon = c.icon
        return (
          <div key={c.label} className="rounded-lg border bg-card px-4 py-3">
            <div className="flex items-center gap-1.5 text-xs uppercase tracking-wider text-muted-foreground">
              <Icon className="size-3.5" />
              {c.label}
            </div>
            <div className="mt-1.5 flex items-center gap-2">
              <span className="truncate text-lg font-semibold tabular-nums">
                {c.value}
              </span>
              {c.badge ? (
                <span className="inline-flex items-center text-xs">
                  <StatusDot ok={c.badge.ok} />
                  <span
                    className={cn(
                      c.badge.ok ? 'text-emerald-600' : 'text-destructive'
                    )}
                  >
                    {c.badge.text}
                  </span>
                </span>
              ) : null}
            </div>
            {c.label === '版本' && data.buildDate ? (
              <div className="mt-0.5 text-[11px] text-muted-foreground">
                构建于 {formatTime(data.buildDate)}
              </div>
            ) : null}
          </div>
        )
      })}
    </div>
  )
}

// ---- 用户视图 ----
function UsersView() {
  const [page, setPage] = useState(1)
  const [userName, setUserName] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(userName.trim() ? { userName: userName.trim() } : {}),
    }),
    [page, userName]
  )

  const { data, isLoading, isError, error, refetch } = useUsers(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['用户', '账号', '角色', '来源', '状态', '最近登录', '创建时间']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索用户名"
            value={userName}
            onChange={(e) => {
              setUserName(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          <span>共 {total} 个用户</span>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无用户</EmptyRow>
            ) : (
              rows.map((u) => {
                const statusMeta =
                  USER_STATUS_META[u.status] ?? USER_STATUS_META.disabled
                const roleNames = u.roles ?? []
                return (
                  <TableRow key={u.id}>
                    <TableCell>
                      <div className="font-medium">
                        {u.displayName || u.username}
                      </div>
                      {u.email ? (
                        <div className="text-xs text-muted-foreground">
                          {u.email}
                        </div>
                      ) : null}
                    </TableCell>
                    <TableCell className="font-mono text-xs">
                      {u.username}
                    </TableCell>
                    <TableCell className="text-xs">
                      {roleNames.length === 0 ? (
                        <span className="text-muted-foreground">—</span>
                      ) : (
                        <div className="flex flex-wrap gap-1">
                          {roleNames.slice(0, 2).map((r) => (
                            <Badge key={r} variant="outline">
                              {r}
                            </Badge>
                          ))}
                          {roleNames.length > 2 ? (
                            <Badge variant="muted">
                              +{roleNames.length - 2}
                            </Badge>
                          ) : null}
                        </div>
                      )}
                    </TableCell>
                    <TableCell className="text-xs">
                      <Badge
                        variant={u.source === 'builtIn' ? 'default' : 'outline'}
                      >
                        {u.source ? USER_SOURCE_LABEL[u.source] ?? u.source : '—'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={statusMeta.variant}>
                        {statusMeta.label}
                      </Badge>
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(u.lastLoginTime)}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(u.createTime)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </>
  )
}

// ---- 角色视图 ----
function RolesView() {
  const [page, setPage] = useState(1)
  const [roleName, setRoleName] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(roleName.trim() ? { roleName: roleName.trim() } : {}),
    }),
    [page, roleName]
  )

  const { data, isLoading, isError, error, refetch } = useRoles(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['角色名', '编码', '用户数', '设备权限', '内置', '描述', '更新时间']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索角色名"
            value={roleName}
            onChange={(e) => {
              setRoleName(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          <span>共 {total} 个角色</span>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无角色</EmptyRow>
            ) : (
              rows.map((r) => {
                const isBuiltIn = r.builtIn === 1 || r.builtIn === 2
                const groupCount = r.deviceGroupIds?.length ?? 0
                return (
                  <TableRow key={r.id}>
                    <TableCell className="font-medium">{r.roleName}</TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {r.roleCode}
                    </TableCell>
                    <TableCell className="tabular-nums">{r.userCount}</TableCell>
                    <TableCell className="text-xs">
                      {isBuiltIn ? (
                        <Badge variant="default">全部设备</Badge>
                      ) : groupCount > 0 ? (
                        <Badge variant="outline">{groupCount} 个分组</Badge>
                      ) : (
                        <Badge variant="warning">未绑定</Badge>
                      )}
                    </TableCell>
                    <TableCell>
                      {isBuiltIn ? (
                        <Badge variant="secondary">内置</Badge>
                      ) : (
                        <span className="text-xs text-muted-foreground">—</span>
                      )}
                    </TableCell>
                    <TableCell
                      className="max-w-[220px] truncate text-xs text-muted-foreground"
                      title={r.description}
                    >
                      {r.description || '—'}
                    </TableCell>
                    <TableCell className="text-xs text-muted-foreground">
                      {formatTime(r.updateTime)}
                    </TableCell>
                  </TableRow>
                )
              })
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </>
  )
}

// ---- 用户组视图 ----
function GroupsView() {
  const [page, setPage] = useState(1)
  const [groupName, setGroupName] = useState('')

  const params = useMemo(
    () => ({
      page,
      pageSize: PAGE_SIZE,
      ...(groupName.trim() ? { groupName: groupName.trim() } : {}),
    }),
    [page, groupName]
  )

  const { data, isLoading, isError, error, refetch } = useGroups(params)
  const rows = data?.items ?? []
  const total = data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE))

  const cols = ['用户组', '用户数', '角色数', '内置', '描述', '更新人', '更新时间']

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="relative">
          <Search className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input
            className="w-72 pl-9"
            placeholder="搜索用户组"
            value={groupName}
            onChange={(e) => {
              setGroupName(e.target.value)
              setPage(1)
            }}
          />
        </div>
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          <span>共 {total} 个用户组</span>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无用户组</EmptyRow>
            ) : (
              rows.map((g) => (
                <TableRow key={g.id}>
                  <TableCell className="font-medium">{g.groupName}</TableCell>
                  <TableCell className="tabular-nums">{g.userCount}</TableCell>
                  <TableCell className="tabular-nums">{g.roleCount}</TableCell>
                  <TableCell>
                    {g.builtIn ? (
                      <Badge variant="secondary">内置</Badge>
                    ) : (
                      <span className="text-xs text-muted-foreground">—</span>
                    )}
                  </TableCell>
                  <TableCell
                    className="max-w-[240px] truncate text-xs text-muted-foreground"
                    title={g.description}
                  >
                    {g.description || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {g.updUser || '—'}
                  </TableCell>
                  <TableCell className="text-xs text-muted-foreground">
                    {formatTime(g.updTime)}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>

      <Pagination
        page={page}
        totalPages={totalPages}
        pageSize={PAGE_SIZE}
        onChange={setPage}
      />
    </>
  )
}

// ---- 菜单树视图 ----
const MENU_TYPE_LABEL: Record<string, string> = {
  menu: '菜单',
  button: '按钮',
  link: '链接',
}

function MenuTreeRows({
  items,
  depth,
}: {
  items: MenuItem[]
  depth: number
}): React.ReactElement {
  return (
    <>
      {items.map((m) => (
        <MenuTreeRow key={m.id} item={m} depth={depth} />
      ))}
    </>
  )
}

function MenuTreeRow({ item, depth }: { item: MenuItem; depth: number }) {
  const hasChildren = (item.children?.length ?? 0) > 0
  return (
    <>
      <TableRow>
        <TableCell>
          <div
            className="flex items-center gap-2"
            style={{ paddingLeft: depth * 18 }}
          >
            {hasChildren ? (
              <Layers className="size-3.5 text-muted-foreground" />
            ) : (
              <span className="inline-block size-3.5" />
            )}
            <span className="font-medium">{item.title}</span>
            <span className="text-xs text-muted-foreground">{item.name}</span>
          </div>
        </TableCell>
        <TableCell className="text-xs">
          <Badge variant={item.type === 'menu' ? 'default' : 'outline'}>
            {MENU_TYPE_LABEL[item.type] ?? item.type}
          </Badge>
        </TableCell>
        <TableCell className="font-mono text-xs text-muted-foreground">
          {item.path || '—'}
        </TableCell>
        <TableCell className="tabular-nums text-xs text-muted-foreground">
          {item.sortOrder}
        </TableCell>
        <TableCell>
          <Badge variant={item.status === 'active' ? 'success' : 'muted'}>
            {item.status === 'active' ? '启用' : '禁用'}
          </Badge>
        </TableCell>
        <TableCell>
          <Badge variant={item.visible ? 'outline' : 'muted'}>
            {item.visible ? '可见' : '隐藏'}
          </Badge>
        </TableCell>
      </TableRow>
      {hasChildren ? (
        <MenuTreeRows items={item.children ?? []} depth={depth + 1} />
      ) : null}
    </>
  )
}

function MenusView() {
  const { data, isLoading, isError, error, refetch } = useMenuTree()
  const rows = data ?? []
  const cols = ['菜单', '类型', '路径', '排序', '状态', '可见性']

  const flatCount = useMemo(() => {
    let count = 0
    const walk = (items: MenuItem[]) => {
      for (const it of items) {
        count += 1
        if (it.children?.length) walk(it.children)
      }
    }
    walk(rows)
    return count
  }, [rows])

  return (
    <>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <div className="text-sm text-muted-foreground">
          菜单权限树（只读视图）
        </div>
        <div className="ml-auto flex items-center gap-2 text-xs text-muted-foreground">
          <span>共 {flatCount} 个菜单项</span>
          <Button variant="outline" size="sm" onClick={() => refetch()}>
            <RefreshCcw /> 刷新
          </Button>
        </div>
      </div>

      <TableCard>
        <Table>
          <TableHeader>
            <TableRow>
              {cols.map((c) => (
                <TableHead key={c}>{c}</TableHead>
              ))}
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols.length} />
            ) : isError ? (
              <ErrorRow colSpan={cols.length} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols.length}>暂无菜单</EmptyRow>
            ) : (
              <MenuTreeRows items={rows} depth={0} />
            )}
          </TableBody>
        </Table>
      </TableCard>
    </>
  )
}

const TAB_ICON: Record<ViewKey, typeof UsersIcon> = {
  users: UsersIcon,
  roles: ShieldCheck,
  groups: Boxes,
  menus: Layers,
}

export function SystemPage() {
  const [view, setView] = useState<ViewKey>('users')

  // 顶部 isFetching 指示：聚合当前视图所属查询。这里只反映用户列表后台刷新，
  // 其余视图各自在视图内的刷新按钮触发，避免跨视图无关 query 干扰。
  return (
    <PageShell
      title="系统管理"
      description="用户 · 角色 · 用户组 · 菜单权限 · 系统信息"
      toolbar={
        <div className="flex flex-wrap items-center gap-1.5">
          {TABS.map((t) => {
            const Icon = TAB_ICON[t.key]
            const active = view === t.key
            return (
              <Button
                key={t.key}
                size="sm"
                variant={active ? 'default' : 'outline'}
                onClick={() => setView(t.key)}
              >
                <Icon className="size-4" />
                {t.label}
              </Button>
            )
          })}
        </div>
      }
    >
      <SystemOverview />

      {view === 'users' ? (
        <UsersView />
      ) : view === 'roles' ? (
        <RolesView />
      ) : view === 'groups' ? (
        <GroupsView />
      ) : (
        <MenusView />
      )}
    </PageShell>
  )
}
