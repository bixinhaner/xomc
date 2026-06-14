import { useEffect, useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import {
  ChevronRight,
  Eye,
  FolderTree,
  Loader2,
  Pencil,
  Plus,
  Search,
  Trash2,
  X,
} from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
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
  Pagination,
  PageShell,
  TableCard,
} from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import {
  useCreateGroup,
  useDeleteGroup,
  useDeviceGroups,
  useDeviceList,
  useUpdateGroup,
  type CreateGroupRequest,
} from '@core/hooks/api/useDevices'
import type { DeviceGroup, DeviceFilter } from '@core/types/device'
import type { PageRequest } from '@core/types/pagination'

// ============================================================
// 设备分组 — 分组树（一级/二级）+ 分组内设备列表 + 分组增删改
// 对照 v1 webcode/src/pages/device/DeviceGrouping 的业务深度
// v1 用左树右表的 TreeListPageLayout + 多 drawer；本页保留同一信息架构，
// 用一个轻量内联对话框承载「新建/重命名分组」，匹配规则等高级编辑未覆盖。
// ============================================================

type DialogMode =
  | { kind: 'none' }
  | { kind: 'create-root' }
  | { kind: 'create-child'; parentId: string; parentName: string }
  | { kind: 'rename'; group: DeviceGroup }

function GroupNode({
  group,
  childGroups,
  selectedId,
  onSelect,
  onAddChild,
  onRename,
  onDelete,
}: {
  group: DeviceGroup
  childGroups: DeviceGroup[]
  selectedId: string | null
  onSelect: (id: string) => void
  onAddChild: (parent: DeviceGroup) => void
  onRename: (g: DeviceGroup) => void
  onDelete: (g: DeviceGroup) => void
}) {
  const [expanded, setExpanded] = useState(true)
  const isSelected = selectedId === group.id

  return (
    <div>
      <div
        className={cn(
          'group flex items-center gap-1 rounded-md px-2 py-1.5 text-sm',
          isSelected ? 'bg-primary/10 text-primary' : 'hover:bg-accent'
        )}
      >
        <button
          type="button"
          className="flex size-4 shrink-0 items-center justify-center text-muted-foreground"
          onClick={() => setExpanded((v) => !v)}
          aria-label={expanded ? '收起' : '展开'}
        >
          {childGroups.length > 0 && (
            <ChevronRight
              className={cn('size-3.5 transition-transform', expanded && 'rotate-90')}
            />
          )}
        </button>
        <button
          type="button"
          className="flex min-w-0 flex-1 items-center gap-1.5 truncate text-left"
          onClick={() => onSelect(group.id)}
        >
          <FolderTree className="size-3.5 shrink-0 opacity-70" />
          <span className="truncate">{group.name}</span>
          <span className="shrink-0 text-xs text-muted-foreground">
            ({group.deviceCount})
          </span>
        </button>
        <div className="hidden shrink-0 items-center gap-0.5 group-hover:flex">
          <button
            type="button"
            className="rounded p-0.5 text-muted-foreground hover:text-foreground"
            title="新建子分组"
            onClick={() => onAddChild(group)}
          >
            <Plus className="size-3.5" />
          </button>
          <button
            type="button"
            className="rounded p-0.5 text-muted-foreground hover:text-foreground"
            title="重命名"
            onClick={() => onRename(group)}
          >
            <Pencil className="size-3.5" />
          </button>
          {group.builtIn !== 1 && (
            <button
              type="button"
              className="rounded p-0.5 text-muted-foreground hover:text-destructive"
              title="删除"
              onClick={() => onDelete(group)}
            >
              <Trash2 className="size-3.5" />
            </button>
          )}
        </div>
      </div>
      {expanded && childGroups.length > 0 && (
        <div className="ml-4 border-l pl-1">
          {childGroups.map((child) => (
            <div
              key={child.id}
              className={cn(
                'group flex items-center gap-1 rounded-md px-2 py-1.5 text-sm',
                selectedId === child.id
                  ? 'bg-primary/10 text-primary'
                  : 'hover:bg-accent'
              )}
            >
              <button
                type="button"
                className="flex min-w-0 flex-1 items-center gap-1.5 truncate text-left"
                onClick={() => onSelect(child.id)}
              >
                <FolderTree className="size-3.5 shrink-0 opacity-50" />
                <span className="truncate">{child.name}</span>
                <span className="shrink-0 text-xs text-muted-foreground">
                  ({child.deviceCount})
                </span>
              </button>
              <div className="hidden shrink-0 items-center gap-0.5 group-hover:flex">
                <button
                  type="button"
                  className="rounded p-0.5 text-muted-foreground hover:text-foreground"
                  title="重命名"
                  onClick={() => onRename(child)}
                >
                  <Pencil className="size-3.5" />
                </button>
                {child.builtIn !== 1 && (
                  <button
                    type="button"
                    className="rounded p-0.5 text-muted-foreground hover:text-destructive"
                    title="删除"
                    onClick={() => onDelete(child)}
                  >
                    <Trash2 className="size-3.5" />
                  </button>
                )}
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function GroupDialog({
  mode,
  onClose,
  onSubmit,
  pending,
}: {
  mode: DialogMode
  onClose: () => void
  onSubmit: (name: string, remark: string) => void
  pending: boolean
}) {
  const [name, setName] = useState(mode.kind === 'rename' ? mode.group.name : '')
  const [remark, setRemark] = useState(
    mode.kind === 'rename' ? mode.group.description ?? '' : ''
  )

  if (mode.kind === 'none') return null

  const title =
    mode.kind === 'create-root'
      ? '新建一级分组'
      : mode.kind === 'create-child'
        ? `在「${mode.parentName}」下新建子分组`
        : '重命名分组'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-xl border bg-card p-5 shadow-lg">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-base font-semibold">{title}</h2>
          <button
            type="button"
            className="text-muted-foreground hover:text-foreground"
            onClick={onClose}
          >
            <X className="size-4" />
          </button>
        </div>
        <div className="flex flex-col gap-3">
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">分组名称</Label>
            <Input
              autoFocus
              placeholder="请输入分组名称"
              value={name}
              onChange={(e) => setName(e.target.value)}
            />
          </div>
          <div className="flex flex-col gap-1.5">
            <Label className="text-xs">备注</Label>
            <Input
              placeholder="可选"
              value={remark}
              onChange={(e) => setRemark(e.target.value)}
            />
          </div>
        </div>
        <div className="mt-5 flex justify-end gap-2">
          <Button variant="ghost" onClick={onClose}>
            取消
          </Button>
          <Button
            disabled={!name.trim() || pending}
            onClick={() => onSubmit(name.trim(), remark.trim())}
          >
            {pending && <Loader2 className="size-4 animate-spin" />}
            确定
          </Button>
        </div>
      </div>
    </div>
  )
}

export default function DeviceGrouping() {
  const navigate = useNavigate()
  const groupsQuery = useDeviceGroups()
  const createGroup = useCreateGroup()
  const updateGroup = useUpdateGroup()
  const deleteGroup = useDeleteGroup()

  const groups = useMemo(
    () => groupsQuery.data?.groups ?? [],
    [groupsQuery.data]
  )
  const rootGroups = useMemo(
    () => groups.filter((g) => g.parentId === null),
    [groups]
  )
  const childrenOf = useMemo(() => {
    const map = new Map<string, DeviceGroup[]>()
    for (const g of groups) {
      if (g.parentId) {
        const list = map.get(g.parentId) ?? []
        list.push(g)
        map.set(g.parentId, list)
      }
    }
    return map
  }, [groups])

  const [selectedGroupId, setSelectedGroupId] = useState<string | null>(null)
  const [treeSearch, setTreeSearch] = useState('')
  const [deviceSearch, setDeviceSearch] = useState('')
  const [page, setPage] = useState(1)
  const [pageSize] = useState(20)
  const [dialog, setDialog] = useState<DialogMode>({ kind: 'none' })

  // 默认选中第一个一级分组下的第一个子分组（与 v1 行为一致）
  useEffect(() => {
    if (selectedGroupId || groups.length === 0) return
    const firstRoot = rootGroups[0]
    if (!firstRoot) return
    const firstChild = childrenOf.get(firstRoot.id)?.[0]
    setSelectedGroupId(firstChild?.id ?? firstRoot.id)
  }, [groups, rootGroups, childrenOf, selectedGroupId])

  const filteredRoots = useMemo(() => {
    const q = treeSearch.trim().toLowerCase()
    if (!q) return rootGroups
    return rootGroups.filter((root) => {
      if (root.name.toLowerCase().includes(q)) return true
      return (childrenOf.get(root.id) ?? []).some((c) =>
        c.name.toLowerCase().includes(q)
      )
    })
  }, [rootGroups, childrenOf, treeSearch])

  const selectedGroup = useMemo(
    () => groups.find((g) => g.id === selectedGroupId) ?? null,
    [groups, selectedGroupId]
  )

  const deviceParams = useMemo<DeviceFilter & PageRequest>(
    () => ({
      page,
      pageSize,
      ...(selectedGroupId ? { groupId: selectedGroupId } : {}),
      ...(deviceSearch.trim() ? { searchText: deviceSearch.trim() } : {}),
    }),
    [page, pageSize, selectedGroupId, deviceSearch]
  )
  const deviceQuery = useDeviceList(deviceParams, {
    enabled: Boolean(selectedGroupId),
  })
  const rows = deviceQuery.data?.items ?? []
  const total = deviceQuery.data?.total ?? 0
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  function handleSubmitDialog(name: string, remark: string) {
    if (dialog.kind === 'rename') {
      updateGroup.mutate(
        { id: dialog.group.id, data: { name, remark } },
        { onSuccess: () => setDialog({ kind: 'none' }) }
      )
      return
    }
    if (dialog.kind === 'create-root' || dialog.kind === 'create-child') {
      const data: CreateGroupRequest = {
        name,
        ...(remark ? { remark } : {}),
        ...(dialog.kind === 'create-child'
          ? { parent_id: dialog.parentId }
          : {}),
      }
      createGroup.mutate(data, { onSuccess: () => setDialog({ kind: 'none' }) })
    }
  }

  function handleDelete(g: DeviceGroup) {
    if (
      !window.confirm(`确定删除分组「${g.name}」？该操作不会删除分组内的设备。`)
    )
      return
    deleteGroup.mutate(g.id, {
      onSuccess: () => {
        if (selectedGroupId === g.id) setSelectedGroupId(null)
      },
    })
  }

  const dialogPending = createGroup.isPending || updateGroup.isPending

  const colCount = 7

  return (
    <PageShell
      title="设备分组"
      description={`${rootGroups.length} 个一级分组 · 共 ${groups.length} 个分组`}
      isFetching={groupsQuery.isFetching || deviceQuery.isFetching}
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[280px_1fr]">
        {/* 分组树 */}
        <div className="flex flex-col gap-2 rounded-lg border bg-card p-3">
          <div className="flex items-center gap-2">
            <div className="relative flex-1">
              <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
              <Input
                className="h-8 pl-8 text-sm"
                placeholder="搜索分组"
                value={treeSearch}
                onChange={(e) => setTreeSearch(e.target.value)}
              />
            </div>
            <Button
              size="sm"
              variant="outline"
              className="h-8 shrink-0 px-2"
              title="新建一级分组"
              onClick={() => setDialog({ kind: 'create-root' })}
            >
              <Plus className="size-4" />
            </Button>
          </div>

          <div className="max-h-[60vh] overflow-auto">
            {groupsQuery.isLoading ? (
              <div className="flex h-32 items-center justify-center">
                <Loader2 className="size-5 animate-spin text-muted-foreground" />
              </div>
            ) : groupsQuery.isError ? (
              <div className="p-3 text-center text-sm text-destructive">
                分组加载失败
              </div>
            ) : filteredRoots.length === 0 ? (
              <div className="p-6 text-center text-sm text-muted-foreground">
                暂无分组
              </div>
            ) : (
              filteredRoots.map((root) => (
                <GroupNode
                  key={root.id}
                  group={root}
                  childGroups={childrenOf.get(root.id) ?? []}
                  selectedId={selectedGroupId}
                  onSelect={(id) => {
                    setSelectedGroupId(id)
                    setPage(1)
                  }}
                  onAddChild={(parent) =>
                    setDialog({
                      kind: 'create-child',
                      parentId: parent.id,
                      parentName: parent.name,
                    })
                  }
                  onRename={(g) => setDialog({ kind: 'rename', group: g })}
                  onDelete={handleDelete}
                />
              ))
            )}
          </div>
        </div>

        {/* 分组内设备列表 */}
        <div className="flex flex-col gap-3">
          <div className="flex flex-wrap items-center gap-2">
            <div className="text-sm font-medium">
              {selectedGroup ? selectedGroup.name : '请选择分组'}
              {selectedGroup && (
                <Badge variant="muted" className="ml-2">
                  {total} 台
                </Badge>
              )}
            </div>
            <div className="ml-auto flex items-center gap-2">
              <div className="relative">
                <Search className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground" />
                <Input
                  className="h-9 w-56 pl-8"
                  placeholder="搜索 SN / 名称"
                  value={deviceSearch}
                  onChange={(e) => {
                    setDeviceSearch(e.target.value)
                    setPage(1)
                  }}
                />
              </div>
            </div>
          </div>

          <TableCard>
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>SN</TableHead>
                  <TableHead>名称</TableHead>
                  <TableHead>连接状态</TableHead>
                  <TableHead>制式</TableHead>
                  <TableHead>产品型号</TableHead>
                  <TableHead>IP 地址</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {!selectedGroupId ? (
                  <EmptyRow colSpan={colCount}>请先在左侧选择一个分组</EmptyRow>
                ) : deviceQuery.isLoading ? (
                  <LoadingRow colSpan={colCount} />
                ) : deviceQuery.isError ? (
                  <ErrorRow colSpan={colCount} error={deviceQuery.error} />
                ) : rows.length === 0 ? (
                  <EmptyRow colSpan={colCount}>该分组下暂无设备</EmptyRow>
                ) : (
                  rows.map((d) => (
                    <TableRow key={d.id}>
                      <TableCell>
                        <span className="font-mono text-xs">{d.sn || '—'}</span>
                      </TableCell>
                      <TableCell className="text-sm">
                        {d.deviceName || d.name || '—'}
                      </TableCell>
                      <TableCell>
                        <Badge variant={d.isOnline ? 'success' : 'muted'}>
                          {d.isOnline ? '在线' : '离线'}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-xs text-muted-foreground">
                        {d.networkType || '—'}
                      </TableCell>
                      <TableCell className="text-xs">
                        {d.deviceModel || d.productClass || '—'}
                      </TableCell>
                      <TableCell className="font-mono text-xs text-muted-foreground">
                        {d.ipAddress || '—'}
                      </TableCell>
                      <TableCell className="text-right">
                        <Button
                          variant="ghost"
                          size="sm"
                          disabled={!d.sn}
                          onClick={() => navigate(`/device/detail/${d.sn}`)}
                        >
                          <Eye className="size-4" /> 详情
                        </Button>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </TableCard>

          {selectedGroupId && total > 0 && (
            <Pagination
              page={page}
              totalPages={totalPages}
              pageSize={pageSize}
              onChange={setPage}
            />
          )}
        </div>
      </div>

      <GroupDialog
        mode={dialog}
        onClose={() => setDialog({ kind: 'none' })}
        onSubmit={handleSubmitDialog}
        pending={dialogPending}
      />
    </PageShell>
  )
}
