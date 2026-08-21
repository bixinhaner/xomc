import { useCallback, useMemo, useState } from 'react';
import { Alert, App, Button, Flex, Modal, Tag, theme, Typography, Upload } from 'antd';
import type { UploadFile, UploadProps } from 'antd';
import {
  CheckCircleOutlined,
  DownloadOutlined,
  InboxOutlined,
  WarningOutlined,
} from '@ant-design/icons';
import { deviceApi } from '@core/services/api/deviceApi';
import type { BatchPreRegisterDevice, BatchPreRegisterResponse, BatchPreRegisterRowResult } from '@core/types/device';

const { Dragger } = Upload;
const { Text, Paragraph } = Typography;

// CSV 列名 → snake_case 字段映射（支持中英文表头）
const HEADER_ALIAS: Record<string, string> = {
  // 英文
  'SN': 'serial_number', 'serial_number': 'serial_number',
  'Device Name': 'device_name', 'device_name': 'device_name',
  'Remark': 'remark', 'remark': 'remark',
  'ProductClass': 'product_class', 'Product Class': 'product_class', 'product_class': 'product_class',
  'Carrier': 'carrier', 'carrier': 'carrier',
  'Technology': 'technology', 'technology': 'technology',
  'OUI': 'oui', 'oui': 'oui',
  // 中文
  '设备名称': 'device_name', '产品类型': 'product_class', '备注': 'remark',
  '运营商': 'carrier', '制式': 'technology',
};

const CARRIER_VALUES = new Set(['cmcc', 'ctcc', 'cucc']);
const TECH_VALUES = new Set(['lte', 'nr']);

function parsePreRegCsv(
  text: string,
  t: (id: string, values?: Record<string, string | number>) => string,
): { ok: true; data: BatchPreRegisterDevice[] } | { ok: false; reason: string } {
  const cleaned = text.charCodeAt(0) === 0xfeff ? text.slice(1) : text;
  const lines = cleaned
    .split(/\r?\n/).map(l => l.trim())
    .filter(l => l.length > 0 && !l.startsWith('#'));

  if (lines.length <= 1) {
    return { ok: false, reason: t('device.batchPreRegister.noDataRows') };
  }

  const rawHeader = lines[0].split(',').map(h => h.trim());
  const header = rawHeader.map(h => HEADER_ALIAS[h] ?? h);

  if (!header.includes('serial_number')) {
    return { ok: false, reason: t('device.batchPreRegister.missingHeader', { field: 'SN' }) };
  }

  const colIdx: Record<string, number> = {};
  header.forEach((col, i) => { colIdx[col] = i; });

  const devices: BatchPreRegisterDevice[] = [];
  for (let i = 1; i < lines.length; i++) {
    const cells = lines[i].split(',').map(c => c.trim());
    const get = (name: string) => {
      const idx = colIdx[name];
      return idx !== undefined && cells[idx] !== '' ? cells[idx] : undefined;
    };

    const sn = get('serial_number');
    if (!sn) continue;

    const rawCarrier = get('carrier');
    const rawTech = get('technology');

    devices.push({
      serial_number: sn,
      device_name: get('device_name'),
      remark: get('remark'),
      product_class: get('product_class'),
      carrier: rawCarrier && CARRIER_VALUES.has(rawCarrier) ? rawCarrier as 'cmcc' | 'ctcc' | 'cucc' : undefined,
      technology: rawTech && TECH_VALUES.has(rawTech) ? rawTech as 'lte' | 'nr' : undefined,
      oui: get('oui'),
    });
  }

  return { ok: true, data: devices };
}

export interface BatchPreRegisterModalProps {
  open: boolean;
  onClose: () => void;
  onPreRegister: (result: BatchPreRegisterResponse) => void | Promise<void>;
  onDownloadTemplate: () => void;
  t: (id: string, values?: Record<string, string | number>) => string;
}

const IMPORT_CHUNK_SIZE = 500;

export default function BatchPreRegisterModal({
  open,
  onClose,
  onPreRegister,
  onDownloadTemplate,
  t,
}: BatchPreRegisterModalProps) {
  const { message } = App.useApp();
  const { token } = theme.useToken();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [parsed, setParsed] = useState<BatchPreRegisterDevice[] | null>(null);
  const [importing, setImporting] = useState(false);
  const [result, setResult] = useState<BatchPreRegisterResponse | null>(null);

  const resetAll = useCallback(() => {
    setFileList([]);
    setParsed(null);
    setResult(null);
  }, []);

  const uploadProps: UploadProps = {
    name: 'file',
    multiple: false,
    accept: '.csv',
    fileList,
    beforeUpload: (file) => {
      if (!file.name.toLowerCase().endsWith('.csv')) {
        void message.error(t('device.fileFormatError'));
        return false;
      }
      if (file.size / 1024 / 1024 >= 10) {
        void message.error(t('device.fileSizeError'));
        return false;
      }
      const reader = new FileReader();
      reader.onload = (e) => {
        const text = (e.target?.result as string) ?? '';
        const r = parsePreRegCsv(text, t);
        if (!r.ok) {
          void message.error(t('device.batchPreRegister.parsingFailed', { reason: r.reason }));
          setFileList([]);
          setParsed(null);
          return;
        }
        setFileList([file]);
        setParsed(r.data);
        setResult(null);
      };
      reader.onerror = () => {
        void message.error(t('device.batchPreRegister.parsingFailed', { reason: 'read file failed' }));
      };
      reader.readAsText(file, 'utf-8');
      return false;
    },
    onRemove: () => resetAll(),
  };

  const errors = useMemo<BatchPreRegisterRowResult[]>(() => result?.errors ?? [], [result]);
  const visibleErrors = useMemo(() => errors.slice(0, 10), [errors]);
  const hiddenErrorCount = Math.max(0, errors.length - visibleErrors.length);

  const handleConfirm = useCallback(async () => {
    if (!parsed || parsed.length === 0) {
      void message.warning(t('device.selectFileFirst'));
      return;
    }
    setImporting(true);
    try {
      let total = 0, created = 0, updated = 0, failed = 0;
      const allErrors: BatchPreRegisterRowResult[] = [];
      let offset = 0;
      for (let i = 0; i < parsed.length; i += IMPORT_CHUNK_SIZE) {
        const chunk = parsed.slice(i, i + IMPORT_CHUNK_SIZE);
        const resp = await deviceApi.batchPreRegisterDevices({ devices: chunk });
        total += resp.total;
        created += resp.created;
        updated += resp.updated;
        failed += resp.failed;
        for (const e of resp.errors ?? []) {
          allErrors.push({ ...e, row: e.row > 0 ? e.row + offset : e.row });
        }
        offset += chunk.length;
      }
      const finalResult: BatchPreRegisterResponse = { total, created, updated, failed, errors: allErrors };
      setResult(finalResult);
      await onPreRegister(finalResult);
    } catch (err) {
      void message.error(err instanceof Error ? err.message : String(err));
    } finally {
      setImporting(false);
    }
  }, [parsed, onPreRegister, t, message]);

  const handleCancel = useCallback(() => {
    if (importing) return;
    resetAll();
    onClose();
  }, [importing, onClose, resetAll]);

  const resultSummary = result ? (
    <Alert
      style={{ marginBottom: 12 }}
      type={result.failed === 0 ? 'success' : result.created + result.updated === 0 ? 'error' : 'warning'}
      showIcon
      icon={result.failed === 0 ? <CheckCircleOutlined /> : <WarningOutlined />}
      message={
        result.failed === 0
          ? t('device.batchPreRegister.allSucceeded', { created: result.created, updated: result.updated })
          : result.created + result.updated === 0
            ? t('device.batchPreRegister.allFailed', { failed: result.failed })
            : t('device.batchPreRegister.partial', { created: result.created, updated: result.updated, failed: result.failed })
      }
    />
  ) : null;

  const parseSummary = parsed && !result ? (
    <Alert
      style={{ marginBottom: 12 }}
      type="info"
      showIcon
      message={
        <Text type="secondary" style={{ fontSize: 13 }}>
          {t('device.batchPreRegister.readyHint', { count: parsed.length })}
        </Text>
      }
    />
  ) : null;

  const errorList = visibleErrors.length > 0 ? (
    <div style={{ marginBottom: 12 }}>
      <Paragraph style={{ marginBottom: 6, fontSize: 13 }} strong>
        {t('device.batchPreRegister.errorListTitle')}
      </Paragraph>
      <Flex vertical style={{ border: `1px solid ${token.colorBorder}`, borderRadius: token.borderRadiusLG }}>
        {visibleErrors.map((item, idx) => (
          <div
            key={`${item.row}-${item.sn}-${idx}`}
            style={{
              fontSize: 12,
              padding: `${token.paddingContentVerticalSM}px ${token.paddingContentHorizontal}px`,
              borderBottom: idx === visibleErrors.length - 1 ? 'none' : `1px solid ${token.colorSplit}`,
            }}
          >
            <Text type="danger" style={{ marginRight: 8 }}>#{item.row}</Text>
            {item.sn ? <Tag style={{ marginRight: 8 }}>{item.sn}</Tag> : null}
            <Text>
              {item.error_code
                ? t(`device.batchPreRegister.error.${item.error_code}`)
                : item.reason}
            </Text>
          </div>
        ))}
      </Flex>
      {hiddenErrorCount > 0 ? (
        <Text type="secondary" style={{ fontSize: 12, marginTop: 4, display: 'block' }}>
          {t('device.batchPreRegister.errorListMore', { n: hiddenErrorCount })}
        </Text>
      ) : null}
    </div>
  ) : null;

  return (
    <Modal
      title={t('device.batchPreRegister')}
      open={open}
      onCancel={handleCancel}
      footer={null}
      width={600}
      destroyOnHidden
    >
      <div style={{ marginBottom: 12 }}>
        <Button icon={<DownloadOutlined />} size="small" onClick={onDownloadTemplate}>
          {t('device.downloadPreRegisterTemplate')}
        </Button>
      </div>

      {resultSummary}
      {parseSummary}
      {errorList}

      {!result && (
        <Dragger {...uploadProps} style={{ marginBottom: 12 }}>
          <p className="ant-upload-drag-icon"><InboxOutlined /></p>
          <p className="ant-upload-text">{t('common.upload')}</p>
          <p className="ant-upload-hint" style={{ fontSize: 12, color: '#8c8c8c' }}>.csv</p>
        </Dragger>
      )}

      <Flex justify="flex-end" gap={8} style={{ marginTop: 8 }}>
        <Button onClick={handleCancel} disabled={importing}>{t('common.cancel')}</Button>
        {result ? (
          <Button onClick={() => { resetAll(); }} >{t('device.batchImport.reselectFile')}</Button>
        ) : (
          <Button
            type="primary"
            onClick={handleConfirm}
            loading={importing}
            disabled={!parsed || parsed.length === 0}
          >
            {t('device.batchPreRegister')}
          </Button>
        )}
      </Flex>
    </Modal>
  );
}
