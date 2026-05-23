/**
 * ImportDrawer (T-0165) — 设备 license 批量导入。结构镜像
 * ConfigSnapshotLibrary/ImportDrawer：拖拽+点击混合上传 + SN 后端校验 +
 * 命名校验 + 失败明细。文件名规范：<SN>_LIC.{lic|bin|dat}。
 */
import { useEffect, useMemo, useState } from 'react';
import { Alert, Button, Drawer, Empty, List, Space, Tag, Typography, Upload, message } from 'antd';
import {
  InboxOutlined, FileDoneOutlined, ExclamationCircleOutlined, DeleteOutlined, LoadingOutlined,
} from '@ant-design/icons';
import { useImportDeviceLicenses } from '@core/hooks/api/useDeviceLicense';
import {
  deviceLicenseApi,
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
  snKnown: 'unknown' | 'existing' | 'missing' | 'pending';
}

export default function ImportDrawer({ open, onClose, onSuccess }: Props) {
  const [files, setFiles] = useState<ParsedFile[]>([]);
  const [submitResult, setSubmitResult] = useState<LicenseImportResult | null>(null);
  const importMutation = useImportDeviceLicenses();

  const isFileOk = (f: ParsedFile) => f.valid && f.snKnown === 'existing';
  const isFileBad = (f: ParsedFile) => !f.valid || f.snKnown === 'missing';
  const validCount = useMemo(() => files.filter(isFileOk).length, [files]);
  const invalidCount = useMemo(() => files.filter(isFileBad).length, [files]);
  const pendingCount = useMemo(() => files.filter((f) => f.snKnown === 'pending').length, [files]);
  const canSubmit = validCount > 0 && invalidCount === 0 && pendingCount === 0;

  function handleAdd(file: File): boolean {
    setFiles((prev) => {
      if (prev.some((f) => f.fileName === file.name)) {
        void message.warning(`已存在同名文件 ${file.name}，已跳过`);
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
          snKnown: v.valid ? 'pending' : 'unknown',
        },
      ];
    });
    return false;
  }

  useEffect(() => {
    const pendingSNs = files
      .filter((f) => f.snKnown === 'pending' && f.serialNumber)
      .map((f) => f.serialNumber as string);
    if (pendingSNs.length === 0) return;
    let cancelled = false;
    deviceLicenseApi
      .validateSNs(pendingSNs)
      .then((res) => {
        if (cancelled) return;
        const missingSet = new Set(res.missing);
        setFiles((prev) =>
          prev.map((f) =>
            f.snKnown === 'pending' && f.serialNumber
              ? { ...f, snKnown: missingSet.has(f.serialNumber) ? 'missing' : 'existing' }
              : f,
          ),
        );
      })
      .catch(() => {
        if (cancelled) return;
        setFiles((prev) =>
          prev.map((f) => (f.snKnown === 'pending' ? { ...f, snKnown: 'existing' } : f)),
        );
      });
    return () => {
      cancelled = true;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [files.map((f) => `${f.fileName}|${f.snKnown}`).join(',')]);

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
        void message.success(`导入成功 ${result.succeeded.length} 台设备`);
        setFiles([]);
      }
      if (result.failed.length > 0) {
        void message.warning(`${result.failed.length} 个文件失败，详见下方列表`);
      }
      onSuccess?.();
    } catch {
      void message.error('导入请求失败，请稍后重试');
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
      title="导入 License 文件"
      width={620}
      onClose={handleClose}
      footer={
        <Space style={{ float: 'right' }}>
          <Button onClick={handleClose}>关闭</Button>
          <Button
            type="primary"
            disabled={!canSubmit}
            loading={importMutation.isPending}
            onClick={() => void handleSubmit()}
          >
            提交导入（{validCount}）
          </Button>
        </Space>
      }
    >
      <Alert
        type="info"
        showIcon
        message="导入规则"
        description={(
          <>
            ① 文件名必须为 <code>&lt;serialNumber&gt;.lic</code>（仅支持 .lic 后缀）。
            <br />
            ② <strong>SN 必须在设备列表中存在</strong>（不存在的设备无法导入）。
            <br />
            ③ 单文件 ≤ 10 MB，单次最多 200 个文件。
            <br />
            <strong>有任一项不符将无法提交，请先移除问题文件。</strong>
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
                点击或拖拽 license 文件到此处，支持一次选择多个
              </div>
              <div style={{ fontSize: 12, color: 'rgba(0,0,0,0.45)' }}>
                仅支持 .lic 后缀，文件名需为 &lt;serialNumber&gt;.lic
              </div>
            </div>
          </Space>
        </Upload.Dragger>
      </div>

      <div style={{ marginTop: 16 }}>
        <Space style={{ width: '100%', justifyContent: 'space-between', marginBottom: 8 }}>
          <Typography.Title level={5} style={{ margin: 0 }}>
            待提交清单（{files.length}）
            {invalidCount > 0 && <Tag color="red" style={{ marginLeft: 8 }}>{invalidCount} 个有问题</Tag>}
            {pendingCount > 0 && <Tag color="processing" style={{ marginLeft: 4 }}>{pendingCount} 个校验中</Tag>}
            {validCount > 0 && <Tag color="green" style={{ marginLeft: 4 }}>{validCount} 个可提交</Tag>}
          </Typography.Title>
          {files.length > 0 && (
            <Button type="link" danger size="small" onClick={handleClearAll}>
              全部清空
            </Button>
          )}
        </Space>

        {files.length === 0 ? (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="还没有文件 — 请拖拽或点击上方区域上传"
          />
        ) : (
          <div style={{ maxHeight: 420, overflowY: 'auto', border: '1px solid #f0f0f0', borderRadius: 6 }}>
            <List
              size="small"
              split
              dataSource={files}
              style={{ background: '#fff' }}
              renderItem={(f) => {
                const bad = isFileBad(f);
                const pending = f.snKnown === 'pending';
                return (
                  <List.Item
                    style={bad ? { background: '#fff1f0' } : undefined}
                    actions={[
                      <Button
                        key="rm"
                        type="link"
                        danger
                        size="small"
                        icon={<DeleteOutlined />}
                        onClick={() => handleRemove(f.fileName)}
                      >
                        移除
                      </Button>,
                    ]}
                  >
                    <List.Item.Meta
                      avatar={
                        pending ? <LoadingOutlined style={{ color: '#1677ff', fontSize: 18 }} />
                          : bad ? <ExclamationCircleOutlined style={{ color: '#ff4d4f', fontSize: 18 }} />
                          : <FileDoneOutlined style={{ color: '#52c41a', fontSize: 18 }} />
                      }
                      title={(
                        <Space size={6} wrap>
                          <span style={{ fontFamily: 'monospace', color: bad ? '#ff4d4f' : undefined }}>
                            {f.fileName}
                          </span>
                          {f.valid && f.serialNumber && (
                            <Tag color={f.snKnown === 'missing' ? 'red' : 'blue'}>
                              SN: {f.serialNumber}
                            </Tag>
                          )}
                          {f.valid && f.ext && <Tag color="geekblue">{f.ext.toUpperCase()}</Tag>}
                          {f.snKnown === 'existing' && <Tag color="green">设备已注册</Tag>}
                          {f.snKnown === 'missing' && <Tag color="red">设备不存在</Tag>}
                          {pending && <Tag color="processing">校验设备中…</Tag>}
                        </Space>
                      )}
                      description={(
                        !f.valid ? (
                          <Typography.Text type="danger">{f.reason}</Typography.Text>
                        ) : f.snKnown === 'missing' ? (
                          <Typography.Text type="danger">
                            SN {f.serialNumber} 不在设备列表中。请检查文件名前缀是否正确，或先在「设备管理」录入该设备。
                          </Typography.Text>
                        ) : (
                          <Typography.Text type="secondary">
                            {(f.rawFile.size / 1024).toFixed(1)} KB
                          </Typography.Text>
                        )
                      )}
                    />
                  </List.Item>
                );
              }}
            />
          </div>
        )}
      </div>

      {submitResult && (
        <div style={{ marginTop: 16 }}>
          <Alert
            type={submitResult.failed.length === 0 ? 'success' : 'warning'}
            showIcon
            message={`本次提交：成功 ${submitResult.succeeded.length} 台，失败 ${submitResult.failed.length} 个`}
          />
          {submitResult.failed.length > 0 && (
            <List
              style={{ marginTop: 8, maxHeight: 280, overflowY: 'auto' }}
              size="small"
              bordered
              header={<Typography.Text strong>失败明细</Typography.Text>}
              dataSource={submitResult.failed}
              renderItem={(f) => (
                <List.Item>
                  <List.Item.Meta
                    title={(
                      <Space size={6}>
                        <span style={{ fontFamily: 'monospace' }}>{f.fileName}</span>
                        <Tag color="red">{f.errorCode}</Tag>
                      </Space>
                    )}
                    description={<Typography.Text type="danger">{f.message}</Typography.Text>}
                  />
                </List.Item>
              )}
            />
          )}
        </div>
      )}
    </Drawer>
  );
}
