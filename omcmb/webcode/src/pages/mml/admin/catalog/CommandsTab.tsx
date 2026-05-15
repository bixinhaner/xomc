import { Empty } from 'antd';
import { useT } from '@/hooks/useT';

export default function CommandsTab() {
  const t = useT();
  return (
    <Empty
      description={t('mml.admin.catalog.xmlImport.pending')}
      style={{ padding: 48 }}
    />
  );
}
