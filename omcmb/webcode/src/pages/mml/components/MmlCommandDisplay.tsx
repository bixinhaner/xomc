import { useMemo, useState } from 'react';
import { Button, Empty, Input, Modal, Popover, Space, Tag, Tooltip, Typography } from 'antd';
import { parseMmlCommandDisplay } from '@core/utils/mmlCommandDisplay';
import type { MmlCommandDisplayParam } from '@core/utils/mmlCommandDisplay';
import { useT } from '@/hooks/useT';

const COMPACT_PARAM_LIMIT = 12;

function operationColor(operation: string): string {
  switch (operation.toUpperCase()) {
    case 'LST':
      return 'blue';
    case 'MOD':
      return 'green';
    case 'ADD':
      return 'purple';
    case 'DEL':
    case 'RMV':
      return 'orange';
    default:
      return 'default';
  }
}

export interface MmlCommandDisplayProps {
  command: string;
  maxTargetWidth?: number;
}

function ParamsList({ params, maxHeight, keyColumnWidth }: {
  params: MmlCommandDisplayParam[];
  maxHeight: number;
  keyColumnWidth: number;
}) {
  const t = useT();
  return (
    <div
      style={{
        maxHeight,
        overflow: 'auto',
        border: '1px solid var(--color-border-secondary, rgba(128,128,128,0.24))',
        borderRadius: 6,
      }}
    >
      {params.length ? params.map((param, index) => (
        <div
          key={`${param.key}-${index}`}
          style={{
            display: 'grid',
            gridTemplateColumns: `${keyColumnWidth}px minmax(0, 1fr)`,
            gap: 12,
            padding: '7px 10px',
            borderBottom: '1px solid var(--color-border-secondary, rgba(128,128,128,0.24))',
          }}
        >
          <Typography.Text code style={{ fontSize: 12, wordBreak: 'break-all' }}>
            {param.key}
          </Typography.Text>
          <Typography.Text style={{ fontSize: 12, wordBreak: 'break-all' }}>
            {param.value || '-'}
          </Typography.Text>
        </div>
      )) : (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('common.noData')} style={{ margin: '18px 0' }} />
      )}
    </div>
  );
}

export default function MmlCommandDisplay({ command, maxTargetWidth = 220 }: MmlCommandDisplayProps) {
  const t = useT();
  const [paramsModalOpen, setParamsModalOpen] = useState(false);
  const [paramsSearch, setParamsSearch] = useState('');
  const parsed = useMemo(() => parseMmlCommandDisplay(command), [command]);
  const isLargeParams = parsed.parameterCount > COMPACT_PARAM_LIMIT;
  const filteredParams = useMemo(() => {
    const keyword = paramsSearch.trim().toLowerCase();
    if (!keyword) return parsed.params;
    return parsed.params.filter((param) => (
      param.key.toLowerCase().includes(keyword)
      || param.value.toLowerCase().includes(keyword)
    ));
  }, [paramsSearch, parsed.params]);
  const paramsContent = parsed.parameterCount > 0 ? (
    <div style={{ width: 520, maxWidth: '70vw' }}>
      <Space size={6} style={{ marginBottom: 8 }}>
        {parsed.operation ? <Tag color={operationColor(parsed.operation)}>{parsed.operation}</Tag> : null}
        <Typography.Text strong>{parsed.target || parsed.commandHead}</Typography.Text>
      </Space>
      <ParamsList params={parsed.params} maxHeight={280} keyColumnWidth={190} />
    </div>
  ) : null;

  return (
    <Space size={6} wrap={false} style={{ maxWidth: '100%' }}>
      {parsed.operation ? (
        <Tag color={operationColor(parsed.operation)} style={{ marginInlineEnd: 0, flex: '0 0 auto' }}>
          {parsed.operation}
        </Tag>
      ) : null}
      <Tooltip title={parsed.parameterCount > 0 ? undefined : command}>
        <Typography.Text style={{ minWidth: 0, maxWidth: maxTargetWidth }} ellipsis>
          {parsed.target || parsed.commandHead || command}
        </Typography.Text>
      </Tooltip>
      {paramsContent && !isLargeParams ? (
        <Popover
          title={t('mml.scriptParamsTitle')}
          content={paramsContent}
          trigger="click"
          placement="bottomLeft"
        >
          <Button type="link" size="small" style={{ padding: 0, flex: '0 0 auto' }}>
            {t('mml.scriptParamsCount', { count: parsed.parameterCount })}
          </Button>
        </Popover>
      ) : null}
      {isLargeParams ? (
        <>
          <Button
            type="link"
            size="small"
            style={{ padding: 0, flex: '0 0 auto' }}
            onClick={() => {
              setParamsSearch('');
              setParamsModalOpen(true);
            }}
          >
            {t('mml.scriptParamsCount', { count: parsed.parameterCount })}
          </Button>
          {paramsModalOpen ? (
            <Modal
              title={t('mml.scriptParamsTitle')}
              open={paramsModalOpen}
              onCancel={() => setParamsModalOpen(false)}
              footer={null}
              width={760}
            >
              <Space direction="vertical" size={12} style={{ width: '100%' }}>
                <Space size={6} wrap>
                  {parsed.operation ? <Tag color={operationColor(parsed.operation)}>{parsed.operation}</Tag> : null}
                  <Typography.Text strong style={{ wordBreak: 'break-all' }}>
                    {parsed.target || parsed.commandHead}
                  </Typography.Text>
                </Space>
                <Input.Search
                  allowClear
                  value={paramsSearch}
                  onChange={(event) => setParamsSearch(event.target.value)}
                  placeholder={t('mml.scriptParamsSearchPlaceholder')}
                />
                <Typography.Text type="secondary">
                  {t('mml.scriptParamsMatchedCount', { shown: filteredParams.length, total: parsed.parameterCount })}
                </Typography.Text>
                <ParamsList params={filteredParams} maxHeight={420} keyColumnWidth={240} />
              </Space>
            </Modal>
          ) : null}
        </>
      ) : null}
    </Space>
  );
}
