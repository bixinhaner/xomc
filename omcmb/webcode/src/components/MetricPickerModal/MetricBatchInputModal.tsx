import { useMemo, useState } from 'react';
import { Alert, Modal, Space, Tag, Typography } from 'antd';
import { useIntl } from 'react-intl';

import {
  buildMetricBatchSelection,
  parseMetricBatchIds,
  type MetricBatchIndicator,
  type MetricBatchSelectionResult,
} from './metricBatchSelection';

const { Text } = Typography;

interface MetricBatchInputModalProps {
  open: boolean;
  candidates: MetricBatchIndicator[];
  currentSelected: string[];
  maxSelected?: number;
  loading?: boolean;
  onApply: (result: MetricBatchSelectionResult) => void;
  onCancel: () => void;
}

export default function MetricBatchInputModal({
  open,
  candidates,
  currentSelected,
  maxSelected,
  loading = false,
  onApply,
  onCancel,
}: MetricBatchInputModalProps) {
  const intl = useIntl();
  const [input, setInput] = useState('');
  const [error, setError] = useState<string | null>(null);
  const [prevOpen, setPrevOpen] = useState(open);
  if (prevOpen !== open) {
    setPrevOpen(open);
    if (!open) {
      setInput('');
      setError(null);
    }
  }

  const result = useMemo(
    () => buildMetricBatchSelection({
      text: input,
      indicators: candidates,
      selected: currentSelected,
      maxSelected,
    }),
    [input, candidates, currentSelected, maxSelected],
  );
  const parsedIds = useMemo(() => parseMetricBatchIds(input), [input]);

  const handleCancel = () => {
    setInput('');
    setError(null);
    onCancel();
  };

  const handleOk = () => {
    if (parsedIds.length === 0) {
      setError(intl.formatMessage({ id: 'perf.metricBatchInput.empty' }));
      return;
    }
    if (result.validIds.length === 0) {
      setError(intl.formatMessage({ id: 'perf.metricBatchInput.allInvalid' }));
      return;
    }
    onApply(result);
    setInput('');
    setError(null);
  };

  return (
    <Modal
      title={intl.formatMessage({ id: 'perf.metricBatchInput.title' })}
      open={open}
      onOk={handleOk}
      onCancel={handleCancel}
      okText={intl.formatMessage({ id: 'perf.metricBatchInput.addToSelected' })}
      cancelText={intl.formatMessage({ id: 'common.cancel' })}
      confirmLoading={loading}
      okButtonProps={{ disabled: loading }}
      destroyOnHidden
      width={560}
    >
      <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
        <Text type="secondary">
          {intl.formatMessage({ id: 'perf.metricBatchInput.hint' })}
        </Text>
        <textarea
          value={input}
          onChange={(event) => {
            setInput(event.target.value);
            setError(null);
          }}
          placeholder={intl.formatMessage({ id: 'perf.metricBatchInput.placeholder' })}
          rows={8}
          style={{
            width: '100%',
            resize: 'vertical',
            fontFamily: 'SFMono-Regular, Consolas, Liberation Mono, Menlo, monospace',
            fontSize: 13,
            lineHeight: 1.6,
            padding: '8px 12px',
            boxSizing: 'border-box',
          }}
        />
        {error ? <Alert type="error" showIcon message={error} /> : null}
        {parsedIds.length > 0 ? (
          <div>
            <Space wrap size={[4, 4]}>
              <Text type="secondary">
                {intl.formatMessage({ id: 'perf.metricBatchInput.preview' }, { count: parsedIds.length })}
              </Text>
              {parsedIds.slice(0, 12).map((id) => (
                <Tag
                  key={id}
                  color={result.invalidIds.includes(id) ? 'error' : 'blue'}
                  style={{ fontFamily: 'monospace' }}
                >
                  {id}
                </Tag>
              ))}
              {parsedIds.length > 12 ? (
                <Tag>{intl.formatMessage({ id: 'perf.metricBatchInput.more' }, { count: parsedIds.length - 12 })}</Tag>
              ) : null}
            </Space>
          </div>
        ) : null}
      </Space>
    </Modal>
  );
}
