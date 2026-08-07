// 日志保留与轮转配置（系统配置“资源保留与背压”页签中的 2 张统一策略卡片）。
//
// 后端 sys_configs 分类（seed migrations/seed/000001，各模块热加载）：
//   log.retention  数据库日志统一按时间保留（worker 每日 05:00 cron 批量删过期行）
//                  enabled + database_days（覆盖全部受管数据库日志表）
//   log.rotation   OMC 服务程序日志统一有效期 + 文件轮转（app/acs/worker watcher ≤1 分钟生效）
//                  service_days / max_size_mb / keep_files / rotate_interval_minutes
//
// 接入方式与 RetentionBackpressureSection 一致：复用 useSysConfigsByCategory + useBatchUpdateSysConfigs，
// 每张卡片自管 form + 保存（一次保存 = 该分类一次 batch upsert，触发后端热加载）。

import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { Alert, Button, Card, Form, InputNumber, Switch, message, Spin } from 'antd';
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
  defaultValue?: number | boolean;
}

interface CardSpec {
  category: string;
  titleKey: string;
  descKey: string;
  fields: FieldSpec[];
}

// 字段定义与后端 sys_configs 键 + value_type 严格对齐（seed 000001）。
const CARDS: CardSpec[] = [
  {
    category: 'log.retention',
    titleKey: 'logCfg.retention.title',
    descKey: 'logCfg.retention.desc',
    fields: [
      { key: 'enabled', type: 'bool' },
      { key: 'database_days', type: 'int', min: 1, max: 3650, defaultValue: 180 },
    ],
  },
  {
    category: 'log.rotation',
    titleKey: 'logCfg.rotation.title',
    descKey: 'logCfg.rotation.desc',
    fields: [
      { key: 'service_days', type: 'int', min: 1, max: 3650, defaultValue: 30 },
      { key: 'max_size_mb', type: 'int', min: 1, max: 10240 },
      { key: 'rotate_interval_minutes', type: 'int', min: 1, max: 1440 },
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

  const {
    data: configs,
    isLoading,
    isFetching,
    isError,
    isSuccess,
    refetch,
  } = useSysConfigsByCategory(spec.category);
  const { mutateAsync: batchUpdate } = useBatchUpdateSysConfigs();
  const canEdit = isSuccess && !isFetching;
  const canEditRef = useRef(canEdit);
  useLayoutEffect(() => {
    canEditRef.current = canEdit;
  }, [canEdit]);

  const initialValues = useMemo<Record<string, number | boolean>>(() => {
    const out: Record<string, number | boolean> = {};
    for (const f of spec.fields) {
      out[f.key] = f.defaultValue ?? (f.type === 'bool' ? false : 0);
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
    if (!isSuccess) return;
    form.setFields(
      spec.fields.map((field) => ({
        name: field.key,
        value: initialValues[field.key],
        touched: false,
      })),
    );
  }, [form, initialValues, isSuccess, spec.fields]);

  const handleSave = async () => {
    if (!canEditRef.current) {
      message.error(t('empty.loadFailed'));
      return;
    }
    try {
      const values = await form.validateFields();
      if (!canEditRef.current) {
        message.error(t('empty.loadFailed'));
        return;
      }
      const changedFields = spec.fields.filter((field) => form.isFieldTouched(field.key));
      if (changedFields.length === 0) {
        message.warning(t('sysconfig.warn.noSaveable'));
        return;
      }
      setSubmitting(true);
      await batchUpdate({
        category: spec.category,
        items: changedFields.map((f) => ({
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
        <Button size="small" type="primary" onClick={handleSave} loading={submitting} disabled={!canEdit}>
          {t('logCfg.save')}
        </Button>
      }
    >
      <div style={{ color: '#888', fontSize: 12, marginBottom: 12 }}>{t(spec.descKey)}</div>
      {isError && (
        <Alert
          type="error"
          showIcon
          title={t('empty.loadFailed')}
          description={t('empty.loadFailedDesc')}
          action={<Button size="small" onClick={() => void refetch()}>{t('common.retry')}</Button>}
          style={{ marginBottom: 12 }}
        />
      )}
      <Spin spinning={isLoading || isFetching}>
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
