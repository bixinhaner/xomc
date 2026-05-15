import { Empty, Alert } from 'antd';
import { useT } from '@/hooks/useT';

export default function XmlImportTab() {
  const t = useT();
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 12 }}>
      <Alert
        type="warning"
        showIcon
        message={t('mml.admin.catalog.xmlImport.uploadHint')}
        description={t('mml.admin.catalog.xmlImport.pending')}
      />
      <Empty description={false} style={{ padding: 48 }} />
    </div>
  );
}
