import { Download } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'

// ============================================================
// 运维下载 — 对齐 v1 webcode/src/pages/ops/Downloads
// v1 即为占位页（设备按需文件采集尚未落地，路线图 W3 / T-0105）
// ============================================================

export default function Downloads() {
  return (
    <PageShell title="运维下载" description="设备配置 / 日志 / 诊断包按需采集（路线图 W3）">
      <Card className="flex flex-col items-center justify-center gap-4 px-6 py-16 text-center">
        <div className="flex size-16 items-center justify-center rounded-full bg-primary/10">
          <Download className="size-8 text-primary" />
        </div>
        <h2 className="text-lg font-semibold">运维下载子系统实施中</h2>
        <p className="max-w-xl text-sm text-muted-foreground">
          设备 → OMC 的按需文件采集（配置 / 日志 / PM / MR / 诊断包 / PCAP / GPS 历史）尚未落地。后端依赖 transfer 模块 Upload 通道 + MinIO presigned URL，参需求文档{' '}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">
            docs/project/prd/F06-ops-management.md §4.5
          </code>
          。
        </p>
        <div className="flex items-center gap-2">
          <Badge variant="default">路线图 W3</Badge>
          <Badge variant="muted">T-0105</Badge>
          <Badge variant="warning">~10 工作日</Badge>
        </div>
      </Card>
    </PageShell>
  )
}
