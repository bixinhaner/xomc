import { Button, Card, Space, Tag, Typography } from 'antd';
import { ApartmentOutlined, CodeOutlined, RightOutlined } from '@ant-design/icons';
import type { CommandItem } from '../types';
import { opColor, opLabel } from '../constants';

const { Text } = Typography;

interface SelectionBarProps {
  deviceCount: number;
  command: CommandItem | null;
  onPickDevices: () => void;
  onPickCommand: () => void;
}

/**
 * 顶部选择条（设计 §3.2 ①）—— 取代常驻的设备树 + 命令树两大栏，仅显示「已选摘要」，
 * 点击弹出对应弹框选择，把主舞台让给左操作区 + 右结果表格。
 */
export default function SelectionBar({
  deviceCount,
  command,
  onPickDevices,
  onPickCommand,
}: SelectionBarProps) {
  return (
    <Card variant="borderless" styles={{ body: { padding: '12px 16px' } }}>
      <Space size={12} wrap>
        <Button
          size="large"
          icon={<ApartmentOutlined />}
          type={deviceCount > 0 ? 'default' : 'primary'}
          onClick={onPickDevices}
        >
          <Space size={6}>
            <span>① 选择目标设备</span>
            <RightOutlined style={{ fontSize: 10, opacity: 0.45 }} />
            {deviceCount > 0 ? (
              <Text strong style={{ color: '#1677ff' }}>已选 {deviceCount} 台</Text>
            ) : (
              <Text type="secondary">未选择</Text>
            )}
          </Space>
        </Button>

        <Button
          size="large"
          icon={<CodeOutlined />}
          type={!command && deviceCount > 0 ? 'primary' : 'default'}
          disabled={deviceCount === 0}
          onClick={onPickCommand}
        >
          <Space size={6}>
            <span>② 选择 MML 命令</span>
            <RightOutlined style={{ fontSize: 10, opacity: 0.45 }} />
            {command ? (
              <Space size={4}>
                <Tag color={opColor(command.operationType)} style={{ marginInlineEnd: 0 }}>
                  {command.operationType} {opLabel(command.operationType)}
                </Tag>
                <Text strong>{command.commandName}</Text>
              </Space>
            ) : (
              <Text type="secondary">未选择</Text>
            )}
          </Space>
        </Button>
      </Space>
    </Card>
  );
}
