import {
  App,
  Form,
  Input,
  Upload,
  Button,
  Divider,
  Space,
  Popconfirm,
  Row,
  Col,
  Card,
} from 'antd';
import { InboxOutlined, UndoOutlined } from '@ant-design/icons';
import type { RcFile } from 'antd/es/upload';
import { useT } from '@/hooks/useT';
import { useUploadUIAsset } from '@core/hooks/api/useSystem';
import {
  UI_CUSTOM_DEFAULTS,
  SIZE_LOGIN_BG,
  SIZE_LOGO,
  ACCEPT_IMAGE_TYPES,
  type UIAssetKind,
} from './uiCustomConstants';

const { Dragger } = Upload;

interface UICustomSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
  initialValues?: Record<string, string>;
  onRestoreDefaults: () => void;
  restoring?: boolean;
}

export default function UICustomSettings({
  form,
  initialValues,
  onRestoreDefaults,
  restoring = false,
}: UICustomSettingsProps) {
  const t = useT();
  const { message } = App.useApp();
  const uploadAsset = useUploadUIAsset();

  // 三个上传位的预览 URL 直接订阅 form 字段值——无需本地 state，也避开 set-state-in-effect。
  const loginBgUrl = (Form.useWatch('ui_login_background', form) as string | undefined) ?? '';
  const logoSmallUrl = (Form.useWatch('ui_menu_logo_up', form) as string | undefined) ?? '';
  const logoLargeUrl = (Form.useWatch('ui_menu_logo_down', form) as string | undefined) ?? '';

  // 通用上传 before-upload 钩子：体积/类型预校验通过 → 调后端上传 → 写回 form 字段。
  const buildBeforeUpload = (key: string, kind: UIAssetKind, sizeLimit: number) => {
    return (file: RcFile): boolean => {
      if (!ACCEPT_IMAGE_TYPES.includes(file.type as (typeof ACCEPT_IMAGE_TYPES)[number])) {
        void message.error(t('system.ui.invalidImageType'));
        return false;
      }
      if (file.size > sizeLimit) {
        void message.error(
          kind === 'login_bg' ? t('system.ui.loginBgSizeExceeded') : t('system.ui.logoSizeExceeded'),
        );
        return false;
      }
      uploadAsset
        .mutateAsync({ file, kind })
        .then(({ url }) => form.setFieldValue(key, url))
        .catch(() => void message.error(t('system.ui.uploadFailed')));
      // 始终阻止 antd 自动上传（我们手动调 useUploadUIAsset），不入 fileList。
      return false;
    };
  };

  return (
    <Form
      form={form}
      layout="vertical"
      size="small"
      initialValues={initialValues ?? UI_CUSTOM_DEFAULTS}
    >
      <Row gutter={24}>
        <Col span={24}>
          <Form.Item
            name="ui_omc_name"
            label={t('system.ui.omcName')}
            rules={[{ required: true, message: t('system.ui.pleaseInputOmcName') }]}
          >
            <Input placeholder={t('system.ui.pleaseInputSystemName')} maxLength={32} />
          </Form.Item>
        </Col>
      </Row>

      <Divider orientation="left" plain>
        {t('system.ui.imageUpload')}
      </Divider>

      <Row gutter={24}>
        <Col span={8}>
          <ImageUploadCard
            title={t('system.ui.loginBackground')}
            sizeHint={t('system.ui.max1mb')}
            fieldName="ui_login_background"
            previewUrl={loginBgUrl}
            beforeUpload={buildBeforeUpload('ui_login_background', 'login_bg', SIZE_LOGIN_BG)}
            uploading={uploadAsset.isPending}
            uploadHintKey="common.supportJpgPng"
            placeholderKey="common.clickOrDragToUpload"
          />
        </Col>
        <Col span={8}>
          <ImageUploadCard
            title={t('system.ui.logoSmall')}
            sizeHint={t('system.ui.max400kb')}
            fieldName="ui_menu_logo_up"
            previewUrl={logoSmallUrl}
            beforeUpload={buildBeforeUpload('ui_menu_logo_up', 'logo_small', SIZE_LOGO)}
            uploading={uploadAsset.isPending}
            uploadHintKey="system.ui.showWhenMenuCollapsed"
            placeholderKey="common.clickOrDragToUpload"
          />
        </Col>
        <Col span={8}>
          <ImageUploadCard
            title={t('system.ui.logoLarge')}
            sizeHint={t('system.ui.max400kb')}
            fieldName="ui_menu_logo_down"
            previewUrl={logoLargeUrl}
            beforeUpload={buildBeforeUpload('ui_menu_logo_down', 'logo_large', SIZE_LOGO)}
            uploading={uploadAsset.isPending}
            uploadHintKey="system.ui.showWhenMenuExpanded"
            placeholderKey="common.clickOrDragToUpload"
          />
        </Col>
      </Row>

      <Divider />

      <Space>
        <Popconfirm
          title={t('system.ui.confirmRestoreDefault')}
          description={t('system.ui.restoreWillOverwrite')}
          onConfirm={onRestoreDefaults}
          okText={t('common.confirm')}
          cancelText={t('common.cancel')}
        >
          <Button icon={<UndoOutlined />} loading={restoring}>
            {t('system.ui.restoreDefault')}
          </Button>
        </Popconfirm>
      </Space>
    </Form>
  );
}

// ------ 单个图片上传卡片 ------

interface ImageUploadCardProps {
  title: string;
  sizeHint: string;
  fieldName: string;
  previewUrl: string;
  beforeUpload: (file: RcFile) => boolean;
  uploading: boolean;
  uploadHintKey: string;
  placeholderKey: string;
}

function ImageUploadCard({
  title,
  sizeHint,
  fieldName,
  previewUrl,
  beforeUpload,
  uploading,
  uploadHintKey,
  placeholderKey,
}: ImageUploadCardProps) {
  const t = useT();
  return (
    <Card
      size="small"
      title={title}
      extra={<span style={{ color: '#999', fontSize: 12 }}>{sizeHint}</span>}
    >
      {/* 预览缩略图：value 既可能是 /api/v1/admin/public/ui-assets/<uuid>.png（已上传），
          也可能是 ./images/...png（首次启动的 bundle 自带资源）。<img> 都能正确解析。 */}
      {previewUrl ? (
        <div style={{ marginBottom: 8, textAlign: 'center' }}>
          <img
            src={previewUrl}
            alt={title}
            style={{
              maxWidth: '100%',
              maxHeight: 80,
              objectFit: 'contain',
              border: '1px solid #f0f0f0',
              borderRadius: 4,
              padding: 4,
              background: '#fafafa',
            }}
            onError={(e) => {
              (e.currentTarget as HTMLImageElement).style.display = 'none';
            }}
          />
        </div>
      ) : null}

      {/* form 字段以隐藏 Input 携带 URL；上传成功后由 form.setFieldValue 写入。 */}
      <Form.Item name={fieldName} hidden>
        <Input />
      </Form.Item>

      <Dragger
        beforeUpload={beforeUpload}
        showUploadList={false}
        disabled={uploading}
        maxCount={1}
        accept="image/jpeg,image/jpg,image/png"
      >
        <p className="ant-upload-drag-icon">
          <InboxOutlined />
        </p>
        <p className="ant-upload-text">{t(placeholderKey)}</p>
        <p className="ant-upload-hint">{uploading ? t('system.ui.uploading') : t(uploadHintKey)}</p>
      </Dragger>
    </Card>
  );
}
