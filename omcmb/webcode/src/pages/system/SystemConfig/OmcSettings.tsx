import { Form } from 'antd';

interface OmcSettingsProps {
  form: ReturnType<typeof Form.useForm>[0];
}

// OMC 协议设置（rsyslog 转发）和磁盘告警设置两个卡片已隐藏（#802）：
// rsyslog 三个 key 后端从未读取、测试按钮为伪造逻辑；
// 磁盘告警四个 key 与「存储设置」重复且后端未实现（#801）。
// 待 rsyslog 后端接通后在此重新上线卡片。
export default function OmcSettings({ form }: OmcSettingsProps) {
  return (
    <Form form={form} layout="vertical" size="small" />
  );
}
