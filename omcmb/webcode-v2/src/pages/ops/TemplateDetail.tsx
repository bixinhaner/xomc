import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { PageShell, formatTime } from '@/components/layout/PageShell'
import { cn } from '@/lib/utils'

import { useOpsTemplateById } from '@core/hooks/api/useOpsTools'
import type { OpsStep } from '@core/mock/data/opsTools'

// ============================================================
// 模板详情 — /ops/templates/:id
// 真实数据：useOpsTemplateById(id)；展示模板元信息 + 执行步骤列表
// ============================================================

const CATEGORY_VARIANT: Record<
  string,
  'default' | 'warning' | 'success' | 'destructive' | 'secondary'
> = {
  巡检运维: 'default',
  故障处置: 'destructive',
  性能优化: 'success',
  软件管理: 'secondary',
  网络配置: 'default',
  维护操作: 'warning',
}

const STEP_TYPE_LABEL: Record<OpsStep['stepType'], string> = {
  mml: 'MML',
  check: '检查',
  wait: '等待',
  notify: '通知',
  script: '脚本',
}

const STEP_TYPE_VARIANT: Record<
  OpsStep['stepType'],
  'default' | 'warning' | 'success' | 'secondary' | 'muted'
> = {
  mml: 'default',
  check: 'success',
  wait: 'warning',
  notify: 'secondary',
  script: 'muted',
}

function formatDuration(seconds: number): string {
  if (!seconds) return '—'
  if (seconds >= 60) return `${Math.floor(seconds / 60)} 分`
  return `${seconds} 秒`
}

function Field({
  label,
  children,
  full,
}: {
  label: string
  children: React.ReactNode
  full?: boolean
}) {
  return (
    <div className={cn(full && 'sm:col-span-2')}>
      <dt className="text-xs uppercase tracking-wider text-muted-foreground">{label}</dt>
      <dd className="mt-1">{children}</dd>
    </div>
  )
}

export default function TemplateDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: template, isLoading, isError, error, isFetching } = useOpsTemplateById(id)

  return (
    <PageShell
      title={template ? template.templateName : '模板详情'}
      description={id ? `模板 ID ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <Button variant="ghost" size="sm" onClick={() => navigate('/ops/templates')}>
          <ArrowLeft className="size-4" /> 返回模板库
        </Button>
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
          <span className="text-sm">未找到模板 {id}</span>
          <Button variant="outline" size="sm" onClick={() => navigate('/ops/templates')}>
            返回模板库
          </Button>
        </Card>
      ) : (
        <div className="flex flex-col gap-4">
          <Card className="p-5">
            <dl className="grid grid-cols-1 gap-x-8 gap-y-4 text-sm sm:grid-cols-2">
              <Field label="分类">
                <Badge variant={CATEGORY_VARIANT[template.category] ?? 'outline'}>
                  {template.category || '—'}
                </Badge>
              </Field>
              <Field label="使用次数">
                <span className="tabular-nums">{template.useCount}</span>
              </Field>
              <Field label="适用设备" full>
                <div className="flex flex-wrap gap-1">
                  {template.targetDeviceTypes.length > 0
                    ? template.targetDeviceTypes.map((d) => (
                        <Badge key={d} variant="muted">
                          {d}
                        </Badge>
                      ))
                    : '—'}
                </div>
              </Field>
              <Field label="预计耗时">{formatDuration(template.estimatedDuration)}</Field>
              <Field label="创建人">{template.creator}</Field>
              <Field label="创建时间">{formatTime(template.createTime)}</Field>
              <Field label="更新时间">{formatTime(template.updateTime)}</Field>
              {template.tags.length > 0 ? (
                <Field label="标签" full>
                  <div className="flex flex-wrap gap-1">
                    {template.tags.map((tag) => (
                      <Badge key={tag} variant="secondary">
                        {tag}
                      </Badge>
                    ))}
                  </div>
                </Field>
              ) : null}
              {template.description ? (
                <Field label="描述" full>
                  <span className="text-muted-foreground">{template.description}</span>
                </Field>
              ) : null}
            </dl>
          </Card>

          <Card className="p-5">
            <div className="mb-3 text-sm font-medium">执行步骤（{template.steps.length}）</div>
            {template.steps.length === 0 ? (
              <div className="rounded-md border bg-muted/30 p-4 text-center text-sm text-muted-foreground">
                暂无步骤
              </div>
            ) : (
              <ol className="space-y-3">
                {template.steps.map((step) => (
                  <li key={step.stepNo} className="relative rounded-md border bg-card p-3 pl-10">
                    <span className="absolute left-3 top-3 flex size-5 items-center justify-center rounded-full bg-primary/10 text-[11px] font-semibold text-primary tabular-nums">
                      {step.stepNo}
                    </span>
                    <div className="flex items-center gap-2">
                      <Badge variant={STEP_TYPE_VARIANT[step.stepType]}>
                        {STEP_TYPE_LABEL[step.stepType]}
                      </Badge>
                      <span className="text-sm font-medium">{step.stepName}</span>
                    </div>
                    {step.description ? (
                      <div className="mt-1 text-xs text-muted-foreground">{step.description}</div>
                    ) : null}
                    {step.command ? (
                      <div className="mt-2 rounded bg-muted/50 px-2 py-1 font-mono text-xs">
                        {step.command}
                      </div>
                    ) : null}
                    {step.condition ? (
                      <div className="mt-1 text-xs text-primary">条件: {step.condition}</div>
                    ) : null}
                    {step.waitSeconds ? (
                      <div className="mt-1 text-xs text-amber-600 dark:text-amber-400">
                        等待 {step.waitSeconds} 秒
                      </div>
                    ) : null}
                    {step.notifyTarget ? (
                      <div className="mt-1 text-xs text-muted-foreground">
                        通知对象: {step.notifyTarget}
                      </div>
                    ) : null}
                    {step.rollbackCommand ? (
                      <div className="mt-1 text-xs text-destructive">
                        回退: {step.rollbackCommand}
                      </div>
                    ) : null}
                  </li>
                ))}
              </ol>
            )}
          </Card>
        </div>
      )}
    </PageShell>
  )
}
