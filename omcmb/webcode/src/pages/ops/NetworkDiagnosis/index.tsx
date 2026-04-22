import { useState, useRef, useMemo } from 'react';
import { Button, Card, Form, Input, InputNumber, Select, Tabs, Statistic, Row, Col, Space } from 'antd';
import { PlayCircleOutlined, StopOutlined } from '@ant-design/icons';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import TerminalOutput from '@/components/TerminalOutput';
import type { TerminalOutputHandle, TerminalLine } from '@/components/TerminalOutput';
import { useT } from '@/hooks/useT';

interface PingStats {
  transmitted: number;
  received: number;
  loss: number;
  minRtt: number;
  avgRtt: number;
  maxRtt: number;
}

function generatePingLines(host: string, count: number, interval: number): TerminalLine[] {
  const lines: TerminalLine[] = [];
  lines.push({ text: `PING ${host} (${host}) 56(84) bytes of data.`, type: 'info' });
  const rtts: number[] = [];
  for (let i = 1; i <= count; i++) {
    const rtt = Math.round((Math.random() * 20 + 5) * 100) / 100;
    rtts.push(rtt);
    const ttl = Math.floor(Math.random() * 10 + 55);
    const seq = i;
    lines.push({
      text: `64 bytes from ${host}: icmp_seq=${seq} ttl=${ttl} time=${rtt} ms`,
      type: rtt > 50 ? 'stderr' : 'stdout',
    });
  }
  const received = rtts.length;
  const loss = Math.round(((count - received) / count) * 100);
  const minRtt = Math.min(...rtts);
  const avgRtt = Math.round((rtts.reduce((a, b) => a + b, 0) / rtts.length) * 100) / 100;
  const maxRtt = Math.max(...rtts);
  lines.push({ text: '', type: 'info' });
  lines.push({ text: `--- ${host} ping statistics ---`, type: 'info' });
  lines.push({
    text: `${count} packets transmitted, ${received} received, ${loss}% packet loss, time ${count * interval * 1000}ms`,
    type: 'info',
  });
  lines.push({
    text: `rtt min/avg/max = ${minRtt}/${avgRtt}/${maxRtt} ms`,
    type: received === count ? 'success' : 'stderr',
  });
  return lines;
}

function generateTracerouteLines(host: string, maxHops: number): TerminalLine[] {
  const lines: TerminalLine[] = [];
  lines.push({ text: `traceroute to ${host} (${host}), ${maxHops} hops max, 60 byte packets`, type: 'info' });
  const hopCount = Math.floor(Math.random() * 6 + 4);
  const hopAddresses = [
    '10.0.0.1', '172.16.1.1', '172.16.2.1', '10.100.1.1', '10.200.1.1',
    '192.168.50.1', '192.168.100.1', '203.100.1.1', '210.50.10.1', host,
  ];
  for (let i = 1; i <= Math.min(hopCount, maxHops); i++) {
    const hopIp = hopAddresses[i - 1] ?? `10.${i}.0.1`;
    const isLast = i === Math.min(hopCount, maxHops);
    const rtt1 = (Math.random() * 15 + 1).toFixed(3);
    const rtt2 = (Math.random() * 15 + 1).toFixed(3);
    const rtt3 = (Math.random() * 15 + 1).toFixed(3);
    if (isLast) {
      lines.push({
        text: ` ${String(i).padStart(2)}  ${host}  ${rtt1} ms  ${rtt2} ms  ${rtt3} ms`,
        type: 'success',
      });
    } else {
      lines.push({
        text: ` ${String(i).padStart(2)}  ${hopIp}  ${rtt1} ms  ${rtt2} ms  ${rtt3} ms`,
        type: 'stdout',
      });
    }
  }
  lines.push({ text: '', type: 'info' });
  lines.push({
    text: `Traceroute complete. Reached ${host} in ${Math.min(hopCount, maxHops)} hops.`,
    type: 'success',
  });
  return lines;
}

export default function NetworkDiagnosis() {
  const t = useT();
  const [pingForm] = Form.useForm();
  const [traceForm] = Form.useForm();
  const [pingRunning, setPingRunning] = useState(false);
  const [traceRunning, setTraceRunning] = useState(false);
  const [pingLines, setPingLines] = useState<TerminalLine[]>([]);
  const [traceLines, setTraceLines] = useState<TerminalLine[]>([]);
  const [pingStats, setPingStats] = useState<PingStats | null>(null);
  const pingTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const traceTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const pingRef = useRef<TerminalOutputHandle>(null);
  const traceRef = useRef<TerminalOutputHandle>(null);

  const deviceOptions = useMemo(() => [
    { label: t('ops.allOmcServer'), value: 'OMC-SERVER' },
    { label: 'ENB00001 — 北京-eNB-0001', value: 'ENB00001' },
    { label: 'ENB00002 — 北京-eNB-0002', value: 'ENB00002' },
    { label: 'GNB00001 — 北京-gNB-0001', value: 'GNB00001' },
    { label: 'ENB00010 — 上海-eNB-0010', value: 'ENB00010' },
  ], [t]);

  const handleStartPing = () => {
    pingForm.validateFields().then((vals) => {
      const host = vals.target as string;
      const count = Number(vals.count ?? 4);
      const interval = Number(vals.interval ?? 1);
      setPingRunning(true);
      setPingLines([]);
      setPingStats(null);
      const lines = generatePingLines(host, count, interval);
      let index = 0;

      const appendNext = () => {
        if (index >= lines.length) {
          setPingRunning(false);
          const received = count;
          const rtts = Array.from({ length: count }, () => Math.round((Math.random() * 20 + 5) * 100) / 100);
          setPingStats({
            transmitted: count,
            received,
            loss: 0,
            minRtt: Math.min(...rtts),
            avgRtt: Math.round((rtts.reduce((a, b) => a + b, 0) / rtts.length) * 100) / 100,
            maxRtt: Math.max(...rtts),
          });
          return;
        }
        setPingLines((prev) => [...prev, lines[index]!]);
        index++;
        pingTimerRef.current = setTimeout(appendNext, index < 2 ? 100 : interval * 1000 / count * 300);
      };

      appendNext();
    });
  };

  const handleStopPing = () => {
    if (pingTimerRef.current) clearTimeout(pingTimerRef.current);
    setPingRunning(false);
    setPingLines((prev) => [...prev, { text: '^C Ping interrupted by user.', type: 'stderr' }]);
  };

  const handleStartTrace = () => {
    traceForm.validateFields().then((vals) => {
      const host = vals.target as string;
      const maxHops = Number(vals.maxHops ?? 30);
      setTraceRunning(true);
      setTraceLines([]);
      const lines = generateTracerouteLines(host, maxHops);
      let index = 0;

      const appendNext = () => {
        if (index >= lines.length) {
          setTraceRunning(false);
          return;
        }
        setTraceLines((prev) => [...prev, lines[index]!]);
        index++;
        traceTimerRef.current = setTimeout(appendNext, index < 2 ? 100 : 400);
      };

      appendNext();
    });
  };

  const handleStopTrace = () => {
    if (traceTimerRef.current) clearTimeout(traceTimerRef.current);
    setTraceRunning(false);
    setTraceLines((prev) => [...prev, { text: '^C Traceroute interrupted by user.', type: 'stderr' }]);
  };

  return (
    <ListPageLayout title={t('nav.ops.networkDiagnosis')} subtitle={t('ops.networkDiagnosisSubtitle')}>
      <Tabs
        items={[
          {
            key: 'ping',
            label: 'Ping',
            children: (
              <div style={{ display: 'flex', gap: 16, flexDirection: 'column' }}>
                <Card size="small" title={t('ops.pingParams')}>
                  <Form form={pingForm} layout="inline" initialValues={{ count: 4, interval: 1, timeout: 5 }}>
                    <Form.Item name="source" label={t('ops.sourceDevice')}>
                      <Select
                        style={{ width: 240 }}
                        placeholder={t('ops.selectSourceDevice')}
                        allowClear
                        options={deviceOptions}
                      />
                    </Form.Item>
                    <Form.Item name="target" label={t('ops.targetAddress')} rules={[{ required: true, message: t('ops.enterTargetAddress') }]}>
                      <Input style={{ width: 200 }} placeholder={t('ops.ipOrDomain')} />
                    </Form.Item>
                    <Form.Item name="count" label={t('ops.count')}>
                      <InputNumber min={1} max={100} style={{ width: 80 }} />
                    </Form.Item>
                    <Form.Item name="interval" label={t('ops.intervalSeconds')}>
                      <InputNumber min={0.2} max={10} step={0.5} style={{ width: 90 }} />
                    </Form.Item>
                    <Form.Item name="timeout" label={t('ops.timeoutSeconds')}>
                      <InputNumber min={1} max={30} style={{ width: 80 }} />
                    </Form.Item>
                    <Form.Item>
                      <Space>
                        {!pingRunning ? (
                          <Button
                            type="primary"
                            icon={<PlayCircleOutlined />}
                            onClick={handleStartPing}
                          >
                            {t('ops.startPing')}
                          </Button>
                        ) : (
                          <Button
                            danger
                            icon={<StopOutlined />}
                            onClick={handleStopPing}
                          >
                            {t('ops.stop')}
                          </Button>
                        )}
                      </Space>
                    </Form.Item>
                  </Form>
                </Card>

                <TerminalOutput
                  ref={pingRef}
                  lines={pingLines}
                  height={300}
                  autoScroll
                  showTimestamp={false}
                />

                {pingStats && (
                  <Card size="small" title={t('ops.pingStats')}>
                    <Row gutter={24}>
                      <Col span={4}>
                        <Statistic title={t('ops.transmitted')} value={pingStats.transmitted} />
                      </Col>
                      <Col span={4}>
                        <Statistic title={t('ops.received')} value={pingStats.received} valueStyle={{ color: '#52c41a' }} />
                      </Col>
                      <Col span={4}>
                        <Statistic
                          title={t('ops.packetLoss')}
                          value={pingStats.loss}
                          suffix="%"
                          valueStyle={{ color: pingStats.loss > 0 ? '#ff4d4f' : '#52c41a' }}
                        />
                      </Col>
                      <Col span={4}>
                        <Statistic title={t('ops.minRtt')} value={pingStats.minRtt} suffix="ms" precision={2} />
                      </Col>
                      <Col span={4}>
                        <Statistic title={t('ops.avgRtt')} value={pingStats.avgRtt} suffix="ms" precision={2} />
                      </Col>
                      <Col span={4}>
                        <Statistic title={t('ops.maxRtt')} value={pingStats.maxRtt} suffix="ms" precision={2} />
                      </Col>
                    </Row>
                  </Card>
                )}
              </div>
            ),
          },
          {
            key: 'traceroute',
            label: 'Traceroute',
            children: (
              <div style={{ display: 'flex', gap: 16, flexDirection: 'column' }}>
                <Card size="small" title={t('ops.tracerouteParams')}>
                  <Form form={traceForm} layout="inline" initialValues={{ maxHops: 30, timeout: 5 }}>
                    <Form.Item name="source" label={t('ops.sourceDevice')}>
                      <Select
                        style={{ width: 240 }}
                        placeholder={t('ops.selectSourceDevice')}
                        allowClear
                        options={deviceOptions}
                      />
                    </Form.Item>
                    <Form.Item name="target" label={t('ops.targetAddress')} rules={[{ required: true, message: t('ops.enterTargetAddress') }]}>
                      <Input style={{ width: 200 }} placeholder={t('ops.ipOrDomain')} />
                    </Form.Item>
                    <Form.Item name="maxHops" label={t('ops.maxHops')}>
                      <InputNumber min={1} max={64} style={{ width: 90 }} />
                    </Form.Item>
                    <Form.Item name="timeout" label={t('ops.timeoutSeconds')}>
                      <InputNumber min={1} max={30} style={{ width: 80 }} />
                    </Form.Item>
                    <Form.Item>
                      <Space>
                        {!traceRunning ? (
                          <Button
                            type="primary"
                            icon={<PlayCircleOutlined />}
                            onClick={handleStartTrace}
                          >
                            {t('ops.startTraceroute')}
                          </Button>
                        ) : (
                          <Button
                            danger
                            icon={<StopOutlined />}
                            onClick={handleStopTrace}
                          >
                            {t('ops.stop')}
                          </Button>
                        )}
                      </Space>
                    </Form.Item>
                  </Form>
                </Card>

                <TerminalOutput
                  ref={traceRef}
                  lines={traceLines}
                  height={400}
                  autoScroll
                  showTimestamp={false}
                />

                {traceLines.length > 0 && !traceRunning && (
                  <Card size="small" style={{ background: '#f6ffed', border: '1px solid #b7eb8f' }}>
                    <div style={{ color: '#52c41a', fontWeight: 500 }}>
                      {t('ops.tracerouteComplete')}
                    </div>
                  </Card>
                )}
              </div>
            ),
          },
        ]}
      />
    </ListPageLayout>
  );
}
