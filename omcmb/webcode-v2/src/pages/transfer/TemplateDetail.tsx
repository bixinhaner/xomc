import { useMemo } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'

import { useUnifiedFileTransferTaskTypes } from '@core/hooks/api/useUnifiedFileTransfer'
import type { UnifiedFileTransferTaskType } from '@core/types/unifiedFileTransfer'

import { RPC_TYPE_LABEL, STEP_LABEL } from './_shared'

// ============================================================
// 模板详情 — 从模板定义管理列表点进 (按 typeCode 定位)
// ============================================================

const SOFT_LIB_LABEL: Record<number, string> = {
  0: '系统镜像',
  1: '补丁',
  5: 'FPGA',
  6: 'License',
}

export default function TemplateDetail() {
  const { typeCode = '' } = useParams<{ typeCode: string }>()
  const navigate = useNavigate()
  const decoded = decodeURIComponent(typeCode)

  const { data: taskTypes = [], isLoading, isError, error, isFetching } =
    useUnifiedFileTransferTaskTypes({ refetchOnMount: 'always' })

  const template = useMemo(
    () => taskTypes.find((tt) => tt.typeCode === decoded),
    [taskTypes, decoded]
  )

  return (
    <PageShell
      title={template ? template.displayName || template.typeCode : '模板详情'}
      description={template ? template.categoryLabel || template.category : undefined}
      isFetching={isFetching}
      toolbar={
        <div className="flex w-full flex-wrap items-center gap-2">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/transfer/template-management')}
          >
            <ArrowLeft className="size-4" /> 返回模板列表
          </Button>
        </div>
      }
    >
      {isLoading ? (
        <div className="flex h-64 items-center justify-center">
          <Loader2 className="size-6 animate-spin text-muted-foreground" />
        </div>
      ) : isError ? (
        <Card className="flex h-48 items-center justify-center p-6 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </Card>
      ) : !template ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到模板 {decoded}</span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/transfer/template-management')}
          >
            返回模板列表
          </Button>
        </Card>
      ) : (
        <Detail template={template} />
      )}
    </PageShell>
  )
}

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 py-1.5">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-sm">{value ?? '—'}</span>
    </div>
  )
}

function Detail({ template: tt }: { template: UnifiedFileTransferTaskType }) {
  return (
    <div className="flex flex-col gap-4">
      <Card>
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-base font-medium">基本信息</CardTitle>
        </CardHeader>
        <CardContent className="grid grid-cols-1 gap-x-8 p-4 pt-2 sm:grid-cols-2 lg:grid-cols-3">
          <InfoRow label="显示名称" value={tt.displayName} />
          <InfoRow
            label="类型编码"
            value={<span className="font-mono text-xs">{tt.typeCode}</span>}
          />
          <InfoRow label="业务分类" value={tt.categoryLabel || tt.category} />
          <InfoRow label="RPC 类型" value={RPC_TYPE_LABEL[tt.rpcType] ?? tt.rpcType} />
          <InfoRow
            label="来源"
            value={
              <Badge variant={tt.builtIn ? 'muted' : 'default'}>
                {tt.builtIn ? '内置' : '自定义'}
              </Badge>
            }
          />
          <InfoRow
            label="状态"
            value={
              <Badge variant={tt.enabled ? 'success' : 'muted'}>
                {tt.enabled ? '启用' : '停用'}
              </Badge>
            }
          />
          <InfoRow label="文件类型" value={tt.fileTypeLabel || tt.fileType} />
          <InfoRow
            label="文件类型可编辑"
            value={tt.fileTypeEditable ? '是' : '否'}
          />
          <InfoRow
            label="软件库文件类型"
            value={
              tt.firmwareFileType != null
                ? SOFT_LIB_LABEL[tt.firmwareFileType] ?? String(tt.firmwareFileType)
                : '—'
            }
          />
          <InfoRow label="权限码" value={tt.permissionCode} />
          <InfoRow label="延迟秒数" value={tt.delaySeconds ?? 0} />
          <InfoRow label="后置 TC 事件码" value={tt.postTcEventCode || '—'} />
          <InfoRow label="最后编辑人" value={tt.lastEditor} />
          <InfoRow
            label="近30天任务数"
            value={<span className="tabular-nums">{tt.taskCount30d}</span>}
          />
          <InfoRow
            label="近30天成功率"
            value={<span className="tabular-nums">{tt.successRate30d}%</span>}
          />
          <InfoRow label="更新时间" value={formatTime(tt.updatedAt)} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader className="p-4 pb-2">
          <CardTitle className="text-base font-medium">模板描述</CardTitle>
        </CardHeader>
        <CardContent className="p-4 pt-2 text-sm">{tt.description || '—'}</CardContent>
      </Card>

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">执行步骤链</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-1.5 p-4 pt-2">
            {tt.stepChain.length === 0 ? (
              <span className="text-sm text-muted-foreground">—</span>
            ) : (
              tt.stepChain.map((step, i) => (
                <Badge key={`${step}-${i}`} variant="outline">
                  {i + 1}. {STEP_LABEL[step] ?? step}
                </Badge>
              ))
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="p-4 pb-2">
            <CardTitle className="text-base font-medium">适用平台范围</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-1.5 p-4 pt-2">
            {tt.platformScope.length === 0 ? (
              <span className="text-sm text-muted-foreground">全部平台</span>
            ) : (
              tt.platformScope.map((p) => (
                <Badge key={p} variant="secondary">
                  {p}
                </Badge>
              ))
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
