import { Empty } from 'antd';
import { useT } from '@/hooks/useT';

// T-0098-P4-02 stub：P4-07 子任务实现完整 UI（列表 + 单台/批量绑定 + 触发重新匹配）
export default function OrphanDevicesPage() {
  const t = useT();
  return (
    <div style={{ padding: 24, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Empty description={t('page.product.orphanDevices.placeholder')} />
    </div>
  );
}
