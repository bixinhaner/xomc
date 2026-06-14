import { Radar } from 'lucide-react'

import { Badge } from '@/components/ui/badge'
import { Card } from '@/components/ui/card'
import { PageShell } from '@/components/layout/PageShell'

// ============================================================
// 网络诊断 — 对齐 v1 webcode/src/pages/ops/NetworkDiagnosis
// v1 即为占位页（TR-181 Diagnostics 对象接入尚未落地，路线图 W3 / T-0104）
// ============================================================

export default function NetworkDiagnosis() {
  return (
    <PageShell title="网络诊断" description="TR-181 Diagnostics 对象统一接入（路线图 W3）">
      <Card className="flex flex-col items-center justify-center gap-4 px-6 py-16 text-center">
        <div className="flex size-16 items-center justify-center rounded-full bg-primary/10">
          <Radar className="size-8 text-primary" />
        </div>
        <h2 className="text-lg font-semibold">网络诊断子系统实施中</h2>
        <p className="max-w-xl text-sm text-muted-foreground">
          后端 TR-181 Diagnostics 对象（IPPing / TraceRoute / Download / Upload / UDPEcho）接入尚未落地，参需求文档{' '}
          <code className="rounded bg-muted px-1 py-0.5 font-mono text-xs">
            docs/project/prd/F06-ops-management.md §4.4
          </code>
          。
        </p>
        <div className="flex items-center gap-2">
          <Badge variant="default">路线图 W3</Badge>
          <Badge variant="muted">T-0104</Badge>
          <Badge variant="warning">~10 工作日</Badge>
        </div>
      </Card>
    </PageShell>
  )
}
