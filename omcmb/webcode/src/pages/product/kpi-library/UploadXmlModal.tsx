/**
 * UploadXmlModal — 自定义 XML 上传弹窗(三库 XML 导入重构 Slice)。
 *
 * 表单字段:
 *   - 名称 *      (用户指定的唯一名称,正则 ^[A-Za-z0-9_-]{1,64}$,落地 <name>.xml)
 *   - 目标制式 *  (ENB / GSM / GNB,Select,与 ?tech= query 对应)
 *   - XML 文件 *  (antd Upload Dragger,单文件 ≤ 1 MiB,.xml only)
 *
 * 双唯一性硬拒(无 force):
 *   - 文件名唯一:上传前用 useIndicatorFiles(tech) 取该制式已存在文件名,同名 → 内联
 *     "名称已存在,请改名";后端 409 兜底同样内联提示改名。
 *   - 内容主键唯一:platform 已存在 → 后端 409,内联展示后端 msg。
 *
 * 校验链(后端 file_handler.UploadXML 已守:name 白名单 + size + tech +
 *   <indicatorModel> 根 + platform 必填 + deviceType 匹配 tech + 双唯一性);
 * 前端做 size/类型/名称正则/同名预检提示,真校验留后端响应。
 */
import { useState } from 'react';
import { Modal, Form, Input, Select, Upload, message, Button, Space } from 'antd';
import type { UploadFile, UploadProps } from 'antd/es/upload/interface';
import { InboxOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import { useIndicatorUploadXml, useIndicatorFiles } from '@core/hooks/api/useIndicatorsLibrary';
import type { TechLower } from '@core/types/indicatorLibrary';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

const MAX_SIZE = 1 * 1024 * 1024; // 1 MiB,与后端 MaxUploadXMLSize 一致
const NAME_PATTERN = /^[A-Za-z0-9_-]{1,64}$/;

const TECH_OPTIONS: { label: string; value: TechLower }[] = [
  { label: 'ENB (LTE)', value: 'enb' },
  { label: 'GSM', value: 'gsm' },
  { label: 'GNB (5G NR)', value: 'gnb' },
];

interface FormValues {
  name?: string;
  tech?: TechLower;
}

// 从 XML 头部 4 KB 抽 <indicatorModel deviceType="..."> 属性。
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

  // 上传前查重数据源:按当前所选制式取已存在的 XML 文件清单(loadedFrom)。
  // 文件名比对走 <name>.xml basename 大小写不敏感,与后端同名判定口径一致。
  const watchedTech = Form.useWatch('tech', form);
  const { data: filesData } = useIndicatorFiles(open ? watchedTech : undefined);
  const isDuplicateName = (name: string): boolean => {
    const target = `${name}.xml`.toLowerCase();
    return (filesData?.items || []).some((f) => {
      const base = f.loadedFrom.split('/').pop()?.toLowerCase() ?? '';
      return base === target;
    });
  };

  const reset = () => {
    form.resetFields();
    setFileList([]);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  const handleSubmit = async () => {
    let v: FormValues;
    try {
      v = await form.validateFields();
    } catch {
      /* form validation 失败,antd 自动显示错误,无需额外处理 */
      return;
    }
    if (!v.name || !v.tech) return;
    if (fileList.length === 0 || !fileList[0].originFileObj) {
      message.error(t('product.kpi.upload.selectXmlMsg'));
      return;
    }
    // 同名预检 → 内联报错到"名称"字段(不弹覆盖确认,无 force)
    if (isDuplicateName(v.name)) {
      form.setFields([{ name: 'name', errors: [t('product.kpi.upload.nameExists')] }]);
      return;
    }
    const file = fileList[0].originFileObj;
    try {
      const r = await uploadMut.mutateAsync({ tech: v.tech, file, name: v.name });
      message.success(t('product.kpi.upload.uploadSuccess', { file: r.filename }));
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ msg?: string; message?: string }>;
      const body = ax.response?.data;
      const msg = body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e));
      // 409 冲突 → 内联报错到"名称"字段,提示改名;其余错误用 message.error
      if (ax.response?.status === 409) {
        form.setFields([{ name: 'name', errors: [t('product.kpi.upload.nameExists')] }]);
        return;
      }
      message.error(msg);
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
      // 选文件即抽 deviceType,自动同步"目标制式",避免后端 2044
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
      return false; // 阻止 antd 自动上传 — 我们走 handleSubmit
    },
    onChange: ({ fileList: newList }) => setFileList(newList.slice(-1)),
    onRemove: () => setFileList([]),
  };

  return (
    <Modal
      title={t('product.kpi.upload.title')}
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
          name="name"
          label={t('product.kpi.upload.name')}
          rules={[
            { required: true, message: t('product.kpi.upload.nameRequired') },
            { pattern: NAME_PATTERN, message: t('product.kpi.upload.namePattern') },
          ]}
        >
          <Input placeholder={t('product.kpi.upload.namePh')} maxLength={64} />
        </Form.Item>
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
