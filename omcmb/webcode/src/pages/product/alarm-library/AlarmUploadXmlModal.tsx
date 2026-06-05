/**
 * AlarmUploadXmlModal — 告警库自定义 XML 上传弹窗(对标 kpi-library UploadXmlModal)。
 *
 * 2026-06-04 单目录 + 双唯一性硬拒(无 force):
 *   - 名称 *      (用户指定的唯一名称,正则 ^[A-Za-z0-9_-]{1,64}$,落地 <name>.xml)
 *   - XML 文件 *  (根元素必须 <alarmModel>,≤ 1 MiB)
 *   上传前用 useAlarmNeTypeStats 取已存在的 XML 文件名,若 <name>.xml 重复则内联报错
 *   "名称已存在,请改名";后端 409 兜底同样内联提示改名(文件名 / neType 内容主键任一冲突)。
 *
 * 校验链(后端 file_handler.UploadXML 已守:name 白名单 + size + <alarmModel> 根 + 双唯一性)。
 */
import { useState } from 'react';
import { Modal, Form, Input, Upload, message, Button, Space } from 'antd';
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
const NAME_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

interface UploadFormValues {
  name?: string;
}

export default function AlarmUploadXmlModal({ open, onClose }: Props) {
  const t = useT();
  const [form] = Form.useForm<UploadFormValues>();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const uploadMut = useAlarmUploadXml();

  // 上传前查重:ne-types 聚合行的 loadedFrom basename(大小写不敏感)对比 <name>.xml。
  const { data: neTypesData } = useAlarmNeTypeStats();
  const isDuplicateName = (name: string): boolean => {
    const target = `${name}.xml`.toLowerCase();
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

  const handleSubmit = async () => {
    const v = await form.validateFields().catch(() => null);
    if (!v?.name) return;
    if (fileList.length === 0 || !fileList[0].originFileObj) {
      message.error(t('product.alarm.upload.selectXmlMsg'));
      return;
    }
    const file = fileList[0].originFileObj;

    // 上传前查重:文件名已存在 → 内联报错到"名称"字段,提示改名(409 仍作兜底)。
    if (isDuplicateName(v.name)) {
      form.setFields([{ name: 'name', errors: [t('product.paramModel.nameExists')] }]);
      return;
    }

    try {
      const r = await uploadMut.mutateAsync({ file, name: v.name });
      message.success(t('product.alarm.upload.uploadSuccess', { file: r.filename }));
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ msg?: string; message?: string }>;
      // 409 冲突(文件名或 neType 内容主键)→ 内联报错到"名称"字段,提示改名。
      if (ax.response?.status === 409) {
        form.setFields([{ name: 'name', errors: [t('product.paramModel.nameExists')] }]);
        return;
      }
      const body = ax.response?.data as { msg?: string; message?: string } | undefined;
      message.error(body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e)));
    }
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
          <Button type="primary" onClick={() => void handleSubmit()} loading={uploadMut.isPending}>
            {t('common.upload')}
          </Button>
        </Space>
      }
      width={560}
      destroyOnHidden
    >
      <Form form={form} layout="vertical">
        <Form.Item
          name="name"
          label={t('common.name')}
          rules={[
            { required: true, message: t('common.nameRequired') },
            { pattern: NAME_PATTERN, message: t('product.paramModel.nameInvalid') },
          ]}
        >
          <Input placeholder="ENB" maxLength={64} allowClear />
        </Form.Item>
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
