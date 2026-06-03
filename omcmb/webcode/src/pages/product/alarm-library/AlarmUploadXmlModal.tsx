/**
 * AlarmUploadXmlModal — 告警库自定义 XML 上传弹窗(严格对标 kpi-library UploadXmlModal;
 * 告警侧无"目标制式",ne_type 由 XML 内容 / 文件名在加载时推断,故只需选文件)。
 *
 * 校验链(后端 file_handler.UploadXML 已守:文件名 + size + <alarmModel> 根);
 * 前端仅做 size/类型提示。
 * 上传前查重(2026-06-03):真正上传前先用 useAlarmNeTypeStats 取已存在的 XML 文件名,
 *   若文件名重复则弹覆盖确认 → force=true 上传;不重复则直接上传(409 仍作兜底)。
 */
import { useState } from 'react';
import { Modal, Form, Upload, message, Button, Space } from 'antd';
import type { UploadFile, UploadProps } from 'antd/es/upload/interface';
import { InboxOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import { useAlarmUploadXml, useAlarmNeTypeStats } from '@core/hooks/api/useAlarmDefinitions';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

const MAX_SIZE = 1 * 1024 * 1024; // 1 MiB,与后端 MaxUploadXMLSize 一致

export default function AlarmUploadXmlModal({ open, onClose }: Props) {
  const t = useT();
  const [form] = Form.useForm();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const uploadMut = useAlarmUploadXml();

  // 上传前查重数据源:ne-types 聚合行的 loadedFrom basename(大小写不敏感)。
  const { data: neTypesData } = useAlarmNeTypeStats();
  const isDuplicate = (fileName: string): boolean => {
    const target = fileName.toLowerCase();
    return (neTypesData?.items || []).some((row) => {
      const base = (row.loadedFrom || '').split('/').pop()?.toLowerCase() ?? '';
      return base === target;
    });
  };

  const handleClose = () => {
    form.resetFields();
    setFileList([]);
    onClose();
  };

  // 弹覆盖确认(查重命中 / 后端 409 兜底共用),确认后带 force=true 上传。
  const confirmOverride = (file: File) => {
    Modal.confirm({
      title: t('product.alarm.upload.overrideTitle'),
      content: (
        <div style={{ maxWidth: 360 }}>
          {t('product.alarm.upload.overrideContentPre')}<code>{file.name}</code>{t('product.alarm.upload.overrideContentPost')}
          <br />{t('product.alarm.upload.backupHint')}
        </div>
      ),
      okText: t('common.override'),
      okButtonProps: { danger: true },
      onOk: () => doUpload(file, true),
    });
  };

  const doUpload = async (file: File, force: boolean) => {
    try {
      const r = await uploadMut.mutateAsync({ file, force });
      message.success(
        r.overwrite && r.backup
          ? t('product.alarm.upload.overrideSuccess', { file: r.filename, backup: r.backup })
          : t('product.alarm.upload.uploadSuccess', { file: r.filename })
      );
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ msg?: string; message?: string }>;
      if (ax.response?.status === 409 && !force) {
        confirmOverride(file);
        return;
      }
      const body = ax.response?.data as { msg?: string; message?: string } | undefined;
      message.error(body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e)));
    }
  };

  const handleSubmit = () => {
    if (fileList.length === 0 || !fileList[0].originFileObj) {
      message.error(t('product.alarm.upload.selectXmlMsg'));
      return;
    }
    const file = fileList[0].originFileObj;
    // 上传前查重:文件名已存在 → 先弹覆盖确认;否则直接上传(409 仍作兜底)。
    if (isDuplicate(file.name)) {
      confirmOverride(file);
      return;
    }
    void doUpload(file, false);
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
      return false; // 阻止 antd 自动上传 — 走 handleSubmit
    },
    onChange: ({ fileList: newList }) => setFileList(newList.slice(-1)),
    onRemove: () => setFileList([]),
  };

  return (
    <Modal
      title={t('product.alarm.upload.title')}
      open={open}
      onCancel={handleClose}
      footer={
        <Space>
          <Button onClick={handleClose}>{t('common.cancel')}</Button>
          <Button type="primary" onClick={handleSubmit} loading={uploadMut.isPending}>
            {t('common.upload')}
          </Button>
        </Space>
      }
      width={560}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item label={t('product.alarm.upload.xmlFile')} required>
          <Upload.Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">{t('product.alarm.upload.dropHint')}</p>
            <p className="ant-upload-hint">
              {t('product.alarm.upload.sizeHintPre')}<code>&lt;alarmModel&gt;</code>{t('product.alarm.upload.sizeHintPost')}
            </p>
          </Upload.Dragger>
        </Form.Item>
      </Form>
    </Modal>
  );
}
