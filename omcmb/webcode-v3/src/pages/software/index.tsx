import { StubPage } from '@/pages/_stub'

export function SoftwarePage() {
  return (
    <StubPage
      code="F06"
      title="SOFTWARE · 软件武库"
      subtitle="FIRMWARE REGISTRY · OTA / ROLLBACK"
      plan={[
        '版本网格视图（厂商 × 制式 × 版本号）— 用六边形磁贴呈现',
        '灰度升级编排：选择版本 → 选择目标分组 → 设置批次比例与回滚阈值',
        '正在执行的 OTA 任务卡片：进度条 + 设备成功/失败统计',
        '版本元数据：哈希、大小、签名、上线时间、CHANGELOG',
      ]}
    />
  )
}
