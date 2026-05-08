import { Empty } from 'antd';
import { useT } from '@/hooks/useT';

// T-0098-P4-02 stub：P4-05 子任务实现完整 UI（5 Tabs + 详情抽屉 + 全平台公式 CRUD）
export default function KpiLibraryPage() {
  const t = useT();
  return (
    <div style={{ padding: 24, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Empty description={t('page.product.kpiLibrary.placeholder')} />
    </div>
  );
}
