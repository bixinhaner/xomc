/**
 * UpdateModal — F06 System License 重构 Step 4。
 *
 * 流程（PRD §6 Update 操作）：
 *   1. 用户拖入 / 点击选择旧项目 TrueLicense 二进制 license 文件（.lic）
 *   2. 前端 FileReader 读取二进制并转 Base64（仅作为 HTTP 传输编码）
 *   3. 用户确认 → POST /system-license → 后端解密、验签、替换
 *   4. 成功 → message.success + 提示是否替换了旧 license
 *   5. 失败 → 按 biz_code 区分 12109/12110/12111 给针对性 toast
 */
import { useCallback, useMemo, useState } from 'react';
import { Alert, Descriptions, Modal, Tag, Typography, Upload, message } from 'antd';
import { InboxOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';

import { useT } from '@/hooks/useT';
import { useUpdateSystemLicense } from '@core/hooks/api/useSystemLicense';
import {
  SystemLicenseErrorCodes,
  extractLicenseErrorCode,
} from '@core/services/api/systemLicenseApi';

const { Dragger } = Upload;
const { Text } = Typography;

interface UpdateModalProps {
  open: boolean;
  onClose: () => void;
}

interface ParsedPreview {
  raw: string;
  legacy: boolean;
  licenseId?: string;
  licenseType?: string;
  issuedAt?: string;
  expiryDate?: string;
  devicesSupport?: Record<string, number>;
  hasSignature: boolean;
}

export default function UpdateModal({ open, onClose }: UpdateModalProps) {
  const t = useT();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [preview, setPreview] = useState<ParsedPreview | null>(null);
  const [parseError, setParseError] = useState<string | null>(null);
  const updateMutation = useUpdateSystemLicense();

  // 关闭弹窗时复位本地状态。把 reset 写进关闭路径（onCancel / onOk 成功）
  // 比放在 useEffect 里更明确，也避开 ESLint react-hooks/set-state-in-effect。
  const resetAndClose = useCallback(() => {
    setFileList([]);
    setPreview(null);
    setParseError(null);
    onClose();
  }, [onClose]);

  const canSubmit = useMemo(
    () => preview !== null && !parseError && !updateMutation.isPending,
    [preview, parseError, updateMutation.isPending],
  );

  // Dragger beforeUpload 钩子：本地读取旧 .lic 二进制并转 Base64（不真上传）
  const beforeUpload = (file: File) => {
    const reader = new FileReader();
    const isLegacy = file.name.toLowerCase().endsWith('.lic');
    reader.onload = () => {
      if (!isLegacy) {
        setPreview(null);
        setParseError(t('systemLicense.update.licOnly'));
        return;
      }
      const bytes = new Uint8Array(reader.result as ArrayBuffer);
      let binary = '';
      for (let offset = 0; offset < bytes.length; offset += 0x8000) {
        binary += String.fromCharCode(...bytes.subarray(offset, offset + 0x8000));
      }
      setPreview({ raw: btoa(binary), legacy: true, hasSignature: false });
      setParseError(null);
    };
    reader.onerror = () => {
      setPreview(null);
      setParseError(reader.error?.message ?? 'FileReader error');
    };
    reader.readAsArrayBuffer(file);
    // Antd Dragger 显示文件本身用，禁止自动上传
    setFileList([
      {
        uid: file.name + Date.now(),
        name: file.name,
        status: 'done',
        size: file.size,
      } as UploadFile,
    ]);
    return false; // prevent auto upload
  };

  const handleSubmit = () => {
    if (!preview) return;
    updateMutation.mutate({ rawContent: preview.raw }, {
      onSuccess: (data) => {
        message.success(t('systemLicense.update.success'));
        if (data.replaced) {
          message.info(t('systemLicense.update.replacedHint', { licenseId: data.replaced.licenseId }));
        } else {
          message.info(t('systemLicense.update.firstInstallHint'));
        }
        resetAndClose();
        // 上传成功后刷新整个页面：license 数据 + 菜单 + feature 状态全部重新加载
        setTimeout(() => window.location.reload(), 800);
      },
      onError: (err) => {
        const code = extractLicenseErrorCode(err);
        const msg = (err as Error)?.message ?? '';
        let key = 'common.error';
        let values: Record<string, string> = { msg };
        switch (code) {
          case SystemLicenseErrorCodes.SignatureVerifyFailed:
            key = 'systemLicense.update.errors.signatureFailed';
            break;
          case SystemLicenseErrorCodes.IDExists:
            key = 'systemLicense.update.errors.idExists';
            values = {};
            break;
          case SystemLicenseErrorCodes.InvalidFormat:
            key = 'systemLicense.update.errors.invalidFormat';
            break;
          default:
            // unknown — show backend message verbatim
            message.error(msg);
            return;
        }
        message.error(t(key, values));
      },
    });
  };

  return (
    <Modal
      title={t('systemLicense.update.title')}
      open={open}
      onCancel={resetAndClose}
      onOk={handleSubmit}
      okText={t('systemLicense.update.confirm')}
      cancelText={t('systemLicense.update.cancel')}
      okButtonProps={{ disabled: !canSubmit, loading: updateMutation.isPending }}
      width={680}
      destroyOnHidden
    >
      <Dragger
        accept=".lic"
        multiple={false}
        showUploadList={{ showRemoveIcon: true }}
        fileList={fileList}
        beforeUpload={beforeUpload}
        onRemove={() => {
          setFileList([]);
          setPreview(null);
          setParseError(null);
          return true;
        }}
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">{t('systemLicense.update.dragHint')}</p>
        <p className="ant-upload-hint">{t('systemLicense.update.dragSubHint')}</p>
      </Dragger>

      {parseError ? (
        <Alert
          style={{ marginTop: 16 }}
          type="error"
          showIcon
          message={t('systemLicense.update.parseError', { msg: parseError })}
        />
      ) : null}

      <div style={{ marginTop: 16 }}>
        <Text strong>{t('systemLicense.update.preview')}</Text>
        {preview?.legacy ? (
          <Text type="secondary">{t('systemLicense.update.legacyPreview')}</Text>
        ) : preview ? (
          <Descriptions
            size="small"
            column={2}
            bordered
            style={{ marginTop: 8 }}
          >
            <Descriptions.Item label="License ID">{preview.licenseId ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="Type">{preview.licenseType ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="Issued At">{preview.issuedAt ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="Expiry">{preview.expiryDate ?? '-'}</Descriptions.Item>
            <Descriptions.Item label="Devices Support" span={2}>
              {preview.devicesSupport ? (
                Object.entries(preview.devicesSupport).map(([k, v]) => (
                  <Tag key={k} style={{ marginRight: 6 }}>
                    {k}: {v}
                  </Tag>
                ))
              ) : (
                <Text type="secondary">-</Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="Signature" span={2}>
              {preview.hasSignature ? (
                <Tag color="success">included</Tag>
              ) : (
                <Tag color="warning">{t('systemLicense.update.noSignatureWarn')}</Tag>
              )}
            </Descriptions.Item>
          </Descriptions>
        ) : (
          <div style={{ color: '#999', marginTop: 8 }}>
            {t('systemLicense.update.previewEmpty')}
          </div>
        )}
      </div>
    </Modal>
  );
}
