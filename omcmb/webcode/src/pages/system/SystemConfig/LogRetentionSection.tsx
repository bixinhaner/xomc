// 日志保留与轮转配置（系统配置页 —— 一个 tab，2 张分类卡片）。
//
// 后端 sys_configs 分类（seed migrations/seed/000005，各模块热加载）：
//   log.retention  审计/业务日志按时间保留（worker 每日 05:00 cron 批量删过期行）
//                  enabled + 8 张表各自保留天数（audit/ops_audit/login/oper/task/system/ne_message/event）
//   log.rotation   运行期日志文件轮转（app/acs/worker logger override watcher 读，≤1 分钟生效）
//                  max_size_mb / max_age_days / keep_files / rotate_interval_minutes
//
// 接入方式与 RetentionBackpressureSection 一致：复用 useSysConfigsByCategory + useBatchUpdateSysConfigs，
// 每张卡片自管 form + 保存（一次保存 = 该分类一次 batch upsert，触发后端热加载）。

import { useEffect, useMemo, useState } from 'react';
import { Button, Card, Form, InputNumber, Switch, message, Spin } from 'antd';
import {
  useSysConfigsByCategory,
  useBatchUpdateSysConfigs,
} from '@core/hooks/api/useSystem';
import { useT } from '@/hooks/useT';

type FieldType = 'int' | 'bool';

interface FieldSpec {
  key: string;
  type: FieldType;
  min?: number;
  max?: number;
}

interface CardSpec {
  category: string;
  titleKey: string;
  descKey: string;
  fields: FieldSpec[];
}

// 字段定义与后端 sys_configs 键 + value_type 严格对齐（seed 000005）。
const CARDS: CardSpec[] = [
  {
    category: 'log.retention',
    titleKey: 'logCfg.retention.title',
    descKey: 'logCfg.retention.desc',
    fields: [
      { key: 'enabled', type: 'bool' },
      { key: 'audit_days', type: 'int', min: 1, max: 3650 },
      { key: 'ops_audit_days', type: 'int', min: 1, max: 3650 },
      { key: 'login_days', type: 'int', min: 1, max: 3650 },
      { key: 'oper_days', type: 'int', min: 1, max: 3650 },
      { key: 'task_days', type: 'int', min: 1, max: 3650 },
      { key: 'system_days', type: 'int', min: 1, max: 3650 },
      { key: 'ne_message_days', type: 'int', min: 1, max: 3650 },
      { key: 'event_days', type: 'int', min: 1, max: 3650 },
    ],
  },
  {
    category: 'log.rotation',
    titleKey: 'logCfg.rotation.title',
    descKey: 'logCfg.rotation.desc',
    fields: [
      { key: 'max_size_mb', type: 'int', min: 1, max: 10240 },
      { key: 'rotate_interval_minutes', type: 'int', min: 1, max: 1440 },
      { key: 'max_age_days', type: 'int', min: 1, max: 3650 },
      { key: 'keep_files', type: 'int', min: 1, max: 1000 },
    ],
  },
];

function decodeValue(raw: string, type: FieldType): number | boolean {
  if (type === 'bool') {
    return raw === 'true' || raw === '1';
  }
  const n = parseInt(raw, 10);
  return Number.isFinite(n) ? n : 0;
}

function encodeValue(v: unknown, type: FieldType): string {
  if (type === 'bool') {
    return v ? 'true' : 'false';
  }
  return String(v ?? '');
}

function CategoryConfigCard({ spec }: { spec: CardSpec }) {
  const t = useT();
  const [form] = Form.useForm<Record<string, number | boolean>>();
  const [submitting, setSubmitting] = useState(false);

  const { data: configs, isLoading } = useSysConfigsByCategory(spec.category);
  const { mutateAsync: batchUpdate } = useBatchUpdateSysConfigs();

  const initialValues = useMemo<Record<string, number | boolean>>(() => {
    const out: Record<string, number | boolean> = {};
    for (const f of spec.fields) {
      out[f.key] = f.type === 'bool' ? false : 0;
    }
    if (configs) {
      for (const cfg of configs) {
        const f = spec.fields.find((x) => x.key === cfg.key);
        if (f) {
          out[cfg.key] = decodeValue(cfg.value, f.type);
        }
      }
    }
    return out;
  }, [configs, spec.fields]);

  useEffect(() => {
    form.setFieldsValue(initialValues);
  }, [form, initialValues]);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      await batchUpdate({
        category: spec.category,
        items: spec.fields.map((f) => ({
          key: f.key,
          value: encodeValue(values[f.key], f.type),
          value_type: f.type,
        })),
      });
      message.success(t('logCfg.save.success'));
    } catch {
      // validateFields 失败或网络错误：交给 antd 自身 UI 提示
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Card
      size="small"
      title={<span style={{ fontSize: 14, fontWeight: 600 }}>{t(spec.titleKey)}</span>}
      style={{ marginBottom: 16 }}
      extra={
        <Button size="small" type="primary" onClick={handleSave} loading={submitting}>
          {t('logCfg.save')}
        </Button>
      }
    >
      <div style={{ color: '#888', fontSize: 12, marginBottom: 12 }}>{t(spec.descKey)}</div>
      <Spin spinning={isLoading}>
        <Form form={form} layout="vertical" size="small">
          {spec.fields.map((f) => (
            <Form.Item
              key={f.key}
              name={f.key}
              label={t(`logCfg.field.${f.key}`)}
              valuePropName={f.type === 'bool' ? 'checked' : 'value'}
              style={{ marginBottom: 12, maxWidth: 360 }}
            >
              {f.type === 'bool' ? (
                <Switch />
              ) : (
                <InputNumber min={f.min} max={f.max} step={1} style={{ width: 200 }} />
              )}
            </Form.Item>
          ))}
        </Form>
      </Spin>
    </Card>
  );
}

export default function LogRetentionSection() {
  return (
    <>
      {CARDS.map((spec) => (
        <CategoryConfigCard key={spec.category} spec={spec} />
      ))}
    </>
  );
}
