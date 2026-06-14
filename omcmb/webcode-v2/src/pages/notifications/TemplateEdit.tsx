import { useNavigate, useParams } from 'react-router-dom'
import { ArrowLeft, Loader2 } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { Card } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'

import { useNotificationTemplate } from '@core/hooks/api/useNotifications'

import TemplateForm from './TemplateForm'

// ============================================================
// 通知中心 → 编辑通知模板（/notifications/templates/:id）
// 对照 v1 TemplateForm 的「编辑」态。
// 真实加载：useNotificationTemplate(id) → GET /notifications/templates/:id。
// 加载成功后将模板带入表单，提交走 update mutation。
// ============================================================

export default function NotificationTemplateEdit() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const { data, isLoading, isError, error, isFetching } =
    useNotificationTemplate(id)

  return (
    <PageShell
      title={data ? `编辑模板 · ${data.name}` : '编辑通知模板'}
      description={id ? `模板 ${id}` : undefined}
      isFetching={isFetching}
      toolbar={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => navigate('/notifications/templates')}
        >
          <ArrowLeft className="size-4" /> 返回模板列表
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
      ) : !data ? (
        <Card className="flex h-48 flex-col items-center justify-center gap-2 p-6 text-muted-foreground">
          <span className="text-sm">未找到模板 {id}</span>
          <Button
            variant="outline"
            size="sm"
            onClick={() => navigate('/notifications/templates')}
          >
            返回模板列表
          </Button>
        </Card>
      ) : (
        <TemplateForm initial={data} />
      )}
    </PageShell>
  )
}
