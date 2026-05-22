import { useCallback, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Col,
  Divider,
  Progress,
  Row,
  Select,
  Space,
  Tag,
  Typography,
  Upload,
  message,
} from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import {
  CheckCircleOutlined,
  CloudDownloadOutlined,
  CloudUploadOutlined,
  DownloadOutlined,
  FileExcelOutlined,
  InboxOutlined,
} from '@ant-design/icons';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;
const { Text, Title } = Typography;

interface ImportRecord {
  fileName: string;
  status: 'success' | 'error' | 'processing';
  total: number;
  success: number;
  failed: number;
  time: string;
}

const MOCK_IMPORT_HISTORY: ImportRecord[] = [
  { fileName: 'devices_batch_20240301.xlsx', status: 'success', total: 50, success: 50, failed: 0, time: '2024-03-01 10:30:00' },
  { fileName: 'devices_batch_20240228.xlsx', status: 'error', total: 30, success: 27, failed: 3, time: '2024-02-28 15:00:00' },
  { fileName: 'devices_import_20240225.xlsx', status: 'success', total: 100, success: 100, failed: 0, time: '2024-02-25 09:00:00' },
];

export default function ImportExport() {
  const t = useT();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [importing, setImporting] = useState(false);
  const [importProgress, setImportProgress] = useState(0);
  const [importDone, setImportDone] = useState(false);
  const [exportFormat, setExportFormat] = useState<'xlsx' | 'csv'>('xlsx');
  const [exporting, setExporting] = useState(false);
  const [importHistory, setImportHistory] = useState<ImportRecord[]>(MOCK_IMPORT_HISTORY);

  const EXPORT_FILTER_FIELDS: FilterField[] = [
    {
      name: 'vendor',
      label: t('device.vendor'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: '华为', value: '华为' },
        { label: '中兴', value: '中兴' },
        { label: '爱立信', value: '爱立信' },
      ],
    },
    {
      name: 'productClass',
      label: t('device.productClass'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: 'eNB', value: 'eNB' },
        { label: 'gNB', value: 'gNB' },
        { label: 'CPE', value: 'CPE' },
        { label: 'eGW', value: 'eGW' },
      ],
    },
    {
      name: 'connStatus',
      label: t('device.connStatus'),
      type: 'select',
      options: [
        { label: t('common.all'), value: '' },
        { label: t('status.online'), value: 'online' },
        { label: t('status.offline'), value: 'offline' },
      ],
    },
    { name: 'timeRange', label: t('table.createTime'), type: 'date-range' },
  ];

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    accept: '.xlsx,.xls,.csv',
    fileList,
    beforeUpload: (file) => {
      const isValidType =
        file.type === 'application/vnd.openxmlformats-officedocument.spreadsheetml.sheet' ||
        file.type === 'application/vnd.ms-excel' ||
        file.name.endsWith('.csv');
      if (!isValidType) {
        void message.error('.xlsx, .xls, .csv');
        return false;
      }
      const isLt10M = file.size / 1024 / 1024 < 10;
      if (!isLt10M) {
        void message.error('10MB');
        return false;
      }
      setFileList([file]);
      return false; // prevent auto-upload
    },
    onRemove: () => {
      setFileList([]);
      setImportDone(false);
    },
  };

  const handleImport = useCallback(async () => {
    if (fileList.length === 0) {
      void message.warning(t('common.pleaseSelect'));
      return;
    }
    setImporting(true);
    setImportProgress(0);
    // Simulate import progress
    for (let p = 0; p <= 100; p += 10) {
      await new Promise<void>((resolve) => setTimeout(resolve, 150));
      setImportProgress(p);
    }
    setImporting(false);
    setImportDone(true);
    const newRecord: ImportRecord = {
      fileName: fileList[0].name ?? 'unknown.xlsx',
      status: 'success',
      total: 45,
      success: 45,
      failed: 0,
      time: new Date().toLocaleString('zh-CN'),
    };
    setImportHistory((prev) => [newRecord, ...prev]);
    void message.success(t('status.success'));
  }, [fileList, t]);

  const handleExport = useCallback(async () => {
    setExporting(true);
    await new Promise<void>((resolve) => setTimeout(resolve, 1500));
    setExporting(false);
    void message.success(t('common.exportInProgress'));
    // In a real app, trigger file download here
  }, [t]);

  const handleDownloadTemplate = useCallback(() => {
    void message.info(t('common.download'));
    // In a real app, trigger template download here
  }, [t]);

  return (
    <ListPageLayout title={t('nav.device.import')}>
      <Row gutter={[16, 16]}>
        {/* Import Section */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <CloudUploadOutlined style={{ color: 'var(--color-primary-600)' }} />
                <span>{t('common.import')}</span>
              </Space>
            }
            size="small"
          >
            <Alert
              type="info"
              showIcon
              message={t('common.import')}
              description={
                <ul style={{ paddingLeft: 20, margin: '4px 0' }}>
                  <li>.xlsx / .xls / .csv, max 10MB</li>
                  <li>max 1000 records</li>
                  <li>SN required, duplicate SN skipped</li>
                </ul>
              }
              style={{ marginBottom: 16 }}
            />

            <div style={{ marginBottom: 12 }}>
              <Button
                icon={<DownloadOutlined />}
                size="small"
                onClick={handleDownloadTemplate}
              >
                {t('common.download')}
              </Button>
            </div>

            <Dragger {...uploadProps} style={{ marginBottom: 16 }}>
              <p className="ant-upload-drag-icon">
                <InboxOutlined style={{ fontSize: 40, color: 'var(--color-primary-600)' }} />
              </p>
              <p className="ant-upload-text">{t('common.upload')}</p>
              <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>
                .xlsx, .xls, .csv
              </p>
            </Dragger>

            {importing && (
              <div style={{ marginBottom: 16 }}>
                <Text type="secondary" style={{ fontSize: 13 }}>
                  {t('common.loading')}
                </Text>
                <Progress percent={importProgress} status="active" />
              </div>
            )}

            {importDone && (
              <Alert
                type="success"
                showIcon
                icon={<CheckCircleOutlined />}
                message={t('status.success')}
                style={{ marginBottom: 16 }}
              />
            )}

            <Button
              type="primary"
              icon={<CloudUploadOutlined />}
              loading={importing}
              onClick={() => void handleImport()}
              disabled={fileList.length === 0}
              block
            >
              {t('common.import')}
            </Button>

            <Divider style={{ margin: '16px 0' }} />

            <Title level={5} style={{ fontSize: 14, marginBottom: 12 }}>
              {t('table.time')}
            </Title>
            {importHistory.map((record, idx) => (
              <div
                key={idx}
                style={{
                  display: 'flex',
                  alignItems: 'center',
                  justifyContent: 'space-between',
                  padding: '6px 0',
                  borderBottom: idx < importHistory.length - 1 ? '1px solid #f0f0f0' : 'none',
                }}
              >
                <div>
                  <FileExcelOutlined style={{ color: '#52C41A', marginRight: 6 }} />
                  <Text style={{ fontSize: 12 }}>{record.fileName}</Text>
                  <Text type="secondary" style={{ fontSize: 11, marginLeft: 8 }}>
                    {record.time}
                  </Text>
                </div>
                <div style={{ display: 'flex', gap: 6, alignItems: 'center' }}>
                  <Tag color={record.status === 'success' ? 'success' : 'error'} style={{ margin: 0 }}>
                    {record.status === 'success' ? t('status.success') : t('status.failed')}
                  </Tag>
                  <Text style={{ fontSize: 11, color: '#8c8c8c' }}>
                    {record.success}/{record.total}
                  </Text>
                </div>
              </div>
            ))}
          </Card>
        </Col>

        {/* Export Section */}
        <Col xs={24} lg={12}>
          <Card
            title={
              <Space>
                <CloudDownloadOutlined style={{ color: '#52C41A' }} />
                <span>{t('common.export')}</span>
              </Space>
            }
            size="small"
          >
            <Alert
              type="info"
              showIcon
              message={t('common.export')}
              description={
                <ul style={{ paddingLeft: 20, margin: '4px 0' }}>
                  <li>Excel (.xlsx) / CSV (.csv)</li>
                  <li>max 10000 records</li>
                </ul>
              }
              style={{ marginBottom: 16 }}
            />

            <div style={{ marginBottom: 16 }}>
              <Text style={{ fontSize: 13, display: 'block', marginBottom: 8 }}>
                {t('alarm.filter')}
              </Text>
              <FilterBar
                filterId="export-filter"
                fields={EXPORT_FILTER_FIELDS}
                onSearch={() => { /* no-op, filter is applied on export */ }}
                onReset={() => { /* no-op */ }}
                collapsedRows={1}
              />
            </div>

            <div style={{ marginBottom: 16 }}>
              <Text style={{ fontSize: 13, display: 'block', marginBottom: 8 }}>
                {t('table.type')}
              </Text>
              <Select
                value={exportFormat}
                onChange={(v) => setExportFormat(v)}
                style={{ width: 200 }}
                options={[
                  {
                    label: (
                      <Space>
                        <FileExcelOutlined style={{ color: '#52C41A' }} />
                        Excel (.xlsx)
                      </Space>
                    ),
                    value: 'xlsx',
                  },
                  { label: 'CSV (.csv)', value: 'csv' },
                ]}
              />
            </div>

            <Button
              type="primary"
              icon={<DownloadOutlined />}
              loading={exporting}
              onClick={() => void handleExport()}
              style={{ background: '#52C41A', borderColor: '#52C41A' }}
              block
            >
              {exporting ? t('common.loading') : t('common.export')}
            </Button>

            <Divider style={{ margin: '16px 0' }} />

            <Title level={5} style={{ fontSize: 14, marginBottom: 12 }}>
              {t('table.description')}
            </Title>
            <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
              {[
                { title: t('common.add'), desc: t('common.import') },
                { title: t('common.edit'), desc: t('common.batchConfig') },
                { title: t('common.export'), desc: t('common.batchExport') },
                { title: t('common.deploy'), desc: t('common.import') },
              ].map(({ title, desc }, idx) => (
                <div
                  key={idx}
                  style={{
                    padding: '10px 12px',
                    background: '#fafafa',
                    borderRadius: 6,
                    border: '1px solid #f0f0f0',
                  }}
                >
                  <Text strong style={{ fontSize: 13, display: 'block' }}>
                    {title}
                  </Text>
                  <Text type="secondary" style={{ fontSize: 12 }}>
                    {desc}
                  </Text>
                </div>
              ))}
            </div>
          </Card>
        </Col>
      </Row>
    </ListPageLayout>
  );
}
