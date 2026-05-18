// LicenseOperations — T-0100-P3 重写：Import / Activate / Revoke / Export 四 Tab
// + 当前用户最近 30 天审计日志。所有写操作通过 system:license:operate 权限点门控。
//
// 后端契约：
//   POST /licenses/import     → ImportLicenseResult { license, signatureStatus, signatureNote }
//   POST /licenses/activate   → 200 License | 409 ActivateConflictBody（同维度冲突）
//   POST /licenses/:id/revoke → noop on success
//   GET  /licenses?status=active → 撤销 Tab 下拉 + 激活 Tab 同维度查询源
//   GET  /licenses/logs?actor_user_id=&start_time= → 我的近 30 天操作历史
//
// 多皮肤注意：本文件只在 webcode 主皮肤；frontend-core 的 API / Hook / 类型已
// 同步，webcode-v2/v3 若复刻 LicenseOperations 直接 import @core 即可。

import { useMemo, useState } from 'react';
import {
  Alert,
  Button,
  Card,
  Form,
  Input,
  message,
  Modal,
  Select,
  Space,
  Tabs,
  Tag,
  Tooltip,
  Upload,
} from 'antd';
import {
  CheckCircleOutlined,
  DownloadOutlined,
  ExclamationCircleOutlined,
  InboxOutlined,
  StopOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { UploadFile, RcFile } from 'antd/es/upload';
import type { AxiosError } from 'axios';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import {
  useActivateLicense,
  useImportLicense,
  useImportSignedLicense,
  useLicenseLogs,
  useLicenses,
  useRevokeLicense,
} from '@core/hooks/api/useLicense';
import {
  LicenseErrorCodes,
  extractLicenseErrorCode,
  licenseApi,
  parseActivateConflict,
  type ActivateConflict,
  type LicenseLog,
  type LicenseLogType,
  type LicenseLogResult,
  type LicenseSignatureStatus,
} from '@core/services/api/licenseApi';
import { useUserStore } from '@core/store/userStore';
import { usePermission } from '@core/hooks/usePermission';
import type { License } from '@core/mock/data/license';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;
const { TextArea } = Input;

const PERM_LICENSE_OPERATE = 'system:license:operate';

// 注：auto_revoke_by_activate 不在 LicenseLogType union（后端规划中），用 partial map
// + Tag 默认色兜底，避免 union 类型扩张导致全局 Record 漏键。T-0136 后续 batch 修。
const logTypeColorMap: Partial<Record<LicenseLogType, string>> & Record<string, string> = {
  import: 'blue',
  activate: 'green',
  revoke: 'orange',
  query_detail: 'default',
  enforcement_capacity: 'volcano',
  enforcement_expiry: 'volcano',
  capacity_alert: 'gold',
  expiry_alert: 'gold',
  auto_expire: 'magenta',
  auto_revoke_by_activate: 'cyan',
};

const logResultColorMap: Record<LicenseLogResult, string> = {
  success: 'green',
  failed: 'red',
  denied: 'orange',
  warning: 'gold',
};

const signatureStatusColor: Record<LicenseSignatureStatus, string> = {
  verified: 'green',
  unverified: 'gold',
  invalid: 'red',
};

interface ImportFormValues {
  licenseName: string;
  licenseCode: string;
  productName: string;
  maxDevices: number;
  issueDate: string;
  expiryDate?: string;
  notes?: string;
  pasted?: string;
}

export default function LicenseOperations() {
  const t = useT();
  const canOperate = usePermission(PERM_LICENSE_OPERATE);
  const currentUserId = useUserStore((s) => s.currentUser?.id);

  // ------------------- queries -------------------
  const activeLicensesQuery = useLicenses({ status: 'active', page: 1, pageSize: 200 });
  // BUG 修复 2026-05-18：原 startTime 直接写 `dayjs().subtract(30, 'day').toISOString()`
  // 每次 render 都重算（毫秒级 ISO 字串），React Query 当 key 变化 → 每次 render
  // 都 refetch。本页有 4 个 mutation hook 的 isPending 状态 + Tab/Upload/Form
  // 输入都会触发 re-render，调用频率极高。useMemo 固化挂载时算一次即可
  // ——审计日志 30 天滚动窗口实时性不敏感。
  const myLogsStartTime = useMemo(
    () => dayjs().subtract(30, 'day').toISOString(),
    [],
  );
  const myLogsQuery = useLicenseLogs({
    page: 1,
    pageSize: 50,
    actorUserId: currentUserId,
    startTime: myLogsStartTime,
  });

  // ------------------- mutations -------------------
  const importLicense = useImportLicense();
  const importSignedLicense = useImportSignedLicense(); // T-0100-P5-c 文件上传路径
  const activateLicense = useActivateLicense();
  const revokeLicense = useRevokeLicense();

  // ------------------- Tab 1: Import -------------------
  const [importForm] = Form.useForm<ImportFormValues>();
  const [importFiles, setImportFiles] = useState<UploadFile[]>([]);
  const [importMode, setImportMode] = useState<'file' | 'paste'>('file');

  const handleImport = async () => {
    if (!canOperate) return;

    // 文件模式 + 有上传文件 → 走 P5-c 签名导入路径
    if (importMode === 'file' && importFiles.length > 0) {
      const file = importFiles[0].originFileObj as RcFile | undefined;
      if (!file) {
        void message.error(t('license.noFileSelected'));
        return;
      }
      try {
        const content = await file.text();
        const result = await importSignedLicense.mutateAsync(content);
        renderImportSuccessModal(result);
        importForm.resetFields();
        setImportFiles([]);
      } catch (err) {
        const code = extractLicenseErrorCode(err);
        if (code === LicenseErrorCodes.SignatureVerifyFailed) {
          // 12109 strict 模式拒绝 — 专属 UI 提示
          Modal.error({
            title: t('license.signatureVerifyFailedTitle'),
            content: (
              <Alert
                type="error"
                showIcon
                message={t('license.signatureVerifyFailedMsg')}
                description={extractErrorMessage(err)}
              />
            ),
            okText: t('common.gotIt'),
          });
          return;
        }
        void message.error(extractErrorMessage(err) || t('common.error'));
      }
      return;
    }

    // 粘贴模式 / 表单模式 → 沿用原 importLicense 路径（无签名）
    let values: ImportFormValues;
    try {
      values = await importForm.validateFields();
    } catch {
      return;
    }
    const payload: Omit<License, 'id'> = {
      licenseName: values.licenseName,
      licenseCode: values.licenseCode,
      productName: values.productName,
      licenseType: 'subscription',
      status: 'pending',
      maxDevices: values.maxDevices,
      usedDevices: 0,
      features: [],
      issueDate: values.issueDate,
      expiryDate: values.expiryDate ?? null,
      licensor: '',
      deviceType: '',
      region: '',
      notes: values.notes,
    };

    try {
      const result = await importLicense.mutateAsync(payload);
      renderImportSuccessModal(result);
      importForm.resetFields();
      setImportFiles([]);
    } catch (err) {
      void message.error(extractErrorMessage(err) || t('common.error'));
    }
  };

  // 复用 Import 成功 Modal — 文件路径 + 表单路径共享。
  const renderImportSuccessModal = (result: { license: License; signatureStatus: LicenseSignatureStatus; signatureNote: string }) => {
    Modal.success({
      title: t('license.importSuccess'),
      content: (
        <div>
          <p>
            <strong>{t('license.licenseCode')}:</strong> {result.license.licenseCode}
          </p>
          <p>
            <strong>{t('license.licenseName')}:</strong> {result.license.licenseName}
          </p>
          <p>
            <strong>{t('license.signature')}:</strong>{' '}
            <Tag color={signatureStatusColor[result.signatureStatus]}>
              {t(`license.signature.${result.signatureStatus}`)}
            </Tag>
          </p>
          {result.signatureStatus !== 'verified' && (
            <Alert
              type="warning"
              showIcon
              message={t('license.signatureWarning')}
              description={result.signatureNote}
              style={{ marginTop: 8 }}
            />
          )}
        </div>
      ),
    });
  };

  // ------------------- Tab 2: Activate -------------------
  const [activateCode, setActivateCode] = useState('');

  const doActivate = async (code: string, force: boolean) => {
    try {
      await activateLicense.mutateAsync({ licenseCode: code, force });
      void message.success(t('license.activateSuccess'));
      setActivateCode('');
    } catch (err) {
      const conflict = extractConflict(err);
      if (conflict) {
        Modal.confirm({
          title: t('license.sameDimensionConflictTitle'),
          icon: <ExclamationCircleOutlined />,
          content: <SameDimensionConflictBody conflict={conflict} t={t} />,
          okText: t('license.forceActivate'),
          okButtonProps: { danger: true },
          cancelText: t('common.cancel'),
          onOk: () => doActivate(code, true),
        });
        return;
      }
      void message.error(extractErrorMessage(err) || t('common.error'));
    }
  };

  const handleActivate = () => {
    if (!canOperate) return;
    const code = activateCode.trim();
    if (!code) {
      void message.warning(t('license.enterActivationCode'));
      return;
    }
    void doActivate(code, false);
  };

  // ------------------- Tab 3: Revoke -------------------
  const [revokeId, setRevokeId] = useState<string | undefined>(undefined);
  const activeLicenses: License[] = activeLicensesQuery.data?.items ?? [];

  const handleRevoke = () => {
    if (!canOperate || !revokeId) return;
    const target = activeLicenses.find((l) => l.id === revokeId);
    Modal.confirm({
      title: t('license.confirmRevoke'),
      icon: <ExclamationCircleOutlined />,
      content: (
        <div>
          <p>{t('license.confirmRevokeMsg', { id: target?.licenseCode ?? revokeId })}</p>
          {activeLicenses.length === 1 && (
            <Alert
              type="warning"
              showIcon
              message={t('license.revokeLastActiveWarning')}
              style={{ marginTop: 8 }}
            />
          )}
        </div>
      ),
      okType: 'danger',
      okText: t('license.revoke'),
      cancelText: t('common.cancel'),
      onOk: async () => {
        try {
          await revokeLicense.mutateAsync(revokeId);
          void message.success(t('license.revokeSuccess'));
          setRevokeId(undefined);
        } catch (err) {
          void message.error(extractErrorMessage(err) || t('common.error'));
        }
      },
    });
  };

  // ------------------- Tab 4: Export (T-0100-P4-A) -------------------
  // Single PDF: GET /licenses/:id/export?format=pdf （后端 fpdf 渲染）
  // Bulk CSV:   GET /licenses/export?format=csv     （后端 stdlib encoding/csv + BOM 头）
  const [exportId, setExportId] = useState<string | undefined>(undefined);
  const [exportingSingle, setExportingSingle] = useState(false);
  const [exportingBulk, setExportingBulk] = useState(false);

  const triggerDownload = (blob: Blob, filename: string) => {
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  const handleExportSinglePDF = async () => {
    if (!exportId) return;
    setExportingSingle(true);
    try {
      const { blob, filename } = await licenseApi.exportLicensePDF(exportId);
      triggerDownload(blob, filename);
      void message.success(t('license.exportSuccess'));
    } catch (err) {
      void message.error(extractErrorMessage(err) || t('common.error'));
    } finally {
      setExportingSingle(false);
    }
  };

  const handleExportBulkCSV = async () => {
    setExportingBulk(true);
    try {
      const { blob, filename } = await licenseApi.exportAllActiveCSV();
      triggerDownload(blob, filename);
      void message.success(t('license.exportSuccess'));
    } catch (err) {
      void message.error(extractErrorMessage(err) || t('common.error'));
    } finally {
      setExportingBulk(false);
    }
  };

  // ------------------- 我的近 30 天日志 -------------------
  const myLogs: LicenseLog[] = myLogsQuery.data?.items ?? [];

  const logColumns: DataTableColumn<LicenseLog & Record<string, unknown>>[] = useMemo(
    () => [
      {
        key: 'createdAt',
        title: t('table.time'),
        dataIndex: 'createdAt',
        width: 170,
        render: (val) => dayjs(String(val)).format('YYYY-MM-DD HH:mm:ss'),
      },
      {
        key: 'logType',
        title: t('license.operationType'),
        dataIndex: 'logType',
        width: 160,
        render: (val) => {
          const lt = val as LicenseLogType;
          return <Tag color={logTypeColorMap[lt] ?? 'default'}>{t(`license.logType.${lt}`)}</Tag>;
        },
      },
      {
        key: 'result',
        title: t('table.result'),
        dataIndex: 'result',
        width: 100,
        render: (val) => {
          const r = val as LicenseLogResult;
          return <Tag color={logResultColorMap[r] ?? 'default'}>{t(`license.logResult.${r}`)}</Tag>;
        },
      },
      {
        key: 'details',
        title: t('license.remark'),
        dataIndex: 'details',
        ellipsis: true,
        render: (val) => {
          if (!val || typeof val !== 'object') return '—';
          const d = val as Record<string, unknown>;
          return (
            <span style={{ fontFamily: 'monospace', fontSize: 12 }}>
              {String(d.summary ?? d.license_code ?? JSON.stringify(d).slice(0, 80))}
            </span>
          );
        },
      },
    ],
    [t],
  );

  // ------------------- render -------------------
  const disabledTooltip = canOperate ? '' : t('license.noOperatePermission');

  return (
    <ListPageLayout
      title={t('nav.license.operations')}
      subtitle={t('license.operationsSubtitle')}
    >
      {!canOperate && (
        <Alert
          type="info"
          showIcon
          message={t('license.noOperatePermission')}
          description={t('license.noOperatePermissionDesc')}
          style={{ marginBottom: 16 }}
        />
      )}

      <Tabs
        items={[
          {
            key: 'import',
            label: t('license.importLicense'),
            children: (
              <Card>
                <Space style={{ marginBottom: 16 }}>
                  <Button
                    type={importMode === 'file' ? 'primary' : 'default'}
                    onClick={() => setImportMode('file')}
                  >
                    {t('license.importByFile')}
                  </Button>
                  <Button
                    type={importMode === 'paste' ? 'primary' : 'default'}
                    onClick={() => setImportMode('paste')}
                  >
                    {t('license.importByPaste')}
                  </Button>
                </Space>

                <Alert
                  type="warning"
                  showIcon
                  message={t('license.signatureMvpWarning')}
                  style={{ marginBottom: 16, maxWidth: 720 }}
                />

                {importMode === 'file' ? (
                  <Dragger
                    fileList={importFiles}
                    beforeUpload={(file: RcFile) => {
                      setImportFiles([file]);
                      return false;
                    }}
                    onRemove={() => setImportFiles([])}
                    maxCount={1}
                    accept=".lic,.dat,.xml,.key,.json"
                    style={{ maxWidth: 720, marginBottom: 16 }}
                  >
                    <p className="ant-upload-drag-icon">
                      <InboxOutlined />
                    </p>
                    <p className="ant-upload-text">{t('license.dragFileHere')}</p>
                    <p className="ant-upload-hint">{t('license.supportedFormats')}</p>
                  </Dragger>
                ) : (
                  <Form.Item label={t('license.pasteActivationCode')} style={{ maxWidth: 720 }}>
                    <TextArea
                      rows={6}
                      placeholder={t('license.pasteActivationCode')}
                      style={{ fontFamily: 'monospace' }}
                    />
                  </Form.Item>
                )}

                <Form
                  form={importForm}
                  layout="vertical"
                  style={{ maxWidth: 720 }}
                  disabled={!canOperate || importLicense.isPending}
                >
                  <Form.Item
                    name="licenseName"
                    label={t('license.licenseName')}
                    rules={[{ required: true, message: t('license.licenseNameRequired') }]}
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name="licenseCode"
                    label={t('license.licenseCode')}
                    rules={[{ required: true, message: t('license.licenseCodeRequired') }]}
                  >
                    <Input style={{ fontFamily: 'monospace' }} />
                  </Form.Item>
                  <Form.Item
                    name="productName"
                    label={t('license.productName')}
                    rules={[{ required: true, message: t('license.productNameRequired') }]}
                  >
                    <Input />
                  </Form.Item>
                  <Form.Item
                    name="maxDevices"
                    label={t('license.capacity')}
                    initialValue={100}
                    rules={[{ required: true }]}
                  >
                    <Input type="number" min={1} />
                  </Form.Item>
                  <Form.Item
                    name="issueDate"
                    label={t('license.issueDate')}
                    rules={[{ required: true }]}
                    initialValue={dayjs().toISOString()}
                  >
                    <Input placeholder="ISO 8601" />
                  </Form.Item>
                  <Form.Item name="expiryDate" label={t('license.expiryDate')}>
                    <Input placeholder="ISO 8601 (空表示永久)" />
                  </Form.Item>
                  <Form.Item name="notes" label={t('license.notes')}>
                    <TextArea rows={2} />
                  </Form.Item>
                </Form>

                <Tooltip title={disabledTooltip}>
                  <Button
                    type="primary"
                    icon={<UploadOutlined />}
                    loading={importLicense.isPending}
                    disabled={!canOperate}
                    onClick={handleImport}
                    style={{ marginTop: 8 }}
                  >
                    {t('license.importLicense')}
                  </Button>
                </Tooltip>
              </Card>
            ),
          },
          {
            key: 'activate',
            label: t('license.activateLicense'),
            children: (
              <Card>
                <div style={{ maxWidth: 720 }}>
                  <p style={{ color: '#666', marginBottom: 16 }}>
                    {t('license.activateDescription')}
                  </p>
                  <TextArea
                    rows={4}
                    value={activateCode}
                    onChange={(e) => setActivateCode(e.target.value)}
                    placeholder={t('license.pasteActivationCode')}
                    style={{ fontFamily: 'monospace', marginBottom: 16 }}
                    disabled={!canOperate || activateLicense.isPending}
                  />
                  <Tooltip title={disabledTooltip}>
                    <Button
                      type="primary"
                      icon={<CheckCircleOutlined />}
                      loading={activateLicense.isPending}
                      disabled={!canOperate}
                      onClick={handleActivate}
                      block
                    >
                      {t('license.activateLicense')}
                    </Button>
                  </Tooltip>
                </div>
              </Card>
            ),
          },
          {
            key: 'revoke',
            label: t('license.revokeLicense'),
            children: (
              <Card>
                <div style={{ maxWidth: 720 }}>
                  <Alert
                    type="error"
                    showIcon
                    message={t('license.revokeWarning')}
                    style={{ marginBottom: 16 }}
                  />
                  <Form.Item label={t('license.selectActiveLicense')}>
                    <Select
                      placeholder={t('license.selectActiveLicensePlaceholder')}
                      value={revokeId}
                      onChange={setRevokeId}
                      loading={activeLicensesQuery.isLoading}
                      disabled={!canOperate}
                      options={activeLicenses.map((l) => ({
                        value: l.id,
                        label: `${l.licenseCode} — ${l.licenseName} (${l.maxDevices})`,
                      }))}
                      style={{ width: '100%', marginBottom: 16 }}
                    />
                  </Form.Item>
                  <Tooltip title={disabledTooltip}>
                    <Button
                      danger
                      type="primary"
                      icon={<StopOutlined />}
                      loading={revokeLicense.isPending}
                      disabled={!canOperate || !revokeId}
                      onClick={handleRevoke}
                      block
                    >
                      {t('license.revokeLicense')}
                    </Button>
                  </Tooltip>
                </div>
              </Card>
            ),
          },
          {
            key: 'export',
            label: t('license.exportLicense'),
            children: (
              <Card>
                <div style={{ maxWidth: 720 }}>
                  <Alert
                    type="info"
                    showIcon
                    message={t('license.exportP4aHint')}
                    style={{ marginBottom: 16 }}
                  />

                  <h4 style={{ marginBottom: 8 }}>{t('license.exportSingleTitle')}</h4>
                  <Form.Item label={t('license.selectActiveLicense')}>
                    <Select
                      placeholder={t('license.selectActiveLicensePlaceholder')}
                      value={exportId}
                      onChange={setExportId}
                      loading={activeLicensesQuery.isLoading}
                      options={activeLicenses.map((l) => ({
                        value: l.id,
                        label: `${l.licenseCode} — ${l.licenseName}`,
                      }))}
                      style={{ width: '100%', marginBottom: 16 }}
                    />
                  </Form.Item>
                  <Button
                    type="primary"
                    icon={<DownloadOutlined />}
                    disabled={!exportId}
                    loading={exportingSingle}
                    onClick={handleExportSinglePDF}
                    block
                    style={{ marginBottom: 24 }}
                  >
                    {t('license.exportPdf')}
                  </Button>

                  <h4 style={{ marginBottom: 8 }}>{t('license.exportBulkTitle')}</h4>
                  <p style={{ color: '#666', marginBottom: 12 }}>
                    {t('license.exportBulkHint')}
                  </p>
                  <Button
                    icon={<DownloadOutlined />}
                    loading={exportingBulk}
                    onClick={handleExportBulkCSV}
                    block
                  >
                    {t('license.exportCsvBulk')}
                  </Button>
                </div>
              </Card>
            ),
          },
        ]}
      />

      <Card title={t('license.myRecent30dLogs')} style={{ marginTop: 24 }}>
        <DataTable
          tableId="license-operations-my-logs"
          columns={logColumns}
          dataSource={myLogs as (LicenseLog & Record<string, unknown>)[]}
          loading={myLogsQuery.isLoading}
          rowKey="id"
          total={myLogs.length}
          pageSize={50}
          currentPage={1}
          scroll={{ x: 800 }}
        />
      </Card>
    </ListPageLayout>
  );
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

interface SameDimensionConflictBodyProps {
  conflict: ActivateConflict;
  t: ReturnType<typeof useT>;
}

function SameDimensionConflictBody({ conflict, t }: SameDimensionConflictBodyProps) {
  return (
    <div>
      <p>{t('license.sameDimensionConflictMsg')}</p>
      <ul style={{ paddingLeft: 16 }}>
        {conflict.conflictingActive.map((l) => (
          <li key={l.id}>
            <strong>{l.licenseCode}</strong> — {l.licenseName} ({l.maxDevices})
          </li>
        ))}
      </ul>
      <p style={{ marginTop: 8, color: '#888', fontSize: 12 }}>
        {t('license.dimensionLabel')}: device_type=
        <code>{conflict.deviceType ?? '<null>'}</code>, region=
        <code>{conflict.region ?? '<null>'}</code>
      </p>
    </div>
  );
}

function extractConflict(err: unknown): ActivateConflict | null {
  const ax = err as AxiosError<unknown> | undefined;
  if (!ax || ax.response?.status !== 409) return null;
  return parseActivateConflict(ax.response.data);
}

function extractErrorMessage(err: unknown): string {
  const ax = err as AxiosError<{ msg?: string; message?: string }> | undefined;
  if (ax?.response?.data) {
    const d = ax.response.data;
    if (typeof d === 'object' && d !== null) {
      return d.msg ?? d.message ?? '';
    }
  }
  if (err instanceof Error) return err.message;
  return '';
}
