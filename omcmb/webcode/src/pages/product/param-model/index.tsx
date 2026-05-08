import { Empty } from 'antd';
import { useT } from '@/hooks/useT';

// T-0098-P4-02 stub：P4-04 子任务实现完整 UI（3 Tabs + XML 导入 + storable 过滤 + {i} 校验）
export default function ParamModelPage() {
  const t = useT();
  return (
    <div style={{ padding: 24, height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      <Empty description={t('page.product.paramModel.placeholder')} />
    </div>
  );
}
