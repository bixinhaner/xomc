import { useState, useEffect } from 'react';
import { Card, Input, Tag, Typography, Space } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import { productApi } from '@core/services/api/productApi';
import type { ProductMatchResult } from '@core/types/product';

const { Text } = Typography;

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
    <Card
      size="small"
      style={{ marginBottom: 12 }}
      title={
        <Space>
          <ThunderboltOutlined />
          <span>测试匹配 / Test Match</span>
        </Space>
      }
    >
      <Space direction="vertical" style={{ width: '100%' }}>
        <Input
          placeholder="输入 productClass（如 PicoCell-LTE-V2-001）查看命中规则"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          allowClear
          style={{ maxWidth: 480 }}
        />
        {debounced && (
          <div>
            {loading && <Text type="secondary">匹配中…</Text>}
            {!loading && result && result.matched && result.product && (
              <Space wrap>
                <Tag color="success">命中</Tag>
                <Text strong>{result.product.name}</Text>
                <Text code>{result.matchedPattern}</Text>
                <Text type="secondary">全局序号 {result.globalOrder}</Text>
                <Text type="secondary">vendor={result.product.vendor}</Text>
                <Text type="secondary">tech={result.product.tech}</Text>
              </Space>
            )}
            {!loading && result && !result.matched && (
              <Space>
                <Tag color="warning">未命中</Tag>
                <Text type="secondary">该 productClass 将进入孤儿设备列表，请在产品的"正则模式"段添加规则。</Text>
              </Space>
            )}
          </div>
        )}
      </Space>
    </Card>
  );
}
