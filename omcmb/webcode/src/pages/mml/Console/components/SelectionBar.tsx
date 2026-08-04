import { Button, Card, Space, Tag, Typography } from 'antd';
import {
  ApartmentOutlined,
  CodeOutlined,
  RightOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { CommandItem } from '../types';
import { opColor, opLabelI18nKey } from '../constants';
import { useT } from '@/hooks/useT';

const { Text } = Typography;

interface SelectionBarProps {
  deviceCount: number;
  command: CommandItem | null;
  /** 配置参数摘要（已配置时显示，如「命令参数 · 4 路径」） */
  configSummary?: string;
  onPickDevices: () => void;
  onPickCommand: () => void;
  onConfigParams: () => void;
}

/**
 * 顶部选择条（设计 §3.2 + §3.10.3）—— ① 选择设备 ② 选择命令 ③ 配置参数(弹框) + 执行。
 * 选完即收纳为摘要,主舞台让给「命令记录 ｜ 执行结果」。
 */
export default function SelectionBar({
  deviceCount,
  command,
  configSummary,
  onPickDevices,
  onPickCommand,
  onConfigParams,
}: SelectionBarProps) {
  const t = useT();
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
            <span>{t('mml.consoleV2.selectionBar.step1')}</span>
            <RightOutlined style={{ fontSize: 10, opacity: 0.45 }} />
            {deviceCount > 0 ? (
              <Text strong style={{ color: '#1677ff' }}>{t('mml.consoleV2.selectionBar.selectedDevices', { count: deviceCount })}</Text>
            ) : (
              <Text type="secondary">{t('mml.consoleV2.selectionBar.notSelected')}</Text>
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
            <span>{t('mml.consoleV2.selectionBar.step2')}</span>
            <RightOutlined style={{ fontSize: 10, opacity: 0.45 }} />
            {command ? (
              <Space size={4}>
                <Tag color={opColor(command.operationType)} style={{ marginInlineEnd: 0 }}>
                  {command.operationType} {t(opLabelI18nKey(command.operationType) ?? '')}
                </Tag>
                <Text strong>{command.commandName}</Text>
              </Space>
            ) : (
              <Text type="secondary">{t('mml.consoleV2.selectionBar.notSelected')}</Text>
            )}
          </Space>
        </Button>

        <Button
          size="large"
          icon={<SettingOutlined />}
          type={command && !configSummary ? 'primary' : 'default'}
          // 选好设备即可打开「配置参数」：命令参数(需选命令)与指定参数(裸路径,无需命令)
          // 两种模式都在此弹框内；仅以命令为门会让「指定参数」流程下无法回到本页编辑。
          disabled={deviceCount === 0}
          onClick={onConfigParams}
        >
          <Space size={6}>
            <span>{t('mml.consoleV2.selectionBar.step3')}</span>
            <RightOutlined style={{ fontSize: 10, opacity: 0.45 }} />
            {configSummary ? (
              <Text strong style={{ color: '#1677ff' }}>{configSummary}</Text>
            ) : (
              <Text type="secondary">{command ? t('mml.consoleV2.selectionBar.defaultAll') : t('mml.consoleV2.selectionBar.notSelected')}</Text>
            )}
          </Space>
        </Button>
      </Space>
    </Card>
  );
}
