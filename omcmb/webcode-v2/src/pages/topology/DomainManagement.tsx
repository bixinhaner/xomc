import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ChevronDown, ChevronRight, FolderTree, MapPin, RefreshCcw } from 'lucide-react'

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
import { cn } from '@/lib/utils'

import { useDomainTree, useSites } from '@core/hooks/api/useTopology'
import type { Domain, SiteStatus } from '@core/types/topology'

// ============================================================
// 域管理（v2 皮肤）— 对照 v1 webcode/pages/topology/DomainManagement
//  v1 左侧设备域树 + 右侧选中域下的设备/站点 Tab（设备列表 v1 是 mock）。
//  v2 全走真实数据：
//   - useDomainTree：真实设备组（域）树（Domain[] 带 children/level/deviceCount）
//   - useSites({ domainId })：选中域后服务端按域筛站点
//  站点行可点击跳站点详情（/topology/site/:id）。
// ============================================================

const SITE_STATUS: Record<SiteStatus, { label: string; variant: 'success' | 'warning' | 'muted' }> = {
  active: { label: '正常', variant: 'success' },
  maintenance: { label: '维护', variant: 'warning' },
  inactive: { label: '停用', variant: 'muted' },
}

// 递归树节点
function DomainTreeNode({
  node,
  depth,
  selectedId,
  expanded,
  onToggle,
  onSelect,
}: {
  node: Domain
  depth: number
  selectedId: string
  expanded: Set<string>
  onToggle: (id: string) => void
  onSelect: (node: Domain) => void
}) {
  const hasChildren = (node.children?.length ?? 0) > 0
  const isOpen = expanded.has(node.id)
  return (
    <>
      <div
        className={cn(
          'flex cursor-pointer items-center gap-1 rounded px-2 py-1.5 text-sm hover:bg-accent',
          selectedId === node.id && 'bg-accent font-medium',
        )}
        style={{ paddingLeft: 8 + depth * 16 }}
        onClick={() => onSelect(node)}
      >
        {hasChildren ? (
          <button
            type="button"
            className="flex size-4 items-center justify-center text-muted-foreground"
            onClick={(e) => {
              e.stopPropagation()
              onToggle(node.id)
            }}
          >
            {isOpen ? <ChevronDown className="size-3.5" /> : <ChevronRight className="size-3.5" />}
          </button>
        ) : (
          <span className="inline-block size-4" />
        )}
        <FolderTree className="size-3.5 shrink-0 text-muted-foreground" />
        <span className="truncate">{node.name}</span>
        {node.deviceCount > 0 && (
          <span className="ml-auto text-[11px] tabular-nums text-muted-foreground">
            {node.deviceCount}
          </span>
        )}
      </div>
      {hasChildren && isOpen &&
        node.children!.map((c) => (
          <DomainTreeNode
            key={c.id}
            node={c}
            depth={depth + 1}
            selectedId={selectedId}
            expanded={expanded}
            onToggle={onToggle}
            onSelect={onSelect}
          />
        ))}
    </>
  )
}

function collectIds(nodes: Domain[] | undefined, out: Set<string> = new Set()): Set<string> {
  for (const n of nodes ?? []) {
    if (n.children?.length) {
      out.add(n.id)
      collectIds(n.children, out)
    }
  }
  return out
}

export default function DomainManagement() {
  const navigate = useNavigate()
  const { data: tree, isLoading: isTreeLoading, isFetching, refetch } = useDomainTree()

  const [selectedDomain, setSelectedDomain] = useState<Domain | null>(null)
  const [expanded, setExpanded] = useState<Set<string>>(new Set())

  // 首次拿到树时默认展开全部
  const allExpandableIds = useMemo(() => collectIds(tree), [tree])
  const [initialized, setInitialized] = useState(false)
  if (!initialized && allExpandableIds.size > 0) {
    setExpanded(new Set(allExpandableIds))
    setInitialized(true)
  }

  const {
    data: sitesData,
    isLoading: isSitesLoading,
    isError: isSitesError,
    error: sitesError,
  } = useSites(selectedDomain ? { domainId: selectedDomain.id } : undefined)

  const sites = sitesData?.items ?? []
  const cols = ['站点名称', '地址', '设备数', '状态', '坐标']

  const toggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  return (
    <PageShell
      title="域管理"
      description="设备域树 · 选中域查看其站点"
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full items-center gap-2">
          <div className="ml-auto">
            <Button variant="outline" size="sm" onClick={() => refetch()}>
              <RefreshCcw className="mr-1 h-4 w-4" />
              刷新
            </Button>
          </div>
        </div>
      }
    >
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[300px_1fr]">
        {/* 左：域树 */}
        <div className="flex max-h-[600px] flex-col overflow-auto rounded-lg border bg-card p-2">
          <div className="px-2 pb-2 pt-1 text-xs font-medium text-muted-foreground">设备域</div>
          {isTreeLoading ? (
            <div className="p-4 text-center text-sm text-muted-foreground">加载中...</div>
          ) : (tree?.length ?? 0) === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">暂无域数据</div>
          ) : (
            tree!.map((n) => (
              <DomainTreeNode
                key={n.id}
                node={n}
                depth={0}
                selectedId={selectedDomain?.id ?? ''}
                expanded={expanded}
                onToggle={toggle}
                onSelect={setSelectedDomain}
              />
            ))
          )}
        </div>

        {/* 右：选中域站点 */}
        <div className="flex flex-col gap-3">
          {selectedDomain ? (
            <>
              <div className="flex flex-wrap items-center gap-3 rounded-lg border bg-card px-4 py-3">
                <div>
                  <div className="text-base font-medium">{selectedDomain.name}</div>
                  <div className="mt-0.5 text-xs text-muted-foreground">
                    层级 L{selectedDomain.level} · 设备数 {selectedDomain.deviceCount}
                  </div>
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
                    {isSitesLoading ? (
                      <LoadingRow colSpan={cols.length} />
                    ) : isSitesError ? (
                      <ErrorRow colSpan={cols.length} error={sitesError} />
                    ) : sites.length === 0 ? (
                      <EmptyRow colSpan={cols.length}>该域下暂无站点</EmptyRow>
                    ) : (
                      sites.map((s) => {
                        const meta = SITE_STATUS[s.status] ?? SITE_STATUS.inactive
                        return (
                          <TableRow
                            key={s.id}
                            className="cursor-pointer"
                            onClick={() => navigate(`/topology/site/${s.id}`)}
                          >
                            <TableCell className="font-medium text-primary hover:underline">
                              {s.name}
                            </TableCell>
                            <TableCell className="text-xs text-muted-foreground">
                              {s.address}
                            </TableCell>
                            <TableCell className="tabular-nums">{s.deviceCount}</TableCell>
                            <TableCell>
                              <Badge variant={meta.variant}>{meta.label}</Badge>
                            </TableCell>
                            <TableCell className="font-mono text-xs text-muted-foreground">
                              <span className="inline-flex items-center gap-1">
                                <MapPin className="h-3 w-3" />
                                {s.latitude != null && s.longitude != null
                                  ? `${s.latitude.toFixed(3)}, ${s.longitude.toFixed(3)}`
                                  : '—'}
                              </span>
                            </TableCell>
                          </TableRow>
                        )
                      })
                    )}
                  </TableBody>
                </Table>
              </TableCard>
            </>
          ) : (
            <div className="flex h-64 flex-col items-center justify-center gap-2 rounded-lg border bg-card text-muted-foreground">
              <FolderTree className="size-8 opacity-40" />
              <span className="text-sm">请从左侧选择一个域查看其站点</span>
            </div>
          )}
        </div>
      </div>
    </PageShell>
  )
}
