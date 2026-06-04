import { Button, Card, Space, Tag, Typography } from 'antd';
import {
  ApartmentOutlined,
  CodeOutlined,
  PlayCircleOutlined,
  RightOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { CommandItem } from '../types';
import { opColor, opLabel } from '../constants';

const { Text } = Typography;

interface SelectionBarProps {
  deviceCount: number;
  command: CommandItem | null;
  /** 配置参数摘要（已配置时显示，如「命令参数 · 4 路径」） */
  configSummary?: string;
  running: boolean;
  canExecute: boolean;
  onPickDevices: () => void;
  onPickCommand: () => void;
  onConfigParams: () => void;
  onExecute: () => void;
}

/**
 * 顶部选择条（设计 §3.2 + §3.10.3）—— ① 选择设备 ② 选择命令 ③ 配置参数(弹框) + 执行。
 * 选完即收纳为摘要,主舞台让给「命令记录 ｜ 执行结果」。
 */
export default function SelectionBar({
  deviceCount,
  command,
  configSummary,
  running,
  canExecute,
  onPickDevices,
  onPickCommand,
  onConfigParams,
  onExecute,
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
            <span>① 选择设备</span>
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
            <span>② 选择命令</span>
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

        <Button
          size="large"
          icon={<SettingOutlined />}
          type={command && !configSummary ? 'primary' : 'default'}
          disabled={!command}
          onClick={onConfigParams}
        >
          <Space size={6}>
            <span>③ 配置参数</span>
            <RightOutlined style={{ fontSize: 10, opacity: 0.45 }} />
            {configSummary ? (
              <Text strong style={{ color: '#1677ff' }}>{configSummary}</Text>
            ) : (
              <Text type="secondary">{command ? '默认全部' : '未选择'}</Text>
            )}
          </Space>
        </Button>

        <Button
          size="large"
          type="primary"
          icon={<PlayCircleOutlined />}
          loading={running}
          disabled={!canExecute}
          onClick={onExecute}
        >
          执行{deviceCount > 0 ? `（${deviceCount} 台）` : ''}
        </Button>
      </Space>
    </Card>
  );
}
