/**
 * T-0132 admin Tab 4 XML 导入 — 完整版（替代 T-0123-P3 占位）。
 *
 * 流程：
 *   1. 选择 XML 文件（File 类型 antd Upload beforeUpload + 拦截不真上传）
 *   2. 填 version_code（默认 "STANDARD"）
 *   3. 点 "预览" → POST /mml/admin/import/preview (multipart) → 显示三桶 summary + 前 200 行 diff
 *   4. Confirm Modal 后点 "确认导入" → POST /mml/admin/import/apply → 显示结果
 *   5. 成功后 React Query 自动 invalidate group-tree + params list
 *
 * 设计取舍：
 *   - preview / apply 各自重传同一文件（防 server-side session 复杂化）
 *   - 文件 size 上限前端先校 2MB（与后端 maxImportUploadBytes 对齐）
 *   - 三桶颜色：add=绿色 / modify=蓝色 / skipped=橙色（admin 改过受守护）
 */

import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Descriptions,
  Empty,
  Form,
  Input,
  Modal,
  Space,
  Spin,
  Table,
  Tag,
  Upload,
  App,
} from 'antd';
import type { UploadFile } from 'antd/es/upload/interface';
import { InboxOutlined, ImportOutlined, EyeOutlined } from '@ant-design/icons';

import { useImportPreview, useImportApply } from '@core/hooks/api/useMmlAdmin';
import type { ImportBucket, ImportDiff } from '@core/types/mmlAdmin';

import { useT } from '@/hooks/useT';

const MAX_XML_BYTES = 2 << 20; // 2 MiB (与后端 maxImportUploadBytes 对齐)

function bucketColor(b: ImportBucket): string {
  switch (b) {
    case 'add':
      return 'green';
    case 'modify':
      return 'blue';
    case 'skipped':
      return 'orange';
    default:
      return 'default';
  }
}

export default function XmlImportTab() {
  const t = useT();
  const { message } = App.useApp();
  const [file, setFile] = useState<File | null>(null);
  const [versionCode, setVersionCode] = useState<string>('STANDARD');

  const preview = useImportPreview();
  const apply = useImportApply();

  const beforeUpload = (f: File): boolean => {
    if (f.size > MAX_XML_BYTES) {
      message.error(
        t('mml.admin.catalog.xmlImport.fileTooLarge', {
          size: Math.round(f.size / 1024),
          limit: MAX_XML_BYTES / 1024,
        }),
      );
      return false;
    }
    setFile(f);
    preview.reset();
    apply.reset();
    return false; // 阻止真上传 (we send manually via mutation)
  };

  const handlePreview = (): void => {
    if (!file) {
      message.warning(t('mml.admin.catalog.xmlImport.fileRequired'));
      return;
    }
    if (!versionCode.trim()) {
      message.warning(t('mml.admin.catalog.xmlImport.versionRequired'));
      return;
    }
    preview.mutate(
      { file, versionCode: versionCode.trim() },
      {
        onError: (err) =>
          message.error(
            t('mml.admin.catalog.xmlImport.previewFailed') + ': ' + err.message,
          ),
      },
    );
  };

  const handleApply = (): void => {
    if (!file || !preview.data) return;
    Modal.confirm({
      title: t('mml.admin.catalog.xmlImport.applyConfirmTitle'),
      content: t('mml.admin.catalog.xmlImport.applyConfirmContent', {
        add: preview.data.summary.add,
        modify: preview.data.summary.modify,
        skipped: preview.data.summary.skipped,
      }),
      okText: t('mml.admin.catalog.xmlImport.applyOK'),
      cancelText: t('mml.admin.catalog.xmlImport.applyCancel'),
      onOk: () => {
        apply.mutate(
          { file, versionCode: versionCode.trim() },
          {
            onSuccess: (data) =>
              message.success(
                t('mml.admin.catalog.xmlImport.applySuccess', {
                  affected: data.rowsAffected,
                  total: data.total,
                }),
              ),
            onError: (err) =>
              message.error(
                t('mml.admin.catalog.xmlImport.applyFailed') + ': ' + err.message,
              ),
          },
        );
      },
    });
  };

  const summary = preview.data?.summary;
  const diffs = preview.data?.diffs ?? [];
  const truncated = preview.data?.truncated ?? false;

  const columns = useMemo(
    () => [
      {
        title: t('mml.admin.catalog.xmlImport.col.bucket'),
        dataIndex: 'bucket',
        width: 100,
        render: (b: ImportBucket) => (
          <Tag color={bucketColor(b)}>
            {t(`mml.admin.catalog.xmlImport.bucket.${b}`)}
          </Tag>
        ),
      },
      {
        title: t('mml.admin.catalog.xmlImport.col.tr069Path'),
        dataIndex: 'tr069Path',
        ellipsis: true,
      },
      {
        title: t('mml.admin.catalog.xmlImport.col.paramCode'),
        dataIndex: 'paramCode',
        width: 220,
        ellipsis: true,
      },
      {
        title: t('mml.admin.catalog.xmlImport.col.valueType'),
        dataIndex: 'valueType',
        width: 100,
      },
      {
        title: t('mml.admin.catalog.xmlImport.col.accessType'),
        dataIndex: 'accessType',
        width: 110,
      },
      {
        title: t('mml.admin.catalog.xmlImport.col.isObject'),
        dataIndex: 'isObject',
        width: 80,
        render: (v: boolean) => (v ? '✓' : ''),
      },
    ],
    [t],
  );

  const fileList: UploadFile[] = file
    ? [{ uid: '-1', name: file.name, status: 'done' as const, size: file.size }]
    : [];

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 16 }}>
      <Alert
        type="info"
        showIcon
        message={t('mml.admin.catalog.xmlImport.title')}
        description={t('mml.admin.catalog.xmlImport.intro')}
      />

      <Form layout="inline">
        <Form.Item label={t('mml.admin.catalog.xmlImport.versionLabel')}>
          <Input
            value={versionCode}
            onChange={(e) => setVersionCode(e.target.value)}
            placeholder="STANDARD"
            style={{ width: 200 }}
          />
        </Form.Item>
      </Form>

      <Upload.Dragger
        accept=".xml,text/xml,application/xml"
        beforeUpload={beforeUpload}
        onRemove={() => {
          setFile(null);
          preview.reset();
          apply.reset();
        }}
        fileList={fileList}
        multiple={false}
        maxCount={1}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">
          {t('mml.admin.catalog.xmlImport.uploadHint')}
        </p>
        <p className="ant-upload-hint">
          {t('mml.admin.catalog.xmlImport.uploadSubHint', {
            limit: MAX_XML_BYTES / 1024 / 1024,
          })}
        </p>
      </Upload.Dragger>

      <Space>
        <Button
          type="primary"
          icon={<EyeOutlined />}
          loading={preview.isPending}
          disabled={!file}
          onClick={handlePreview}
        >
          {t('mml.admin.catalog.xmlImport.previewButton')}
        </Button>
        <Button
          type="primary"
          danger
          icon={<ImportOutlined />}
          loading={apply.isPending}
          disabled={!preview.data}
          onClick={handleApply}
        >
          {t('mml.admin.catalog.xmlImport.applyButton')}
        </Button>
      </Space>

      {preview.isPending && (
        <Spin tip={t('mml.admin.catalog.xmlImport.previewLoading')} />
      )}

      {summary && (
        <Descriptions
          bordered
          size="small"
          column={4}
          title={t('mml.admin.catalog.xmlImport.summaryTitle')}
        >
          <Descriptions.Item label={t('mml.admin.catalog.xmlImport.bucket.add')}>
            <Tag color="green">{summary.add}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('mml.admin.catalog.xmlImport.bucket.modify')}>
            <Tag color="blue">{summary.modify}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('mml.admin.catalog.xmlImport.bucket.skipped')}>
            <Tag color="orange">{summary.skipped}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label={t('mml.admin.catalog.xmlImport.total')}>
            <Tag>{summary.total}</Tag>
          </Descriptions.Item>
        </Descriptions>
      )}

      {summary && diffs.length > 0 && (
        <div>
          <Table<ImportDiff>
            size="small"
            rowKey={(r) => r.tr069Path}
            dataSource={diffs}
            columns={columns}
            pagination={{ pageSize: 20, size: 'small' }}
          />
          {truncated && (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 8 }}
              message={t('mml.admin.catalog.xmlImport.truncatedHint', {
                shown: diffs.length,
                total: preview.data?.total ?? 0,
              })}
            />
          )}
        </div>
      )}

      {!file && !summary && (
        <Empty description={t('mml.admin.catalog.xmlImport.emptyHint')} />
      )}
    </div>
  );
}
