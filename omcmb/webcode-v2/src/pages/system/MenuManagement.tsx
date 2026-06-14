import { useMemo } from 'react'
import { Layers, RefreshCcw } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
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
  TableCard,
} from '@/components/layout/PageShell'

import { useMenuTree } from '@core/hooks/api/useAdmin'
import type { MenuItem } from '@core/types/system'

// ============================================================
// 系统管理 / 菜单管理 — 对齐 v1 webcode/src/pages/system/MenuManagement
// 真实数据 useMenuTree（adminApi.getMenuTree）。RBAC 菜单权限树（只读视图）。
// ============================================================

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

export default function MenuManagement() {
  const { data, isLoading, isError, error, isFetching, refetch } = useMenuTree()
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
    <PageShell
      title="菜单管理"
      description={`RBAC 菜单权限树 · 共 ${flatCount} 个菜单项`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <span className="text-sm text-muted-foreground">菜单权限树（只读视图）</span>
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="size-4" /> 刷新
            </Button>
          </div>
        </div>
      }
    >
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
    </PageShell>
  )
}
