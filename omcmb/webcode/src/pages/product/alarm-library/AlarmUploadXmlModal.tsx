/**
 * AlarmUploadXmlModal — 告警库自定义 XML 上传弹窗(对标 kpi-library UploadXmlModal)。
 *
 * 导入 XML 调整(2026-06-05 取消手填名称):
 *   - XML 文件 *  (根元素必须 <alarmModel>,≤ 1 MiB)
 *   - 唯一名称取自 XML <alarmModel neType="..."> 属性,文件将保存为 <neType>.xml
 *     (选文件后即时预览)。
 *   重复允许覆盖(二次确认):本地预检(useAlarmNeTypeStats 文件名 / neType 比对)
 *   或后端 409(data.overwritable=true)→ Modal.confirm「已存在,确认覆盖?」→
 *   确认后带 force=true 重试,后端覆盖归属文件并自动备份旧文件 .bak.<ts>。
 */
import { useEffect, useState } from 'react';
import { Modal, Form, Upload, message, Button, Space, Typography } from 'antd';
import type { UploadFile, UploadProps } from 'antd/es/upload/interface';
import { InboxOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import { useAlarmUploadXml, useAlarmNeTypeStats } from '@core/hooks/api/useAlarmDefinitions';
import { useSystemLicense } from '@core/hooks/api/useSystemLicense';
import { isDeviceStandardValueVisibleByLicense } from '@core/utils/licenseFeatures';
import { extractXmlRootAttr } from '@core/utils/xmlRootAttr';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

const MAX_SIZE = 1 * 1024 * 1024; // 1 MiB,与后端 MaxUploadXMLSize 一致

export default function AlarmUploadXmlModal({ open, onClose }: Props) {
  const t = useT();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  // 选文件后从 XML neType 属性提取的名称(将保存为 <neType>.xml);null = 未提取到
  const [derivedName, setDerivedName] = useState<string | null>(null);
  const [fileError, setFileError] = useState<string | undefined>();
  const uploadMut = useAlarmUploadXml();
  const { data: systemLicense, isLoading: systemLicenseLoading } = useSystemLicense();

  // 上传前查重:ne-types 聚合行的 neType / loadedFrom basename(大小写不敏感)对比。
  const { data: neTypesData } = useAlarmNeTypeStats();
  const isDuplicate = (neType: string): boolean => {
    const target = `${neType}.xml`.toLowerCase();
    const lower = neType.toLowerCase();
    return (neTypesData?.items || []).some((row) => {
      const base = (row.loadedFrom || '').split('/').pop()?.toLowerCase() ?? '';
      return base === target || row.neType.toLowerCase() === lower;
    });
  };

  const handleClose = () => {
    setFileList([]);
    setDerivedName(null);
    setFileError(undefined);
    onClose();
  };

  useEffect(() => {
    if (!derivedName || systemLicenseLoading) return;
    if (!isDeviceStandardValueVisibleByLicense(derivedName, systemLicense, false)) {
      setFileError(t('systemLicense.deviceStandardImportDenied', { standard: derivedName }));
    }
  }, [derivedName, systemLicense, systemLicenseLoading, t]);

  // 实际上传(force = 二次确认后的覆盖);成功后关弹窗,失败统一 message。
  const doUpload = async (file: File, force: boolean) => {
    try {
      const r = await uploadMut.mutateAsync({ file, force });
      message.success(
        r.overwritten
          ? t('product.upload.overwriteSuccess', { file: r.filename })
          : t('product.alarm.upload.uploadSuccess', { file: r.filename }),
      );
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ msg?: string; message?: string; data?: { overwritable?: boolean } }>;
      const body = ax.response?.data;
      const msg = body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e));
      // 409 + overwritable(本地预检漏判的重复)→ 弹确认覆盖
      if (!force && ax.response?.status === 409 && body?.data?.overwritable) {
        confirmOverwrite(derivedName ?? '', () => void doUpload(file, true));
        return;
      }
      message.error(msg);
    }
  };

  // 覆盖二次确认弹框 —— 确认后才带 force=true 重试。
  const confirmOverwrite = (name: string, onOk: () => void) => {
    Modal.confirm({
      title: t('product.upload.overwriteConfirmTitle'),
      content: t('product.upload.overwriteConfirmContent', { name }),
      okText: t('common.confirm'),
      cancelText: t('common.cancel'),
      okButtonProps: { danger: true },
      onOk,
    });
  };

  const handleSubmit = async () => {
    if (fileList.length === 0 || !fileList[0].originFileObj) {
      message.error(t('product.alarm.upload.selectXmlMsg'));
      return;
    }
    // neType 属性是名称唯一来源 —— 读不到直接拒绝(后端也会 400)
    if (!derivedName) {
      setFileError(t('product.upload.missingAttr', { attr: 'neType' }));
      return;
    }
    if (systemLicenseLoading) {
      setFileError(t('common.loading'));
      return;
    }
    if (!isDeviceStandardValueVisibleByLicense(derivedName, systemLicense, false)) {
      setFileError(t('systemLicense.deviceStandardImportDenied', { standard: derivedName }));
      return;
    }
    const file = fileList[0].originFileObj;
    // 上传前查重:neType / 文件名已存在 → 弹覆盖确认(后端 409 仍是兜底真值源)。
    if (isDuplicate(derivedName)) {
      confirmOverwrite(derivedName, () => void doUpload(file, true));
      return;
    }
    await doUpload(file, false);
  };

  const uploadProps: UploadProps = {
    accept: '.xml',
    maxCount: 1,
    fileList,
    beforeUpload: (f) => {
      if (f.size > MAX_SIZE) {
        message.error(t('product.alarm.upload.fileTooLarge', { size: (f.size / 1024).toFixed(1) }));
        return Upload.LIST_IGNORE;
      }
      setFileError(undefined);
      // 选文件即抽 neType,预览"将保存为 <neType>.xml"。fire-and-forget。
      void extractXmlRootAttr(f, 'neType').then((nt) => setDerivedName(nt));
      return false; // 阻止 antd 自动上传 — 走 handleSubmit
    },
    onChange: ({ fileList: newList }) => setFileList(newList.slice(-1)),
    onRemove: () => {
      setFileList([]);
      setDerivedName(null);
      setFileError(undefined);
    },
  };

  return (
    <Modal
      title={t('product.alarm.upload.title')}
      open={open}
      onCancel={handleClose}
      footer={
        <Space>
          <Button onClick={handleClose}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={() => void handleSubmit()} loading={uploadMut.isPending}>
            {t('common.upload')}
          </Button>
        </Space>
      }
      width={560}
      destroyOnHidden
    >
      <Form layout="vertical">
        <Form.Item
          label={t('product.alarm.upload.xmlFile')}
          required
          validateStatus={fileError ? 'error' : undefined}
          help={fileError}
        >
          <Upload.Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">{t('product.alarm.upload.dropHint')}</p>
            <p className="ant-upload-hint">
              {t('product.alarm.upload.sizeHintPre')}<code>&lt;alarmModel&gt;</code>{t('product.alarm.upload.sizeHintPost')}
              <br />
              {t('product.alarm.upload.nameAutoHint')}
            </p>
          </Upload.Dragger>
          {derivedName && (
            <Typography.Text type="secondary" style={{ display: 'block', marginTop: 8 }}>
              {t('product.upload.savedAs', { name: derivedName })}
            </Typography.Text>
          )}
        </Form.Item>
      </Form>
    </Modal>
  );
}
