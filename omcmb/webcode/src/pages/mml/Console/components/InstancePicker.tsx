/**
 * T-0130 InstancePicker + R-7 多选删除。
 *
 * 历史路径：探测态用 antd Select 单选 + 探测失败回退 InputNumber 单值；
 * 升级后：统一 antd `Select mode="tags"` 同时支持
 *   ① 探测后从下拉勾选（多选）
 *   ② 探测失败/未启动时手输（粘贴 `1,3,5` 或回车追加）
 * 数据出口走 store.setRmvIndices；空选 → setRmvIndices(uid, undefined) 同时把 rmvInstanceIndex 也清掉。
 *
 * 设计取舍：
 *   - 单设备探测：多设备时按钮 disabled，提示选单设备
 *   - supports_delete 仅作为 UI affordance，不直接派发 RMV 命令
 *   - cleanup race-safe：useGPVProbe 内部 SSE close + 60s timeout 兜底
 *   - 验证：所有标签必须解析为非负整数；非法标签忽略（onChange 时过滤）
 */

import { useMemo } from 'react';
import type React from 'react';
import { Button, Select, Space, Spin, Tooltip } from 'antd';
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons';

import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import { useGPVProbe } from '@core/hooks/api/useGPVProbe';
import type { Statement } from '@core/types/mmlConsole';

import { useT } from '@/hooks/useT';
import { effectiveIndices, parseTagValues } from './instanceSelection';

export interface InstancePickerProps {
  statement: Statement;
}

// effectiveIndices / parseTagValues 已拆到同级 ./instanceSelection.ts
// （react-refresh/only-export-components：组件文件只导出组件）。

export default function InstancePicker({ statement }: InstancePickerProps): React.JSX.Element {
  const t = useT();
  const setRmvIndices = useMmlConsoleStore((s) => s.setRmvIndices);
  const selectedDeviceSns = useMmlConsoleStore((s) => s.selectedDeviceSns);

  const singleDeviceSn = selectedDeviceSns.length === 1 ? selectedDeviceSns[0] : undefined;
  const targetObject = statement.targetObject;

  const { probe, reset, status, instances, error } = useGPVProbe(
    singleDeviceSn,
    targetObject,
  );

  const probing = status === 'probing';
  const hasInstances = status === 'success' && instances.length > 0;
  const probeDisabled = !singleDeviceSn || !targetObject || probing;

  const supportsDelete = useMemo(
    () => statement.subFields.some((sf) => sf.supportsDelete),
    [statement.subFields],
  );

  const probeButtonHint = !singleDeviceSn
    ? t('mml.console.picker.singleDeviceOnly')
    : !targetObject
      ? t('mml.console.picker.noTargetObject')
      : probing
        ? t('mml.console.picker.probing')
        : t('mml.console.picker.probeButton');

  // tags 模式 value 用 string[]（antd 类型约束）；展示时再 stringify 一次保险。
  const currentIndices = effectiveIndices(statement);
  const tagValue = currentIndices.map((n) => String(n));

  const handleChange = (raw: string[]): void => {
    const parsed = parseTagValues(raw);
    setRmvIndices(statement.uid, parsed.length === 0 ? undefined : parsed);
  };

  const handleReset = (): void => {
    reset();
    setRmvIndices(statement.uid, undefined);
  };

  // 探测命中的实例集合作为下拉候选；fallback 时 options 空，纯手输。
  const options = useMemo(
    () =>
      instances.map((i) => ({
        label: supportsDelete ? (
          <Space size={4}>
            <span>{i}</span>
            <Tooltip title={t('mml.console.picker.deleteHint')}>
              <DeleteOutlined style={{ color: '#ff7875' }} />
            </Tooltip>
          </Space>
        ) : (
          String(i)
        ),
        value: String(i),
      })),
    [instances, supportsDelete, t],
  );

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
      {/* Row 1: Probe button + status */}
      <Space size={8} align="center" wrap>
        <Tooltip title={probeButtonHint}>
          <Button
            type="primary"
            ghost
            icon={probing ? <Spin size="small" /> : <ReloadOutlined />}
            disabled={probeDisabled}
            onClick={probe}
            data-testid="instance-picker-probe-btn"
          >
            {t('mml.console.picker.probeButton')}
          </Button>
        </Tooltip>
        {status === 'success' && (
          <span style={{ fontSize: 12, color: '#52c41a' }}>
            {t('mml.console.picker.probeSuccess', { count: instances.length })}
          </span>
        )}
        {(status === 'failed' || status === 'timeout') && (
          <Tooltip title={error}>
            <span style={{ fontSize: 12, color: '#ff4d4f' }}>
              {t('mml.console.picker.probeError')}
            </span>
          </Tooltip>
        )}
        {currentIndices.length > 1 && (
          <span style={{ fontSize: 12, color: '#1677ff' }}>
            {t('mml.console.picker.multiSelected', { count: currentIndices.length })}
          </span>
        )}
      </Space>

      {/* Row 2: 多选实例号输入 — tags 模式同时支持下拉勾选与手动粘贴 */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ flex: '0 0 200px', fontWeight: 500 }}>
          <span style={{ color: '#ff4d4f', marginRight: 4 }} aria-label="required">
            *
          </span>
          {t('mml.console.picker.indexLabel')}
        </span>
        <Select<string[]>
          mode="tags"
          value={tagValue}
          onChange={handleChange}
          placeholder={
            hasInstances
              ? t('mml.console.picker.selectInstance')
              : t('mml.console.picker.indexHelp')
          }
          style={{ width: 320 }}
          options={options}
          allowClear
          tokenSeparators={[',', ' ']}
          data-testid="instance-picker-select"
        />
        {hasInstances && (
          <Tooltip title={t('mml.console.picker.resetHint')}>
            <Button size="small" onClick={handleReset}>
              {t('mml.console.picker.reset')}
            </Button>
          </Tooltip>
        )}
      </div>

      <span style={{ fontSize: 12, color: '#999', paddingLeft: 208 }}>
        {hasInstances
          ? t('mml.console.picker.selectedHelp')
          : t('mml.console.picker.indexHelp')}
      </span>
    </div>
  );
}
