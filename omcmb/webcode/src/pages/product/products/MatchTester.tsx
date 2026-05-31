import { useState, useEffect } from 'react';
import { Input, Tag, Typography, Space, Spin } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import { productApi } from '@core/services/api/productApi';
import type { ProductMatchResult } from '@core/types/product';

const { Text } = Typography;

interface MatchResultInlineProps {
  result: ProductMatchResult | null;
  loading: boolean;
}

function MatchResultInline({ result, loading }: MatchResultInlineProps) {
  if (loading) {
    return (
      <Space size={4}>
        <Spin size="small" />
        <Text type="secondary">{t('product.matchTester.matching')}</Text>
      </Space>
    );
  }
  if (!result) {
    return null;
  }
  if (result.matched && result.product) {
    return (
      <Space size={6} wrap>
        <Tag color="success" style={{ marginRight: 0 }}>
          命中
        </Tag>
        <Text strong>{result.product.name}</Text>
        <Text code>{result.matchedPattern}</Text>
        <Text type="secondary">
          全局序号 {result.globalOrder} · {result.product.vendor}/{result.product.tech}
        </Text>
      </Space>
    );
  }
  return (
    <Space size={4}>
      <Tag color="warning" style={{ marginRight: 0 }}>
        未命中
      </Tag>
      <Text type="secondary">{t('product.matchTester.noMatch')}</Text>
    </Space>
  );
}

export default function MatchTester() {
  const [value, setValue] = useState('');
  const [debounced, setDebounced] = useState('');
  const [result, setResult] = useState<ProductMatchResult | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const t = setTimeout(() => setDebounced(value.trim()), 300);
    return () => clearTimeout(t);
  }, [value]);

  useEffect(() => {
    if (!debounced) {
      setResult(null);
      return;
    }
    setLoading(true);
    productApi
      .match(debounced)
      .then((r) => setResult(r))
      .catch(() => setResult({ matched: false, productClass: debounced }))
      .finally(() => setLoading(false));
  }, [debounced]);

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 12, flexWrap: 'wrap' }}>
      <Space>
        <ThunderboltOutlined style={{ color: '#1677ff' }} />
        <Text>{t('product.matchTester.matched')}</Text>
      </Space>
      <Input
        placeholder={t('product.matchTester.inputPh')}
        value={value}
        onChange={(e) => setValue(e.target.value)}
        allowClear
        style={{ width: 280 }}
      />
      <MatchResultInline result={result} loading={loading} />
    </div>
  );
}
