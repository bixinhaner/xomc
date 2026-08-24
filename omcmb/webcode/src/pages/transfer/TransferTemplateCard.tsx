import { Card, Space, Tag, Typography } from 'antd';

import { useT } from '@/hooks/useT';
import type { UnifiedFileTransferTaskType } from '@core/types/unifiedFileTransfer';

import {
  getSoftwareLibraryFileTypeLabel,
  localizeBuiltinDescription,
  localizeBuiltinTypeName,
} from './shared';

const { Text } = Typography;

/**
 * 单个 UFTE 模板卡片。从 shared.tsx 拆出（react-refresh/only-export-components：
 * shared.tsx 是工具/常量集合，组件单独成文件）。
 */
export function TransferTemplateCard({
  taskType,
  active = false,
}: {
  taskType: UnifiedFileTransferTaskType;
  active?: boolean;
}) {
  const t = useT();
  const displayName = localizeBuiltinTypeName(taskType.typeCode, taskType.displayName, t);
  const description = localizeBuiltinDescription(taskType.typeCode, taskType.description, t);
  return (
    <Card
      size="small"
      style={{
        borderColor: active ? '#1677ff' : undefined,
        boxShadow: active ? '0 0 0 1px rgba(22,119,255,0.18)' : undefined,
        height: '100%',
      }}
    >
      <Space direction="vertical" size={8} style={{ width: '100%' }}>
        <Space wrap>
          <Text strong>{displayName}</Text>
          <Tag color={taskType.builtIn ? 'blue' : 'gold'}>{taskType.builtIn ? t('ufte.tag.builtIn') : t('ufte.tag.custom')}</Tag>
          <Tag>{taskType.rpcType}</Tag>
          {/* ims_core：展示报文实际字面值（fileTypeLabel），catalog fileType 是
              任务反查键不面向用户；其它分类两者一致。 */}
          <Tag color="cyan">FileType {taskType.category === 'ims_core' ? taskType.fileTypeLabel : taskType.fileType}</Tag>
          {taskType.firmwareFileType !== undefined ? (
            <Tag color="geekblue">{t('ufte.template.softLibTag', { label: getSoftwareLibraryFileTypeLabel(taskType.firmwareFileType, t) })}</Tag>
          ) : null}
          <Tag color={taskType.enabled ? 'green' : 'default'}>{taskType.enabled ? t('ufte.template.enabled') : t('ufte.template.disabled')}</Tag>
        </Space>
        <Text type="secondary">{description}</Text>
      </Space>
    </Card>
  );
}
