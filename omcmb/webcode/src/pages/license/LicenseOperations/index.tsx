import { useState, useMemo } from 'react';
import { Button, Card, Upload, Input, Tag, Space, message, Tabs, Form, Modal } from 'antd';
import { InboxOutlined, CheckCircleOutlined, StopOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadFile, RcFile } from 'antd/es/upload';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import { useActivateLicense, useRevokeLicense, useImportLicense } from '@/hooks/api/useLicense';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;

type OperationType = 'import' | 'activate' | 'revoke' | 'query';
type OperationResult = 'success' | 'failed' | 'pending';

interface LicenseOperationRecord {
  id: string;
  operationTime: string;
  operationType: OperationType;
  licenseId: string;
  operator: string;
  result: OperationResult;
  remark?: string;
}

const mockOperationHistory: LicenseOperationRecord[] = [
  { id: 'lo-001', operationTime: '2024-06-01T09:00:00.000Z', operationType: 'import', licenseId: 'OMC-BASIC-HB-2024-001', operator: 'admin', result: 'success', remark: '导入基础许可证' },
  { id: 'lo-002', operationTime: '2024-06-01T09:05:00.000Z', operationType: 'activate', licenseId: 'OMC-BASIC-HB-2024-001', operator: 'admin', result: 'success' },
  { id: 'lo-003', operationTime: '2024-05-15T14:00:00.000Z', operationType: 'import', licenseId: 'OMC-ADV-HD-2024-001', operator: 'admin', result: 'success', remark: '导入高级许可证' },
  { id: 'lo-004', operationTime: '2024-05-15T14:05:00.000Z', operationType: 'activate', licenseId: 'OMC-ADV-HD-2024-001', operator: 'admin', result: 'success' },
  { id: 'lo-005', operationTime: '2024-04-01T10:00:00.000Z', operationType: 'import', licenseId: 'OMC-TRIAL-2024-001', operator: 'operator1', result: 'failed', remark: 'License文件格式错误' },
  { id: 'lo-006', operationTime: '2024-03-01T16:00:00.000Z', operationType: 'revoke', licenseId: 'OMC-OLD-2023-001', operator: 'admin', result: 'success', remark: '旧版许可证到期撤销' },
];

const opTypeColorMap: Record<OperationType, string> = {
  import: 'blue',
  activate: 'green',
  revoke: 'orange',
  query: 'default',
};

const resultColorMap: Record<OperationResult, string> = {
  success: 'green',
  failed: 'red',
  pending: 'processing',
};

export default function LicenseOperations() {
  const t = useT();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [activateCode, setActivateCode] = useState('');
  const [revokeId, setRevokeId] = useState('');
  const [history, setHistory] = useState<LicenseOperationRecord[]>(mockOperationHistory);
  const [importForm] = Form.useForm();
  const [importLoading, setImportLoading] = useState(false);

  const activateLicense = useActivateLicense();
  const revokeLicense = useRevokeLicense();
  const importLicense = useImportLicense();

  const opTypeLabelMap: Record<OperationType, string> = useMemo(() => ({
    import: t('common.import'),
    activate: t('license.activate'),
    revoke: t('license.revoke'),
    query: t('license.query'),
  }), [t]);

  const resultLabelMap: Record<OperationResult, string> = useMemo(() => ({
    success: t('status.success'),
    failed: t('status.failed'),
    pending: t('status.pending'),
  }), [t]);

  const handleImport = () => {
    if (fileList.length === 0) { void message.warning(t('license.selectFileFirst')); return; }
    setImportLoading(true);
    setTimeout(() => {
      setImportLoading(false);
      const newRecord: LicenseOperationRecord = {
        id: `lo-${Date.now()}`,
        operationTime: new Date().toISOString(),
        operationType: 'import',
        licenseId: `OMC-IMPORT-${Date.now()}`,
        operator: 'admin',
        result: 'success',
        remark: `${t('license.importFile')}: ${fileList[0]?.name ?? ''}`,
      };
      setHistory((prev) => [newRecord, ...prev]);
      setFileList([]);
      void message.success(t('license.importSuccess'));
    }, 1500);
  };

  const handleActivate = () => {
    if (!activateCode.trim()) { void message.warning(t('license.enterActivationCode')); return; }
    activateLicense.mutate(activateCode, {
      onSuccess: () => {
        const newRecord: LicenseOperationRecord = {
          id: `lo-${Date.now()}`,
          operationTime: new Date().toISOString(),
          operationType: 'activate',
          licenseId: activateCode,
          operator: 'admin',
          result: 'success',
        };
        setHistory((prev) => [newRecord, ...prev]);
        setActivateCode('');
        void message.success(t('license.activateSuccess'));
      },
    });
  };

  const handleRevoke = () => {
    if (!revokeId.trim()) { void message.warning(t('license.enterRevokeId')); return; }
    Modal.confirm({
      title: t('license.confirmRevoke'),
      content: t('license.confirmRevokeMsg', { id: revokeId }),
      okType: 'danger',
      onOk: () => {
        revokeLicense.mutate(revokeId, {
          onSuccess: () => {
            const newRecord: LicenseOperationRecord = {
              id: `lo-${Date.now()}`,
              operationTime: new Date().toISOString(),
              operationType: 'revoke',
              licenseId: revokeId,
              operator: 'admin',
              result: 'success',
            };
            setHistory((prev) => [newRecord, ...prev]);
            setRevokeId('');
            void message.success(t('license.revokeSuccess'));
          },
        });
      },
    });
  };

  const historyColumns: DataTableColumn<LicenseOperationRecord & Record<string, unknown>>[] = useMemo(() => [
    {
      key: 'operationTime', title: t('table.time'), dataIndex: 'operationTime', width: 170,
      render: (val) => new Date(String(val)).toLocaleString('zh-CN'),
    },
    {
      key: 'operationType', title: t('license.operationType'), dataIndex: 'operationType', width: 100,
      render: (val) => {
        const tp = val as OperationType;
        return <Tag color={opTypeColorMap[tp]}>{opTypeLabelMap[tp]}</Tag>;
      },
    },
    {
      key: 'licenseId', title: 'License ID', dataIndex: 'licenseId', width: 220,
      render: (val) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{String(val)}</span>,
    },
    { key: 'operator', title: t('table.operator'), dataIndex: 'operator', width: 100 },
    {
      key: 'result', title: t('table.result'), dataIndex: 'result', width: 90,
      render: (val) => {
        const r = val as OperationResult;
        return <Tag color={resultColorMap[r]}>{resultLabelMap[r]}</Tag>;
      },
    },
    { key: 'remark', title: t('license.remark'), dataIndex: 'remark', ellipsis: true, render: (val) => val ? String(val) : '—' },
  ], [t, opTypeLabelMap, resultLabelMap]);

  return (
    <ListPageLayout title={t('nav.license.operations')} subtitle={t('license.operationsSubtitle')}>
      <Tabs
        items={[
          {
            key: 'import',
            label: t('license.importLicense'),
            children: (
              <Card>
                <div style={{ maxWidth: 600 }}>
                  <Dragger
                    fileList={fileList}
                    beforeUpload={(file: RcFile) => { setFileList([file]); return false; }}
                    onRemove={() => setFileList([])}
                    maxCount={1}
                    accept=".lic,.dat,.xml,.key"
                  >
                    <p className="ant-upload-drag-icon"><InboxOutlined /></p>
                    <p className="ant-upload-text">{t('license.dragFileHere')}</p>
                    <p className="ant-upload-hint">{t('license.supportedFormats')}</p>
                  </Dragger>
                  <Button
                    type="primary"
                    icon={<UploadOutlined />}
                    loading={importLoading}
                    onClick={handleImport}
                    style={{ marginTop: 16, width: '100%' }}
                  >
                    {t('license.importLicense')}
                  </Button>
                </div>
              </Card>
            ),
          },
          {
            key: 'activate',
            label: t('license.activateLicense'),
            children: (
              <Card>
                <div style={{ maxWidth: 500 }}>
                  <p style={{ color: '#666', marginBottom: 16 }}>{t('license.activateDescription')}</p>
                  <Input.TextArea
                    rows={4}
                    value={activateCode}
                    onChange={(e) => setActivateCode(e.target.value)}
                    placeholder={t('license.pasteActivationCode')}
                    style={{ fontFamily: 'monospace', marginBottom: 16 }}
                  />
                  <Button
                    type="primary"
                    icon={<CheckCircleOutlined />}
                    loading={activateLicense.isPending}
                    onClick={handleActivate}
                    block
                  >
                    {t('license.activateLicense')}
                  </Button>
                </div>
              </Card>
            ),
          },
          {
            key: 'revoke',
            label: t('license.revokeLicense'),
            children: (
              <Card>
                <div style={{ maxWidth: 500 }}>
                  <p style={{ color: '#ff4d4f', marginBottom: 16, fontWeight: 500 }}>
                    {t('license.revokeWarning')}
                  </p>
                  <Input
                    value={revokeId}
                    onChange={(e) => setRevokeId(e.target.value)}
                    placeholder={t('license.enterRevokeIdPlaceholder')}
                    style={{ marginBottom: 16, fontFamily: 'monospace' }}
                  />
                  <Button
                    danger
                    type="primary"
                    icon={<StopOutlined />}
                    loading={revokeLicense.isPending}
                    onClick={handleRevoke}
                    block
                  >
                    {t('license.revokeLicense')}
                  </Button>
                </div>
              </Card>
            ),
          },
          {
            key: 'history',
            label: t('license.operationHistory'),
            children: (
              <DataTable
                tableId="license-operations-history"
                columns={historyColumns}
                dataSource={history as (LicenseOperationRecord & Record<string, unknown>)[]}
                loading={false}
                rowKey="id"
                total={history.length}
                pageSize={20}
                currentPage={1}
                scroll={{ x: 900 }}
              />
            ),
          },
        ]}
      />
    </ListPageLayout>
  );
}
