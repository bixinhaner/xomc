import { Empty } from 'antd';
import { useT } from '@/hooks/useT';

// T-0098-P4-02 stub：P4-06 子任务实现完整 UI（单页 + 详情抽屉 + 未识别频次 Modal）
export default function AlarmLibraryPage() {
  const t = useT();
  return (
    <div style={{ padding: 24, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Empty description={t('page.product.alarmLibrary.placeholder')} />
    </div>
  );
}
