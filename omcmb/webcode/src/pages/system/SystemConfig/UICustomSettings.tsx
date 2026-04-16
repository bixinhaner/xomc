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
    void message.info(t('system.ui.previewThemeColor'));
    // 实际实现中可以动态修改CSS变量
  };

  const handleRestore = () => {
    form.setFieldsValue(defaultUIConfig);
    setLoginBgFile([]);
    setLogoSmallFile([]);
    setLogoBigFile([]);
    void message.success(t('system.ui.restoredDefault'));
  };

  const beforeUploadBg = (file: RcFile) => {
    const isLt1M = file.size / 1024 / 1024 < 1;
    if (!isLt1M) {
      void message.error(t('system.ui.loginBgSizeExceeded'));
      return false;
    }
    setLoginBgFile([file]);
    return false;
  };

  const beforeUploadLogo = (file: RcFile) => {
    const isLt400K = file.size / 1024 < 400;
    if (!isLt400K) {
      void message.error(t('system.ui.logoSizeExceeded'));
      return false;
    }
    return false;
  };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={defaultUIConfig}>
      <Row gutter={24}>
        <Col span={12}>
          <Form.Item name="ui_omc_name" label={t('system.ui.omcName')} rules={[{ required: true, message: t('system.ui.pleaseInputOmcName') }]}>
            <Input placeholder={t('system.ui.pleaseInputSystemName')} />
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name="ui_color" label={t('system.ui.themeColor')} rules={[{ required: true }]}>
            <Space>
              <ColorPicker format="hex" />
              <Button icon={<EyeOutlined />} onClick={handlePreview}>
                {t('common.preview')}
              </Button>
            </Space>
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left" plain>{t('system.ui.imageUpload')}</Divider>

      <Row gutter={24}>
        <Col span={8}>
          <Card size="small" title={t('system.ui.loginBackground')} extra={<span style={{ color: '#999', fontSize: 12 }}>{t('system.ui.max1mb')}</span>}>
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
                <p className="ant-upload-text">{t('common.clickOrDragToUpload')}</p>
                <p className="ant-upload-hint">{t('common.supportJpgPng')}</p>
              </Dragger>
            </Form.Item>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={t('system.ui.logoSmall')} extra={<span style={{ color: '#999', fontSize: 12 }}>{t('system.ui.max400kb')}</span>}>
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
                <p className="ant-upload-text">{t('system.ui.showWhenMenuCollapsed')}</p>
              </Dragger>
            </Form.Item>
          </Card>
        </Col>
        <Col span={8}>
          <Card size="small" title={t('system.ui.logoLarge')} extra={<span style={{ color: '#999', fontSize: 12 }}>{t('system.ui.max400kb')}</span>}>
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
                <p className="ant-upload-text">{t('system.ui.showWhenMenuExpanded')}</p>
              </Dragger>
            </Form.Item>
          </Card>
        </Col>
      </Row>

      <Divider />

      <Space>
        <Popconfirm
          title={t('system.ui.confirmRestoreDefault')}
          description={t('system.ui.restoreWillOverwrite')}
          onConfirm={handleRestore}
          okText={t('common.confirm')}
          cancelText={t('common.cancel')}
        >
          <Button icon={<UndoOutlined />}>
            {t('system.ui.restoreDefault')}
          </Button>
        </Popconfirm>
      </Space>
    </Form>
  );
}
