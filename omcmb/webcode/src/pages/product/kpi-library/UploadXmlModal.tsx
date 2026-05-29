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

export default function UploadXmlModal({ open, onClose }: Props) {
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
          ? `已覆盖上传:${r.filename}(旧版备份 ${r.backup})`
          : `已上传:${r.filename}`
      );
      handleClose();
    } catch (e: unknown) {
      const ax = e as AxiosError<{ message?: string }>;
      // 409 → 弹覆盖确认,force=true 重试
      if (ax.response?.status === 409 && !force) {
        Modal.confirm({
          title: '同名 XML 已存在',
          content: (
            <div style={{ maxWidth: 360 }}>
              检测到 <code>{file.name}</code> 已存在,确认覆盖?
              <br />旧文件会自动备份为 <code>.bak.&lt;ts&gt;</code>。
            </div>
          ),
          okText: '覆盖',
          okButtonProps: { danger: true },
          onOk: () => doUpload(tech, file, true),
        });
        return;
      }
      const msg = ax.response?.data?.message ?? (e instanceof Error ? e.message : String(e));
      message.error(msg);
    }
  };

  const handleSubmit = async () => {
    try {
      const v = await form.validateFields();
      if (!v.tech) return;
      if (fileList.length === 0 || !fileList[0].originFileObj) {
        message.error('请选择 XML 文件');
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
        message.error(`文件超过 1 MiB(实际 ${(f.size / 1024).toFixed(1)} KiB)`);
        return Upload.LIST_IGNORE;
      }
      return false; // 阻止 antd 自动上传 — 我们走 customRequest in handleSubmit
    },
    onChange: ({ fileList: newList }) => setFileList(newList.slice(-1)),
    onRemove: () => setFileList([]),
  };

  return (
    <Modal
      title={
        <Space>
          <span>上传自定义 XML</span>
          <Tag color="blue">indicator-library-custom</Tag>
        </Space>
      }
      open={open}
      onCancel={handleClose}
      footer={
        <Space>
          <Button onClick={handleClose}>取消</Button>
          <Button type="primary" onClick={() => void handleSubmit()} loading={uploadMut.isPending}>
            上传
          </Button>
        </Space>
      }
      width={560}
      destroyOnHidden
    >
      <Form<FormValues> form={form} layout="vertical" requiredMark>
        <Form.Item
          name="tech"
          label="目标制式"
          rules={[{ required: true, message: '请选择目标制式' }]}
        >
          <Select options={TECH_OPTIONS} placeholder="选择 ENB / GSM / GNB" />
        </Form.Item>
        <Form.Item label="XML 文件" required>
          <Upload.Dragger {...uploadProps}>
            <p className="ant-upload-drag-icon">
              <InboxOutlined />
            </p>
            <p className="ant-upload-text">拖拽 .xml 到此处或点击选择</p>
            <p className="ant-upload-hint">
              单文件 ≤ 1 MiB;根元素必须为 <code>&lt;indicatorModel&gt;</code>;
              <br />
              <code>platform</code> 属性必填,<code>deviceType</code> 若有必须匹配目标制式。
            </p>
          </Upload.Dragger>
        </Form.Item>
      </Form>
    </Modal>
  );
}
