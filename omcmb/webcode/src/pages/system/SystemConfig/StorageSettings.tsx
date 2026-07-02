import { Form, Input, InputNumber, Card, Space, theme } from 'antd';
import { useT } from '@/hooks/useT';

interface StorageSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// 设置行样式
const settingRowStyle: React.CSSProperties = {
  marginBottom: 16,
};

// 子设置区域布局（颜色随主题，见组件内 subSettingStyle）
const subSettingBaseStyle: React.CSSProperties = {
  marginTop: 12,
  padding: '12px 16px',
  borderRadius: 4,
};


export default function StorageSettings({ form }: StorageSettingsProps) {
  const t = useT();
  const { token } = theme.useToken();
  // 子设置区域背景跟随明/暗主题，不再写死 #fafafa
  const subSettingStyle: React.CSSProperties = { ...subSettingBaseStyle, backgroundColor: token.colorFillAlter };

  return (
    <Form form={form} layout="vertical" size="small" initialValues={{
      // MinIO 对外可达 endpoint：issue #548 切片 3。空 = 走 env / 派生回退（后端订阅桥处理）
      minio_public_endpoint: '',
      alarmHisMaxHoldTime: 365,
    }}>
      {/* MinIO 对象存储 — issue #548 切片 3 收敛：
           只暴露 `MinIO 对外可达 endpoint`（sys_configs.storage.minio_public_endpoint）
           一字段；其他 endpoint/port/accessKey/secret/bucket/region/pathStyle/useSSL/enable
           都是部署期决策（docker compose / yaml）不该 UI 编辑——改了不生效就是 §5
           设计原则禁止的"打字进数据库不生效"半成品。详见 issue-548-slice3-ledger.md。 */}
      <Card
        size="small"
        title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.minio')}</span>}
        style={{ marginBottom: 16 }}
      >
        <div style={subSettingStyle}>
          <Form.Item
            label={t('system.storage.minioPublicEndpoint')}
            name="minio_public_endpoint"
            // 后端 storage.minio_public_endpoint validator 已对相同正则校验并返 400；
            // 此处加前端 pattern 让用户在输入即时看到错误，避免一次保存才得知。
            // 规则与后端 ValidatePublicEndpoint 同步：禁 scheme/path/IPv6，允许空。
            rules={[{
              validator: (_, value: string) => {
                if (!value) return Promise.resolve();
                if (value.includes('://')) return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrScheme')));
                if (/[/?#]/.test(value)) return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrPath')));
                if (/[[\]]/.test(value) || (value.match(/:/g) || []).length > 1) {
                  return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrIpv6')));
                }
                const m = value.match(/^([^:]+)(?::(\d+))?$/);
                if (!m || !m[1]) return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrFormat')));
                if (m[2]) {
                  const p = parseInt(m[2], 10);
                  if (!Number.isFinite(p) || p < 1 || p > 65535) {
                    return Promise.reject(new Error(t('system.storage.minioPublicEndpointErrPort')));
                  }
                }
                return Promise.resolve();
              },
            }]}
            extra={<span style={{ color: token.colorTextSecondary }}>{t('system.storage.minioPublicEndpointDesc')}</span>}
          >
            <Input
              style={{ width: 360 }}
              placeholder={t('system.storage.minioPublicEndpointPlaceholder')}
              allowClear
              maxLength={253}
            />
          </Form.Item>
        </div>
      </Card>

      {/* 告警 */}
      <Card size="small" title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t('system.storage.alarm')}</span>} style={{ marginBottom: 16 }}>
        <div style={settingRowStyle}>
          <Space>
            <span>{t('sysconfig.storage.alarmHistoryLabel')}</span>
            <Form.Item name="alarmHisMaxHoldTime" noStyle>
              <InputNumber min={1} max={365} style={{ width: 70 }} />
            </Form.Item>
            <span>{t('sysconfig.storage.unit.day')}</span>
          </Space>
        </div>
      </Card>

    </Form>
  );
}
