/**
 * UploadXmlModal — T-0180 P4 自定义 XML 上传弹窗。
 *
 * 表单字段:
 *   - 目标制式 *  (ENB / GSM / GNB,Select,与 ?tech= query 对应)
 *   - XML 文件 *  (antd Upload Dragger,单文件 ≤ 1 MiB,.xml only)
 *
 * 校验链(后端 file_handler.UploadXML 已守:文件名 + size + tech +
 *   <indicatorModel> 根 + platform 必填 + deviceType 匹配 tech);
 * 前端仅做 size/类型提示,真校验留后端响应。
 *
 * 409 冲突 → 弹二次确认走 force=true 重试;后端备份为 .bak.<ts>。
 */
import { useState } from 'react';
import { Modal, Form, Select, Upload, message, Button, Space, Tag } from 'antd';
import type { UploadFile, UploadProps } from 'antd/es/upload/interface';
import { InboxOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import { useIndicatorUploadXml } from '@core/hooks/api/useIndicatorsLibrary';
import type { TechLower } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

const MAX_SIZE = 1 * 1024 * 1024; // 1 MiB,与后端 MaxUploadXMLSize 一致

const TECH_OPTIONS: { label: string; value: TechLower }[] = [
  { label: 'ENB (LTE)', value: 'enb' },
  { label: 'GSM', value: 'gsm' },
  { label: 'GNB (5G NR)', value: 'gnb' },
];

interface FormValues {
  tech?: TechLower;
}

// 2026-05-29:从 XML 头部 4 KB 抽 <indicatorModel deviceType="..."> 属性。
// 用途:beforeUpload 阶段自动同步表单"目标制式",避免用户手选 ENB 但传 GNB
// 文件导致后端 2044 (ErrCodeIndicatorUploadInvalidRoot) 拒收的来回。
// 容错:正则不匹配 / deviceType 不在三制式白名单 → 返 null,沿用用户当前选项。
async function detectTechFromXml(file: File): Promise<TechLower | null> {
  try {
    const head = await file.slice(0, 4096).text();
    const m = head.match(/<indicatorModel[^>]*\bdeviceType\s*=\s*"([^"]+)"/i);
    if (!m) return null;
    const t = m[1].toLowerCase() as TechLower;
    return (['enb', 'gsm', 'gnb'] as TechLower[]).includes(t) ? t : null;
  } catch {
    return null;
  }
}

export default function UploadXmlModal({ open, onClose }: Props) {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const uploadMut = useIndicatorUploadXml();

  const reset = () => {
    form.resetFields();
    setFileList([]);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const doUpload = async (tech: TechLower, file: File, force: boolean) => {
    try {
      const r = await uploadMut.mutateAsync({ tech, file, force });
      message.success(
        r.overwrite && r.backup
          ? t('product.kpi.upload.overrideSuccess', { file: r.filename, backup: r.backup ?? '' })
          : t('product.kpi.upload.uploadSuccess', { file: r.filename })
      );
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ message?: string }>;
      // 409 → 弹覆盖确认,force=true 重试
      if (ax.response?.status === 409 && !force) {
        Modal.confirm({
          title: t('product.kpi.upload.overrideTitle'),
          content: (
            <div style={{ maxWidth: 360 }}>
              {t('product.kpi.upload.overrideContentPre')}<code>{file.name}</code>{t('product.kpi.upload.overrideContentPost')}
              <br />{t('product.kpi.upload.backupHint')}
            </div>
          ),
          okText: t('common.override'),
          okButtonProps: { danger: true },
          onOk: () => doUpload(tech, file, true),
        });
        return;
      }
      // 后端信封字段名是 msg(非 message);message 保留兼容老链路。
      const body = ax.response?.data as { msg?: string; message?: string } | undefined;
      const msg = body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e));
      message.error(msg);
    }
  };

  const handleSubmit = async () => {
    try {
      const v = await form.validateFields();
      if (!v.tech) return;
      if (fileList.length === 0 || !fileList[0].originFileObj) {
        message.error(t('product.kpi.upload.selectXmlMsg'));
        return;
      }
      await doUpload(v.tech, fileList[0].originFileObj, false);
    } catch {
      /* form validation 失败,antd 自动显示错误,无需额外处理 */
    }
  };

  const uploadProps: UploadProps = {
    accept: '.xml',
    maxCount: 1,
    fileList,
    beforeUpload: (f) => {
      if (f.size > MAX_SIZE) {
        message.error(t('product.kpi.upload.fileTooLarge', { size: (f.size / 1024).toFixed(1) }));
        return Upload.LIST_IGNORE;
      }
      // 2026-05-29:选文件即抽 deviceType,自动同步"目标制式",避免后端 2044
      // (e.g. ENB 上下文选 GNB.xml 必拒)。fire-and-forget,不阻塞 UI。
      void detectTechFromXml(f).then((detected) => {
        if (!detected) return;
        const current = form.getFieldValue('tech') as TechLower | undefined;
        if (!current) {
          form.setFieldsValue({ tech: detected });
        } else if (current !== detected) {
          form.setFieldsValue({ tech: detected });
          message.info(t('product.kpi.upload.autoSyncTech', { tech: detected.toUpperCase() }));
        }
      });
      return false; // 阻止 antd 自动上传 — 我们走 customRequest in handleSubmit
    },
    onChange: ({ fileList: newList }) => setFileList(newList.slice(-1)),
    onRemove: () => setFileList([]),
  };

  return (
    <Modal
      title={
        <Space>
          <span>{t('product.kpi.upload.title')}</span>
          <Tag color="blue">indicator-library-custom</Tag>
        </Space>
      }
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
      <Form<FormValues> form={form} layout="vertical" requiredMark>
        <Form.Item
          name="tech"
          label={t('product.kpi.upload.targetTech')}
          rules={[{ required: true, message: t('product.kpi.upload.techRequired') }]}
        >
          <Select options={TECH_OPTIONS} placeholder={t('product.kpi.upload.techPh')} />
        </Form.Item>
        <Form.Item label={t('product.kpi.upload.xmlFile')} required>
          <Upload.Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">{t('product.kpi.upload.dropHint')}</p>
            <p className="ant-upload-hint">
              {t('product.kpi.upload.sizeHintPre')}<code>&lt;indicatorModel&gt;</code>{t('product.kpi.upload.sizeHintPost')}
              <br />
              {t('product.kpi.upload.platformHint')}
            </p>
          </Upload.Dragger>
        </Form.Item>
      </Form>
    </Modal>
  );
}
