import { useMemo, useState } from 'react'
import { RefreshCcw, X } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
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
  formatTime,
} from '@/components/layout/PageShell'

import { useUnifiedFileTransferTaskTypes } from '@core/hooks/api/useUnifiedFileTransfer'
import {
  DEVICE_UPGRADE_CATEGORY,
  aggregateCategoryOptions,
  filterTaskTypesForCategory,
} from '@core/utils/ufteCategory'

import type { UnifiedFileTransferTaskType } from '@core/types/unifiedFileTransfer'
import { RPC_TYPE_LABEL } from './_shared'

// ============================================================
// 模板定义管理 — 对照 v1 transfer/template-management
// 内置 + 自定义任务类型模板列表,按业务分类过滤,可点进模板详情
// ============================================================

const ALL = 'all'

export default function TemplateManagement() {
  const { data: taskTypes = [], isLoading, isError, error, isFetching, refetch } =
    useUnifiedFileTransferTaskTypes({ refetchOnMount: 'always' })

  const [category, setCategory] = useState<string>(ALL)
  const [kind, setKind] = useState<'all' | 'builtin' | 'custom'>('all')
  const [selectedTemplate, setSelectedTemplate] = useState<UnifiedFileTransferTaskType | null>(null)

  // #483：4G/5G/2G 折叠为单条『设备升级』(device_upgrade)，与任务创建页一致
  // （共享 @core/utils/ufteCategory，三皮肤同一口径）。
  const categoryOptions = useMemo(
    () =>
      aggregateCategoryOptions(taskTypes).map((opt) =>
        opt.value === DEVICE_UPGRADE_CATEGORY ? { value: opt.value, label: '设备升级' } : opt
      ),
    [taskTypes]
  )

  const rows = useMemo(() => {
    // device_upgrade 是折叠虚拟分类，需经 filterTaskTypesForCategory 展开成员（4G/5G/2G）；
    // 精确等值会漏掉真实分类 enb/gnb/gsm_upgrade 的模板。
    const byCategory = category === ALL ? taskTypes : filterTaskTypesForCategory(taskTypes, category)
    return byCategory.filter((tt) => {
      if (kind === 'builtin' && !tt.builtIn) return false
      if (kind === 'custom' && tt.builtIn) return false
      return true
    })
  }, [taskTypes, category, kind])

  const builtinCount = taskTypes.filter((tt) => tt.builtIn).length
  const customCount = taskTypes.length - builtinCount
  const cols = 8

  return (
    <>
    <PageShell
      title="模板定义管理"
      description={`共 ${taskTypes.length} 个模板 · 内置 ${builtinCount} · 自定义 ${customCount}`}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Select value={category} onValueChange={setCategory}>
            <SelectTrigger className="w-44">
              <SelectValue placeholder="业务分类" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value={ALL}>全部分类</SelectItem>
              {categoryOptions.map((o) => (
                <SelectItem key={o.value} value={o.value}>
                  {o.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          <Select value={kind} onValueChange={(v) => setKind(v as typeof kind)}>
            <SelectTrigger className="w-32">
              <SelectValue placeholder="来源" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部来源</SelectItem>
              <SelectItem value="builtin">内置</SelectItem>
              <SelectItem value="custom">自定义</SelectItem>
            </SelectContent>
          </Select>

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
              <TableHead>显示名称</TableHead>
              <TableHead>类型编码</TableHead>
              <TableHead>业务分类</TableHead>
              <TableHead>RPC 类型</TableHead>
              <TableHead>来源</TableHead>
              <TableHead>状态</TableHead>
              <TableHead>近30天任务数</TableHead>
              <TableHead>更新时间</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {isLoading ? (
              <LoadingRow colSpan={cols} />
            ) : isError ? (
              <ErrorRow colSpan={cols} error={error} />
            ) : rows.length === 0 ? (
              <EmptyRow colSpan={cols}>暂无匹配的模板</EmptyRow>
            ) : (
              rows.map((tt) => (
                <TableRow key={tt.typeCode}>
                  <TableCell>
                    <button
                      type="button"
                      className="text-left text-sm font-medium text-primary hover:underline"
                      onClick={() => setSelectedTemplate(tt)}
                    >
                      {tt.displayName || tt.typeCode}
                    </button>
                  </TableCell>
                  <TableCell>
                    <span className="font-mono text-xs text-muted-foreground">{tt.typeCode}</span>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs text-muted-foreground">
                      {tt.categoryLabel || tt.category}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs">{RPC_TYPE_LABEL[tt.rpcType] ?? tt.rpcType}</span>
                  </TableCell>
                  <TableCell>
                    <Badge variant={tt.builtIn ? 'muted' : 'default'}>
                      {tt.builtIn ? '内置' : '自定义'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant={tt.enabled ? 'success' : 'muted'}>
                      {tt.enabled ? '启用' : '停用'}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs tabular-nums text-muted-foreground">
                      {tt.taskCount30d}
                    </span>
                  </TableCell>
                  <TableCell>
                    <span className="text-xs text-muted-foreground">
                      {formatTime(tt.updatedAt)}
                    </span>
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableCard>
    </PageShell>
    {selectedTemplate && (
      <>
        <div
          className="fixed inset-0 z-40 bg-black/40"
          onClick={() => setSelectedTemplate(null)}
          aria-hidden
        />
        <aside
          className="fixed inset-y-0 right-0 z-50 w-[520px] max-w-[92vw] overflow-y-auto border-l bg-background shadow-xl"
          role="dialog"
          aria-modal="true"
        >
          <div className="flex items-center justify-between border-b px-5 py-4">
            <div>
              <p className="font-semibold">{selectedTemplate.displayName || selectedTemplate.typeCode}</p>
              <p className="font-mono text-xs text-muted-foreground">{selectedTemplate.typeCode}</p>
            </div>
            <Button variant="ghost" size="icon" onClick={() => setSelectedTemplate(null)}>
              <X className="size-4" />
            </Button>
          </div>
          <div className="space-y-5 p-5">
            <div className="grid grid-cols-2 gap-3 text-sm">
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">业务分类</p>
                <p>{selectedTemplate.categoryLabel || selectedTemplate.category}</p>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">RPC 类型</p>
                <p>{RPC_TYPE_LABEL[selectedTemplate.rpcType] ?? selectedTemplate.rpcType}</p>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">文件类型</p>
                <p>{selectedTemplate.fileType || '—'}</p>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">权限码</p>
                <p className="font-mono text-xs">{selectedTemplate.permissionCode || '—'}</p>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">文件类型可编辑</p>
                <Badge variant={selectedTemplate.fileTypeEditable ? 'success' : 'muted'}>
                  {selectedTemplate.fileTypeEditable ? '可编辑' : '固定'}
                </Badge>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">延迟秒数</p>
                <p>{selectedTemplate.delaySeconds ?? 0} 秒</p>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">来源</p>
                <Badge variant={selectedTemplate.builtIn ? 'muted' : 'default'}>
                  {selectedTemplate.builtIn ? '内置' : '自定义'}
                </Badge>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">状态</p>
                <Badge variant={selectedTemplate.enabled ? 'success' : 'muted'}>
                  {selectedTemplate.enabled ? '启用' : '停用'}
                </Badge>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">最后编辑</p>
                <p>{selectedTemplate.lastEditor || '—'}</p>
              </div>
              <div>
                <p className="mb-0.5 text-xs text-muted-foreground">后置事件</p>
                <p className="font-mono text-xs">{selectedTemplate.postTcEventCode || '—'}</p>
              </div>
            </div>
            <div>
              <p className="mb-1 text-xs text-muted-foreground">功能描述</p>
              <p className="text-sm leading-relaxed">{selectedTemplate.description || '—'}</p>
            </div>
            <div>
              <p className="mb-1 text-xs text-muted-foreground">适用产品</p>
              <p className="text-sm">
                {(selectedTemplate.products ?? []).length > 0
                  ? (selectedTemplate.products ?? []).join(' / ')
                  : '全部产品'}
              </p>
            </div>
            <div>
              <p className="mb-2 text-xs text-muted-foreground">步骤链</p>
              <div className="flex flex-wrap gap-2">
                {selectedTemplate.stepChain.map((stepId, i) => (
                  <Badge
                    key={stepId}
                    variant={i < 2 ? 'default' : i === selectedTemplate.stepChain.length - 1 ? 'secondary' : 'outline'}
                  >
                    {i + 1}. {stepId}
                  </Badge>
                ))}
              </div>
            </div>
          </div>
        </aside>
      </>
    )}
  </>
  )
}
