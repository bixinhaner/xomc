import { StubPage } from '@/pages/_stub'

export function MRPage() {
  return (
    <StubPage
      code="F05"
      title="MR · 测量报告"
      subtitle="MRO / MRS / MRE FILE INTAKE"
      plan={[
        '文件采集流：每分钟入库速率 + 解析失败率 sparkline',
        'RSRP / RSRQ / SINR 频谱热力图（一日 24h × 100 设备）',
        'MRE 事件密度地球散点 — 与 BRIDGE 共用 HoloGlobe',
        '原始文件下载与重放（带 sample 抽样）',
      ]}
    />
  )
}
