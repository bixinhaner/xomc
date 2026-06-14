import { useNavigate } from 'react-router-dom'
import { ArrowLeft } from 'lucide-react'

import { Button } from '@/components/ui/button'
import { PageShell } from '@/components/layout/PageShell'

import TemplateForm from './TemplateForm'

// ============================================================
// 通知中心 → 新建通知模板（/notifications/templates/new）
// 对照 v1 TemplateForm 的「新建」态。整页表单，提交后回模板列表。
// ============================================================

export default function NotificationTemplateCreate() {
  const navigate = useNavigate()

  return (
    <PageShell
      title="新建通知模板"
      description="创建邮件 / 短信 / Webhook 通知模板"
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
      <TemplateForm />
    </PageShell>
  )
}
