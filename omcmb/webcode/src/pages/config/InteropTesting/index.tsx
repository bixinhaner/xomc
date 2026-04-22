import { useCallback, useMemo, useState } from 'react';
import {
  Alert,
  Badge,
  Button,
  Card,
  Col,
  Collapse,
  Form,
  Input,
  Row,
  Select,
  Space,
  Statistic,
  Table,
  Tag,
  Typography,
  message,
} from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  ExperimentOutlined,
  PlayCircleOutlined,
  ThunderboltOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import {
  useTestCases,
  useRunTests,
  useRunTestsByCategory,
  useValidateDevice,
} from '@core/hooks/api/useInterop';
import type { TestCase, TestResult, RunTestsResponse } from '@core/services/api/interopApi';

const CATEGORY_COLOR: Record<string, string> = {
  protocol: 'blue',
  datamodel: 'green',
  rpc: 'orange',
  inform: 'purple',
};

const CATEGORY_LABEL: Record<string, string> = {
  protocol: 'Protocol',
  datamodel: 'Data Model',
  rpc: 'RPC',
  inform: 'Inform',
};

const CATEGORY_OPTIONS = [
  { label: 'Protocol', value: 'protocol' },
  { label: 'Data Model', value: 'datamodel' },
  { label: 'RPC', value: 'rpc' },
  { label: 'Inform', value: 'inform' },
];

const CARRIER_OPTIONS = [
  { label: 'CMCC', value: 'cmcc' },
  { label: 'CTCC', value: 'ctcc' },
  { label: 'CUCC', value: 'cucc' },
];

const TECH_OPTIONS = [
  { label: 'LTE', value: 'lte' },
  { label: 'NR', value: 'nr' },
];

export default function InteropTesting() {
  const [runForm] = Form.useForm();
  const [validateForm] = Form.useForm();
  const [testResults, setTestResults] = useState<RunTestsResponse | null>(null);

  const { data: testCasesMap, isLoading: casesLoading } = useTestCases();
  const runTestsMutation = useRunTests();
  const runByCategoryMutation = useRunTestsByCategory();
  const validateMutation = useValidateDevice();

  // Flatten test cases for display
  const allTestCases = useMemo(() => {
    if (!testCasesMap) return [];
    const result: TestCase[] = [];
    for (const cases of Object.values(testCasesMap)) {
      result.push(...cases);
    }
    return result;
  }, [testCasesMap]);

  const categoryCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const tc of allTestCases) {
      counts[tc.category] = (counts[tc.category] || 0) + 1;
    }
    return counts;
  }, [allTestCases]);

  const handleRunTests = useCallback(async () => {
    try {
      const values = await runForm.validateFields();
      const result = await runTestsMutation.mutateAsync({
        deviceSn: values.deviceSn,
        categories: values.categories?.length ? values.categories : undefined,
      });
      setTestResults(result);
      void message.success(`Tests completed: ${result.passed}/${result.total} passed`);
    } catch {
      // validation failed or API error
    }
  }, [runForm, runTestsMutation]);

  const handleRunByCategory = useCallback(
    async (category: string) => {
      const deviceSn = runForm.getFieldValue('deviceSn') as string;
      if (!deviceSn) {
        void message.warning('Please enter a Device SN first');
        return;
      }
      const result = await runByCategoryMutation.mutateAsync({
        category,
        req: { deviceSn },
      });
      setTestResults(result);
      void message.success(
        `${CATEGORY_LABEL[category]} tests: ${result.passed}/${result.total} passed`
      );
    },
    [runForm, runByCategoryMutation]
  );

  const handleValidateDevice = useCallback(async () => {
    try {
      const values = await validateForm.validateFields();
      await validateMutation.mutateAsync({
        deviceId: values.deviceId,
        carrier: values.carrier,
        tech: values.tech,
      });
      void message.success('Device validation completed');
    } catch {
      // validation or API error
    }
  }, [validateForm, validateMutation]);

  const resultColumns = useMemo(
    (): ColumnsType<TestResult> => [
      {
        title: 'Test',
        dataIndex: 'testName',
        key: 'testName',
        width: 220,
      },
      {
        title: 'Category',
        dataIndex: 'category',
        key: 'category',
        width: 100,
        render: (val: string) => (
          <Tag color={CATEGORY_COLOR[val] ?? 'default'}>
            {CATEGORY_LABEL[val] ?? val}
          </Tag>
        ),
      },
      {
        title: 'Result',
        dataIndex: 'passed',
        key: 'passed',
        width: 100,
        render: (val: boolean) =>
          val ? (
            <Tag icon={<CheckCircleOutlined />} color="success">
              PASS
            </Tag>
          ) : (
            <Tag icon={<CloseCircleOutlined />} color="error">
              FAIL
            </Tag>
          ),
      },
      {
        title: 'Duration',
        dataIndex: 'durationMs',
        key: 'durationMs',
        width: 100,
        render: (val: number) => `${val}ms`,
      },
      {
        title: 'Details',
        dataIndex: 'details',
        key: 'details',
        ellipsis: true,
      },
      {
        title: 'Error',
        dataIndex: 'error',
        key: 'error',
        width: 200,
        ellipsis: true,
        render: (val: string) =>
          val ? <Typography.Text type="danger">{val}</Typography.Text> : '-',
      },
    ],
    []
  );

  const testCaseColumns = useMemo(
    (): ColumnsType<TestCase> => [
      {
        title: 'ID',
        dataIndex: 'id',
        key: 'id',
        width: 160,
        render: (val: string) => <span style={{ fontFamily: 'monospace', fontSize: 12 }}>{val}</span>,
      },
      {
        title: 'Name',
        dataIndex: 'name',
        key: 'name',
        width: 220,
      },
      {
        title: 'Category',
        dataIndex: 'category',
        key: 'category',
        width: 100,
        render: (val: string) => (
          <Tag color={CATEGORY_COLOR[val] ?? 'default'}>
            {CATEGORY_LABEL[val] ?? val}
          </Tag>
        ),
      },
      {
        title: 'Description',
        dataIndex: 'description',
        key: 'description',
        ellipsis: true,
      },
      {
        title: 'Steps',
        key: 'steps',
        width: 80,
        render: (_: unknown, record: TestCase) => record.steps?.length ?? 0,
      },
    ],
    []
  );

  return (
    <ListPageLayout
      title="Interop Testing"
      subtitle="Conformance testing and device validation (F10)"
      extra={
        <Tag icon={<ExperimentOutlined />} color="processing">
          F10
        </Tag>
      }
    >
      {/* Run Tests Section */}
      <Card title="Run Conformance Tests" style={{ marginBottom: 16 }}>
        <Form form={runForm} layout="inline" style={{ marginBottom: 16 }}>
          <Form.Item
            name="deviceSn"
            label="Device SN"
            rules={[{ required: true, message: 'Required' }]}
          >
            <Input placeholder="e.g. ENB00001" style={{ width: 200 }} />
          </Form.Item>
          <Form.Item name="categories" label="Categories">
            <Select
              mode="multiple"
              options={CATEGORY_OPTIONS}
              placeholder="All categories"
              style={{ width: 280 }}
              allowClear
            />
          </Form.Item>
          <Form.Item>
            <Button
              type="primary"
              icon={<PlayCircleOutlined />}
              onClick={() => void handleRunTests()}
              loading={runTestsMutation.isPending}
            >
              Run Tests
            </Button>
          </Form.Item>
        </Form>

        <Space wrap>
          {Object.entries(CATEGORY_LABEL).map(([key, label]) => (
            <Button
              key={key}
              size="small"
              icon={<ThunderboltOutlined />}
              onClick={() => void handleRunByCategory(key)}
              loading={runByCategoryMutation.isPending}
            >
              Run {label}
              {categoryCounts[key] ? (
                <Badge
                  count={categoryCounts[key]}
                  size="small"
                  style={{ marginLeft: 4 }}
                  color={CATEGORY_COLOR[key]}
                />
              ) : null}
            </Button>
          ))}
        </Space>
      </Card>

      {/* Test Results */}
      {testResults && (
        <Card title="Test Results" style={{ marginBottom: 16 }}>
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={6}>
              <Statistic title="Total Tests" value={testResults.total} />
            </Col>
            <Col span={6}>
              <Statistic
                title="Passed"
                value={testResults.passed}
                valueStyle={{ color: '#52C41A' }}
                prefix={<CheckCircleOutlined />}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="Failed"
                value={testResults.failed}
                valueStyle={{ color: '#FF4D4F' }}
                prefix={<CloseCircleOutlined />}
              />
            </Col>
            <Col span={6}>
              <Statistic
                title="Pass Rate"
                value={
                  testResults.total > 0
                    ? Math.round((testResults.passed / testResults.total) * 100)
                    : 0
                }
                suffix="%"
                valueStyle={{
                  color:
                    testResults.total > 0 && testResults.passed === testResults.total
                      ? '#52C41A'
                      : '#FA8C16',
                }}
              />
            </Col>
          </Row>

          {testResults.failed > 0 && (
            <Alert
              type="warning"
              showIcon
              message={`${testResults.failed} test(s) failed for device ${testResults.deviceSn}`}
              style={{ marginBottom: 16 }}
            />
          )}

          <Table<TestResult>
            columns={resultColumns}
            dataSource={testResults.results}
            rowKey="testCaseId"
            size="small"
            pagination={false}
            rowClassName={(record) => (record.passed ? '' : 'ant-table-row-error')}
          />
        </Card>
      )}

      {/* Device Validation */}
      <Collapse
        style={{ marginBottom: 16 }}
        items={[
          {
            key: 'validate',
            label: 'Device Data Model Validation',
            children: (
              <Form form={validateForm} layout="inline">
                <Form.Item
                  name="deviceId"
                  label="Device ID"
                  rules={[{ required: true }]}
                >
                  <Input placeholder="Device UUID" style={{ width: 280 }} />
                </Form.Item>
                <Form.Item
                  name="carrier"
                  label="Carrier"
                  rules={[{ required: true }]}
                >
                  <Select options={CARRIER_OPTIONS} style={{ width: 120 }} />
                </Form.Item>
                <Form.Item
                  name="tech"
                  label="Technology"
                  rules={[{ required: true }]}
                >
                  <Select options={TECH_OPTIONS} style={{ width: 120 }} />
                </Form.Item>
                <Form.Item>
                  <Button
                    type="primary"
                    icon={<CheckCircleOutlined />}
                    onClick={() => void handleValidateDevice()}
                    loading={validateMutation.isPending}
                  >
                    Validate
                  </Button>
                </Form.Item>
              </Form>
            ),
          },
        ]}
      />

      {/* Test Case Library */}
      <Card title="Test Case Library">
        <Table<TestCase>
          columns={testCaseColumns}
          dataSource={allTestCases}
          loading={casesLoading}
          rowKey="id"
          size="small"
          pagination={{
            pageSize: 20,
            showSizeChanger: true,
            showTotal: (t) => `Total ${t} test cases`,
          }}
        />
      </Card>
    </ListPageLayout>
  );
}
