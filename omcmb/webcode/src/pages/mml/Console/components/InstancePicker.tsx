/**
 * T-0130 InstancePicker 完整版 — GPV SSE 闭环 + supports_delete 🗑 提示。
 *
 * P2-b 简化版仅 InputNumber 手输；本期升级为：
 *   ① "探测实例"按钮 → POST /ops/commands/rpc (action="get_param") + SSE 订阅 mml_device_frame
 *   ② 探测成功 → antd Select options = 实例索引集合（用户选 index → store.setRmvIndex）
 *   ③ 探测失败/超时/多设备/未选设备 → 回退 InputNumber 手输 fallback（保 P2-b 兼容）
 *   ④ supports_delete=true 的实例旁 🗑 affordance 图标（提示该实例可删；点击高亮 setRmvIndex）
 *
 * 设计取舍：
 *   - 单设备探测：多设备时按钮 disabled，提示选单设备
 *   - supports_delete 仅作为 UI affordance，不直接派发 RMV 命令（用户仍走 DO 提交）
 *   - cleanup race-safe：useGPVProbe 内部 SSE close + 60s timeout 兜底
 */

import { useMemo } from 'react';
import { Button, InputNumber, Select, Space, Spin, Tooltip } from 'antd';
import { DeleteOutlined, ReloadOutlined } from '@ant-design/icons';

import { useMmlConsoleStore } from '@core/store/mmlConsoleStore';
import { useGPVProbe } from '@core/hooks/api/useGPVProbe';
import type { Statement } from '@core/types/mmlConsole';

import { useT } from '@/hooks/useT';

export interface InstancePickerProps {
  statement: Statement;
}

export default function InstancePicker({ statement }: InstancePickerProps) {
  const t = useT();
  const setRmvIndex = useMmlConsoleStore((s) => s.setRmvIndex);
  const selectedDeviceSns = useMmlConsoleStore((s) => s.selectedDeviceSns);

  // 单设备探测：多设备 / 0 设备时按钮 disabled
  const singleDeviceSn = selectedDeviceSns.length === 1 ? selectedDeviceSns[0] : undefined;
  const targetObject = statement.targetObject;

  const { probe, reset, status, instances, error } = useGPVProbe(
    singleDeviceSn,
    targetObject,
  );

  const probing = status === 'probing';
  const hasInstances = status === 'success' && instances.length > 0;
  const probeDisabled = !singleDeviceSn || !targetObject || probing;

  // supports_delete：RMV 命令通常有一个 target sub-field（or 兜底从首个 sub-field 取）
  // 仅当 sub_field 含 supportsDelete=true 时显示 🗑 affordance（PRD §6.4）。
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

  const handleSelect = (value: number | null) => {
    if (value === null || value === undefined) {
      setRmvIndex(statement.uid, undefined);
      return;
    }
    if (value < 0) return;
    setRmvIndex(statement.uid, value);
  };

  const handleReset = () => {
    reset();
    setRmvIndex(statement.uid, undefined);
  };

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
      </Space>

      {/* Row 2: Instance picker (Select if probed, else InputNumber fallback) */}
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ flex: '0 0 200px', fontWeight: 500 }}>
          <span style={{ color: '#ff4d4f', marginRight: 4 }} aria-label="required">
            *
          </span>
          {t('mml.console.picker.indexLabel')}
        </span>
        {hasInstances ? (
          <Select<number>
            value={statement.rmvInstanceIndex ?? undefined}
            onChange={(v) => handleSelect(v)}
            placeholder={t('mml.console.picker.selectInstance')}
            style={{ width: 200 }}
            allowClear
            options={instances.map((i) => ({
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
              value: i,
            }))}
            data-testid="instance-picker-select"
          />
        ) : (
          <InputNumber
            value={statement.rmvInstanceIndex ?? null}
            onChange={handleSelect}
            min={0}
            step={1}
            precision={0}
            style={{ width: 200 }}
            data-testid="instance-picker-input"
          />
        )}
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
