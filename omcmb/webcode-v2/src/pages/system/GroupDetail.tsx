import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { PageShell, formatTime } from '@/components/layout/PageShell'

import { useGroupById } from '@core/hooks/api/useSystem'

// ============================================================
// 系统管理 / 用户组详情 — :id 带参。真实数据 useGroupById（adminApi.getGroupById）
// ============================================================

function Field({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="border-b px-4 py-3 last:border-b-0">
      <div className="text-xs uppercase tracking-wider text-muted-foreground">
        {label}
      </div>
      <div className="mt-1 text-sm">{children}</div>
    </div>
  )
}

export default function GroupDetail() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data: group, isLoading, isError, error } = useGroupById(id)

  return (
    <PageShell
      title="用户组详情"
      description={group ? group.groupName : id}
      toolbar={
        <Button variant="outline" size="sm" onClick={() => navigate('/system/groups')}>
          <ArrowLeft className="size-4" /> 返回用户组列表
        </Button>
      }
    >
      {isLoading ? (
        <div className="flex h-40 items-center justify-center text-muted-foreground">
          <Loader2 className="size-5 animate-spin" />
        </div>
      ) : isError ? (
        <div className="rounded-lg border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          加载失败：{error instanceof Error ? error.message : '未知错误'}
        </div>
      ) : !group ? (
        <div className="rounded-lg border bg-card px-4 py-3 text-sm text-muted-foreground">
          未找到该用户组
        </div>
      ) : (
        <div className="grid gap-4 md:grid-cols-2">
          <div className="overflow-hidden rounded-lg border bg-card">
            <Field label="用户组">{group.groupName}</Field>
            <Field label="用户数">
              <span className="tabular-nums">{group.userCount}</span>
            </Field>
            <Field label="角色数">
              <span className="tabular-nums">{group.roleCount}</span>
            </Field>
          </div>
          <div className="overflow-hidden rounded-lg border bg-card">
            <Field label="内置">
              {group.builtIn ? (
                <Badge variant="secondary">内置</Badge>
              ) : (
                <span className="text-muted-foreground">—</span>
              )}
            </Field>
            <Field label="更新人">{group.updUser || '—'}</Field>
            <Field label="更新时间">{formatTime(group.updTime)}</Field>
          </div>
          <div className="overflow-hidden rounded-lg border bg-card md:col-span-2">
            <Field label="描述">{group.description || '—'}</Field>
          </div>
        </div>
      )}
    </PageShell>
  )
}
