/**
 * UploadXmlModal — 自定义 XML 上传弹窗(导入 XML 调整,2026-06-05)。
 *
 * 表单字段:
 *   - 目标制式 *  (ENB / GSM / GNB,Select,与 ?tech= query 对应;选文件后按 deviceType 自动同步)
 *   - XML 文件 *  (antd Upload Dragger,单文件 ≤ 1 MiB,.xml only)
 *
 * 名称取消手填:唯一名称取自 XML <indicatorModel platform="..."> 属性,
 * 文件将保存为 <platform>.xml(选文件后即时预览)。
 *
 * 重复允许覆盖(2026-06-05 调整,二次确认):
 *   - 本地预检(useIndicatorFiles(tech) 文件名比对)或后端 409(data.overwritable=true)
 *     → Modal.confirm「已存在,确认覆盖?」→ 确认后带 force=true 重试,
 *     后端覆盖归属文件并自动备份旧文件 .bak.<ts>。
 *
 * 落地目录:ENB → indicator-library/enb/;GSM/GNB → indicator-library/ 根级
 * (GSM/GNB 的 XML 必须带 deviceType 属性,后端据此分类制式)。
 */
import { useState } from 'react';
import { Modal, Form, Select, Upload, message, Button, Space, Typography } from 'antd';
import type { UploadFile, UploadProps } from 'antd/es/upload/interface';
import { InboxOutlined } from '@ant-design/icons';
import type { AxiosError } from 'axios';
import { useIndicatorUploadXml, useIndicatorFiles } from '@core/hooks/api/useIndicatorsLibrary';
import { useTechnologyDictionary } from '@core/hooks/api/useTechnologyDictionary';
import type { TechLower } from '@core/types/indicatorLibrary';
import { extractXmlRootAttr } from '@core/utils/xmlRootAttr';
import { useT } from '@/hooks/useT';

interface Props {
  open: boolean;
  onClose: () => void;
}

const MAX_SIZE = 1 * 1024 * 1024; // 1 MiB,与后端 MaxUploadXMLSize 一致

const FALLBACK_TECH_OPTIONS: { label: string; value: TechLower }[] = [
  { label: 'eNB (LTE)', value: 'enb' },
  { label: 'gNB (NR)', value: 'gnb' },
  { label: 'GSM', value: 'gsm' },
];

const DEVICE_TYPE_TO_TECH: Record<string, TechLower> = {
  ENB: 'enb',
  GNB: 'gnb',
  GSM: 'gsm',
};

interface FormValues {
  tech?: TechLower;
}

const TECH_WHITELIST: TechLower[] = ['enb', 'gsm', 'gnb'];

export default function UploadXmlModal({ open, onClose }: Props) {
  const t = useT();
  const [form] = Form.useForm<FormValues>();
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  // 选文件后从 XML platform 属性提取的名称(将保存为 <platform>.xml);null = 未提取到
  const [derivedName, setDerivedName] = useState<string | null>(null);
  const [fileError, setFileError] = useState<string | undefined>();
  const uploadMut = useIndicatorUploadXml();
  const { deviceTypeOptions, isLoading: techOptionsLoading } = useTechnologyDictionary();
  const techOptions =
    deviceTypeOptions.length > 0
      ? deviceTypeOptions.map((option) => ({
          label: option.label,
          value: DEVICE_TYPE_TO_TECH[option.value],
        })).filter((option): option is { label: string; value: TechLower } => Boolean(option.value))
      : FALLBACK_TECH_OPTIONS;

  // 上传前查重数据源:按当前所选制式取已存在的 XML 文件清单(loadedFrom)。
  // 文件名比对走 <platform>.xml basename 大小写不敏感,与后端同名判定口径一致。
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
    setDerivedName(null);
    setFileError(undefined);
  };

  const handleClose = () => {
    reset();
    onClose();
  };

  // 实际上传(force = 二次确认后的覆盖);成功后关弹窗,失败统一 message。
  const doUpload = async (tech: TechLower, file: File, force: boolean) => {
    try {
      const r = await uploadMut.mutateAsync({ tech, file, force });
      message.success(
        r.overwritten
          ? t('product.upload.overwriteSuccess', { file: r.filename })
          : t('product.kpi.upload.uploadSuccess', { file: r.filename }),
      );
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ msg?: string; message?: string; data?: { overwritable?: boolean } }>;
      const body = ax.response?.data;
      const msg = body?.msg ?? body?.message ?? (e instanceof Error ? e.message : String(e));
      // 409 + overwritable(本地预检漏判的重复,如内容主键归属其它文件)→ 弹确认覆盖
      if (!force && ax.response?.status === 409 && body?.data?.overwritable) {
        confirmOverwrite(derivedName ?? '', () => void doUpload(tech, file, true));
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
    let v: FormValues;
    try {
      v = await form.validateFields();
    } catch {
      /* form validation 失败,antd 自动显示错误,无需额外处理 */
      return;
    }
    if (!v.tech) return;
    if (fileList.length === 0 || !fileList[0].originFileObj) {
      message.error(t('product.kpi.upload.selectXmlMsg'));
      return;
    }
    // platform 属性是名称唯一来源 —— 读不到直接拒绝(后端也会 400)
    if (!derivedName) {
      setFileError(t('product.upload.missingAttr', { attr: 'platform' }));
      return;
    }
    const file = fileList[0].originFileObj;
    const tech = v.tech;
    // 同名预检 → 直接弹覆盖确认(后端 409 仍是兜底真值源)
    if (isDuplicateName(derivedName)) {
      confirmOverwrite(derivedName, () => void doUpload(tech, file, true));
      return;
    }
    await doUpload(tech, file, false);
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
      setFileError(undefined);
      // 选文件即抽 platform(名称预览)与 deviceType(自动同步"目标制式",
      // 避免后端 2044 拒收的来回)。fire-and-forget,不阻塞 UI。
      void extractXmlRootAttr(f, 'platform').then((p) => setDerivedName(p));
      void extractXmlRootAttr(f, 'deviceType').then((dt) => {
        const detected = dt?.toLowerCase() as TechLower | undefined;
        if (!detected || !TECH_WHITELIST.includes(detected)) return;
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
    onRemove: () => {
      setFileList([]);
      setDerivedName(null);
      setFileError(undefined);
    },
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
          name="tech"
          label={t('product.kpi.upload.targetTech')}
          rules={[{ required: true, message: t('product.kpi.upload.techRequired') }]}
        >
          <Select options={techOptions} loading={techOptionsLoading} placeholder={t('common.pleaseSelect')} />
        </Form.Item>
        <Form.Item
          label={t('product.kpi.upload.xmlFile')}
          required
          validateStatus={fileError ? 'error' : undefined}
          help={fileError}
        >
          <Upload.Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">{t('product.kpi.upload.dropHint')}</p>
            <p className="ant-upload-hint">
              {t('product.kpi.upload.sizeHintPre')}<code>&lt;indicatorModel&gt;</code>{t('product.kpi.upload.sizeHintPost')}
              <br />
              {t('product.kpi.upload.nameAutoHint')}
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
