/**
 * ImportDrawer (T-0165) — 设备 license 批量导入。结构镜像
 * ConfigSnapshotLibrary/ImportDrawer：拖拽+点击混合上传 +
 * 命名校验 + 失败明细。文件名规范：<SN>.lic（兼容 .lic.json 等下载尾缀）。
 */
import { useMemo, useState } from 'react';
import { Alert, Button, Drawer, Empty, Space, Tag, Typography, Upload, message } from 'antd';
import {
  InboxOutlined, FileDoneOutlined, ExclamationCircleOutlined, DeleteOutlined,
} from '@ant-design/icons';
import { MetaList, MetaListItem } from '@/components/common/MetaListItem';
import { useT } from '@/hooks/useT';
import { useImportDeviceLicenses } from '@core/hooks/api/useDeviceLicense';
import {
  validateLicenseFileName,
  type LicenseImportResult,
} from '@core/services/api/deviceLicenseApi';

interface Props {
  open: boolean;
  onClose: () => void;
  onSuccess?: () => void;
}

interface ParsedFile {
  rawFile: File;
  fileName: string;
  serialNumber?: string;
  ext?: string;
  valid: boolean;
  reason?: string;
}

export default function ImportDrawer({ open, onClose, onSuccess }: Props) {
  const t = useT();
  const [files, setFiles] = useState<ParsedFile[]>([]);
  const [submitResult, setSubmitResult] = useState<LicenseImportResult | null>(null);
  const importMutation = useImportDeviceLicenses();

  const isFileOk = (f: ParsedFile) => f.valid;
  const isFileBad = (f: ParsedFile) => !f.valid;
  const validCount = useMemo(() => files.filter(isFileOk).length, [files]);
  const invalidCount = useMemo(() => files.filter(isFileBad).length, [files]);
  const canSubmit = validCount > 0 && invalidCount === 0;

  function handleAdd(file: File): boolean {
    setFiles((prev) => {
      if (prev.some((f) => f.fileName === file.name)) {
        void message.warning(t('transfer.fileLib.import.duplicate', { name: file.name }));
        return prev;
      }
      const v = validateLicenseFileName(file.name);
      return [
        ...prev,
        {
          rawFile: file,
          fileName: file.name,
          serialNumber: v.serialNumber,
          ext: v.ext,
          valid: v.valid,
          reason: v.message,
        },
      ];
    });
    return false;
  }

  function handleRemove(name: string) {
    setFiles((prev) => prev.filter((f) => f.fileName !== name));
  }

  function handleClearAll() {
    setFiles([]);
  }

  async function handleSubmit() {
    if (!canSubmit) return;
    try {
      const result = await importMutation.mutateAsync(files.map((f) => f.rawFile));
      setSubmitResult(result);
      if (result.succeeded.length > 0) {
        void message.success(t('transfer.fileLib.import.successMsg', { count: result.succeeded.length }));
        setFiles([]);
      }
      if (result.failed.length > 0) {
        void message.warning(t('transfer.fileLib.import.partialFailMsg', { count: result.failed.length }));
      }
      onSuccess?.();
    } catch {
      void message.error(t('transfer.fileLib.import.requestFailed'));
    }
  }

  function handleClose() {
    setFiles([]);
    setSubmitResult(null);
    onClose();
  }

  return (
    <Drawer
      open={open}
      title={t('transfer.fileLib.import.licenseTitle')}
      size={620}
      onClose={handleClose}
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={handleClose}>{t('transfer.fileLib.import.close')}</Button>
          <Button
            type="primary"
            disabled={!canSubmit}
            loading={importMutation.isPending}
            onClick={() => void handleSubmit()}
          >
            {t('transfer.fileLib.import.submit', { count: validCount })}
          </Button>
        </Space>
      }
    >
      <Alert
        type="info"
        showIcon
        message={t('transfer.fileLib.import.rulesTitle')}
        description={(
          <>
            ① {t('transfer.fileLib.import.licenseRule1')}
            <br />
            ② <strong>{t('transfer.fileLib.import.licenseRule2')}</strong>
            <br />
            ③ {t('transfer.fileLib.import.ruleSize')}
            <br />
            <strong>{t('transfer.fileLib.import.ruleSummary')}</strong>
          </>
        )}
        style={{ marginBottom: 12 }}
      />
      <div
        onDragOver={(e) => { e.preventDefault(); e.stopPropagation(); }}
        onDrop={(e) => {
          e.preventDefault();
          e.stopPropagation();
          const dropped = Array.from(e.dataTransfer.files);
          if (dropped.length === 0) return;
          dropped.forEach((f) => handleAdd(f));
        }}
      >
        {/* 不设 accept 后缀过滤：macOS 下 .lic 没有标准 UTI，加 accept=".lic"
            会把这类文件灰掉无法选中；改由 validateLicenseFileName JS 校验拦
            非法文件，行为更可靠。 */}
        <Upload.Dragger
          multiple
          beforeUpload={handleAdd}
          showUploadList={false}
          height={88}
          style={{ padding: 0 }}
        >
          <Space size={12} align="center" style={{ width: '100%', justifyContent: 'center' }}>
            <InboxOutlined style={{ fontSize: 22, color: '#1677ff' }} />
            <div style={{ textAlign: 'left' }}>
              <div style={{ fontSize: 13, color: 'rgba(0,0,0,0.88)' }}>
                {t('transfer.fileLib.import.licenseDropTitle')}
              </div>
              <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)' }}>
                {t('transfer.fileLib.import.licenseDropHint')}
              </div>
            </div>
          </Space>
        </Upload.Dragger>
      </div>

      <div style={{ marginTop: 16 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 8 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            {t('transfer.fileLib.import.pendingList', { count: files.length })}
            {invalidCount > 0 && <Tag color="red" style={{ marginLeft: 8 }}>{t('transfer.fileLib.import.tagInvalid', { count: invalidCount })}</Tag>}
            {validCount > 0 && <Tag color="green" style={{ marginLeft: 4 }}>{t('transfer.fileLib.import.tagValid', { count: validCount })}</Tag>}
          </Typography.Title>
          {files.length > 0 && (
            <Button type="link" danger size="small" onClick={handleClearAll}>
              {t('transfer.fileLib.import.clearAll')}
            </Button>
          )}
        </Space>

        {files.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description={t('transfer.fileLib.import.empty')}
          />
        ) : (
          // antd6 List/List.Item.Meta 已废弃：用 MetaList + MetaListItem 等价复刻
          // （avatar + title/description + 右侧 actions），视觉与原 size=small 列表一致。
          <div style={{ maxHeight: 420, overflowY: 'auto', border: '1px solid #f0f0f0', borderRadius: 6 }}>
            <MetaList style={{ background: '#fff' }}>
              {files.map((f, idx) => {
                const bad = isFileBad(f);
                return (
                  <MetaListItem
                    key={f.fileName}
                    last={idx === files.length - 1}
                    style={bad ? { background: '#fff1f0' } : undefined}
                    avatar={
                      bad ? <ExclamationCircleOutlined style={{ color: '#ff4d4f', fontSize: 18 }} />
                        : <FileDoneOutlined style={{ color: '#52c41a', fontSize: 18 }} />
                    }
                    title={(
                      <Space size={6} wrap>
                        <span style={{ fontFamily: 'monospace', color: bad ? '#ff4d4f' : undefined }}>
                          {f.fileName}
                        </span>
                        {f.valid && f.serialNumber && (
                          <Tag color="blue">
                            SN: {f.serialNumber}
                          </Tag>
                        )}
                        {f.valid && f.ext && <Tag color="geekblue">{f.ext.toUpperCase()}</Tag>}
                        {f.valid && <Tag color="green">{t('transfer.fileLib.import.readyForPreinstall')}</Tag>}
                      </Space>
                    )}
                    description={(
                      !f.valid ? (
                        <Typography.Text type="danger">{f.reason}</Typography.Text>
                      ) : (
                        <Typography.Text type="secondary">
                          {(f.rawFile.size / 1024).toFixed(1)} KB
                        </Typography.Text>
                      )
                    )}
                    actions={[
                      <Button
                        key="rm"
                        type="link"
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onClick={() => handleRemove(f.fileName)}
                      >
                        {t('transfer.fileLib.import.remove')}
                      </Button>,
                    ]}
                  />
                );
              })}
            </MetaList>
          </div>
        )}
      </div>

      {submitResult && (
        <div style={{ marginTop: 16 }}>
          <Alert
            type={submitResult.failed.length === 0 ? 'success' : 'warning'}
            showIcon
            message={t('transfer.fileLib.import.resultSummary', { succeeded: submitResult.succeeded.length, failed: submitResult.failed.length })}
          />
          {submitResult.failed.length > 0 && (
            // antd6 List 已废弃：MetaList(bordered+header) + MetaListItem 等价复刻。
            <MetaList
              bordered
              header={<Typography.Text strong>{t('transfer.fileLib.import.failureDetails')}</Typography.Text>}
              style={{ marginTop: 8, maxHeight: 280, overflowY: 'auto' }}
            >
              {submitResult.failed.map((f, idx) => (
                <MetaListItem
                  key={f.fileName}
                  last={idx === submitResult.failed.length - 1}
                  title={(
                    <Space size={6}>
                      <span style={{ fontFamily: 'monospace' }}>{f.fileName}</span>
                      <Tag color="red">{f.errorCode}</Tag>
                    </Space>
                  )}
                  description={<Typography.Text type="danger">{f.message}</Typography.Text>}
                />
              ))}
            </MetaList>
          )}
        </div>
      )}
    </Drawer>
  );
}
