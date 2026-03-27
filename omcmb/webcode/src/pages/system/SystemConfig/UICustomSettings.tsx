import { useState } from 'react';
import { Form, Input, ColorPicker, Upload, Button, Divider, Space, message, Popconfirm, Row, Col, Card } from 'antd';
import { InboxOutlined, EyeOutlined, UndoOutlined } from '@ant-design/icons';
import type { UploadFile, RcFile } from 'antd/es/upload';
import { useT } from '@/hooks/useT';

const { Dragger } = Upload;

interface UICustomSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 默认值
const defaultUIConfig = {
  ui_omc_name: 'BaiOMC',
  ui_color: '#FF4614',
  ui_login_background: './images/login/login_bg.png',
  ui_menu_logo_up: './images/login/nav_logo_collapse.png',
  ui_menu_logo_down: './images/login/logo_big.png',
};

export default function UICustomSettings({ form }: UICustomSettingsProps) {
  const t = useT();
  const [loginBgFile, setLoginBgFile] = useState<UploadFile[]>([]);
  const [logoSmallFile, setLogoSmallFile] = useState<UploadFile[]>([]);
  const [logoBigFile, setLogoBigFile] = useState<UploadFile[]>([]);

  const handlePreview = () => {
    void message.info('预览主题色效果');
    // 实际实现中可以动态修改CSS变量
  };

  const handleRestore = () => {
    form.setFieldsValue(defaultUIConfig);
    setLoginBgFile([]);
    setLogoSmallFile([]);
    setLogoBigFile([]);
    void message.success('已恢复默认UI设置');
  };

  const beforeUploadBg = (file: RcFile) => {
    const isLt1M = file.size / 1024 / 1024 < 1;
    if (!isLt1M) {
      void message.error('登录背景图片大小不能超过1MB');
      return false;
    }
    setLoginBgFile([file]);
    return false;
  };

  const beforeUploadLogo = (file: RcFile) => {
    const isLt400K = file.size / 1024 < 400;
    if (!isLt400K) {
      void message.error('Logo图片大小不能超过400KB');
      return false;
    }
    return false;
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={defaultUIConfig}>
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item name="ui_omc_name" label="OMC名称" rules={[{ required: true, message: '请输入OMC名称' }]}>
            <Input placeholder="请输入系统名称，显示在浏览器标题" />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="ui_color" label="主题色" rules={[{ required: true }]}>
            <Space>
              <ColorPicker format="hex" />
              <Button icon={<EyeOutlined />} onClick={handlePreview}>
                预览
              </Button>
            </Space>
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left" plain>图片上传</Divider>

      <Row gutter={24}>
        <Col span={8}>
          <Card size="small" title="登录背景" extra={<span style={{ color: '#999', fontSize: 12 }}>最大1MB</span>}>
            <Form.Item name="ui_login_background">
              <Dragger
                fileList={loginBgFile}
                beforeUpload={beforeUploadBg}
                onRemove={() => setLoginBgFile([])}
                maxCount={1}
                accept="image/jpeg,image/jpg,image/png"
              >
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">点击或拖拽上传</p>
                <p className="ant-upload-hint">支持 JPG / PNG 格式</p>
              </Dragger>
            </Form.Item>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title="Logo小图" extra={<span style={{ color: '#999', fontSize: 12 }}>最大400KB</span>}>
            <Form.Item name="ui_menu_logo_up">
              <Dragger
                fileList={logoSmallFile}
                beforeUpload={(file) => {
                  if (beforeUploadLogo(file) === false) return false;
                  setLogoSmallFile([file]);
                  return false;
                }}
                onRemove={() => setLogoSmallFile([])}
                maxCount={1}
                accept="image/jpeg,image/jpg,image/png"
              >
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">菜单收起时显示</p>
              </Dragger>
            </Form.Item>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title="Logo大图" extra={<span style={{ color: '#999', fontSize: 12 }}>最大400KB</span>}>
            <Form.Item name="ui_menu_logo_down">
              <Dragger
                fileList={logoBigFile}
                beforeUpload={(file) => {
                  if (beforeUploadLogo(file) === false) return false;
                  setLogoBigFile([file]);
                  return false;
                }}
                onRemove={() => setLogoBigFile([])}
                maxCount={1}
                accept="image/jpeg,image/jpg,image/png"
              >
                <p className="ant-upload-drag-icon">
                  <InboxOutlined />
                </p>
                <p className="ant-upload-text">菜单展开时显示</p>
              </Dragger>
            </Form.Item>
          </Card>
        </Col>
      </Row>

      <Divider />

      <Space>
        <Popconfirm
          title="确定恢复默认UI设置？"
          description="此操作将覆盖当前设置"
          onConfirm={handleRestore}
          okText="确定"
          cancelText="取消"
        >
          <Button icon={<UndoOutlined />}>
            恢复默认
          </Button>
        </Popconfirm>
      </Space>
    </Form>
  );
}
