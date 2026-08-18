import { useCallback, useMemo, useState } from 'react';
import { Alert, App, Checkbox, Input, Modal, Select, Space, Typography } from 'antd';
import { AxiosError } from 'axios';
import { useAddObject, useUpdateParameters } from '@core/hooks/api/useDeviceParameters';
import { deviceTaskApi } from '@core/services/api/deviceTaskApi';
import { isDeviceTaskTerminal } from '@core/types/deviceTask';
import type { ParameterUpdateRequest } from '@core/types/deviceParameter';
import type { QuickSettingsGroup, QuickSettingsParam } from '@core/types/quicksettings';
import { feedbackKey, useQuickSettingsFeedbackStore } from '@core/store/quickSettingsFeedbackStore';
import { BSC_BTS_FEEDBACK_GROUP_ID } from './bscBtsFeedback';
import {
  isBscCodecSupportParam,
  normalizeBscCodecSupportValue,
  parseQuickSettingsMultiCheckboxValue,
  resolveQuickSettingsParameterType,
  serializeBscCodecSupportValue,
  serializeQuickSettingsMultiCheckboxValue,
} from './validators';

const { Text } = Typography;

/**
 * 「新增 BTS」弹窗。
 *
 * 触发路径:QuickSettingsTab/index.tsx 顶部 BSC 实例选择器旁的「新增 BTS」按钮。
 *
 * 行为:
 *   1. 列出 BSC.xml bsc-bts-config / bsc-bts-handover 两组中预选的可配字段
 *      (与产品截图一致,而非 XML 全集)。
 *   2. 用户填写完成 → AddObject(DeviceGSM.Bts.) → 等待 task 完成,
 *      从 task.result.instance_number 读到新实例号 → SetParameterValues 批量下发。
 *   3. 各阶段 success / failure 分别用 message / notification 提示。
 */

/** 弹窗里展示的字段(leaf 名),按显示顺序排列。与截图保持一致。 */
const VISIBLE_LEAVES: string[] = [
  'LocationAreaCode',
  'Bsic',
  'IpaUnitId',
  'CellId',
  'MsMaxPower',
  'GprsMode',
  'NeighborListMode',
  'CodecSupport',
  'handover',
  'HandoverAlgorithm',
  'handover1.window.rxlev.averaging',
  'handover1.window.rxqual.averaging',
  'handover1.window.rxlev.neighbor.averaging',
  'handover1.power.budget.interval',
  'handover1.power.budget.hysteresis',
  'handover1.maximum.distance',
];

/** 弹窗默认值。空字符串表示无默认,留给用户填。 */
const DEFAULT_VALUES: Record<string, string> = {
  LocationAreaCode: '',
  Bsic: '',
  IpaUnitId: '',
  CellId: '',
  MsMaxPower: '31',
  GprsMode: 'none',
  NeighborListMode: 'automatic',
  // CodecSupport 页面展示 fr 必选,但下发时只发送 hr/efr/amr 的增量组合。
  // 设备 Get 会自动带回 fr；直接 Set fr 或 fr-* 会被 9007 拒绝。
  CodecSupport: 'fr',
  handover: '',
  HandoverAlgorithm: '1',
  'handover1.window.rxlev.averaging': '10',
  'handover1.window.rxqual.averaging': '10',
  'handover1.window.rxlev.neighbor.averaging': '10',
  'handover1.power.budget.interval': '6',
  'handover1.power.budget.hysteresis': '3',
  'handover1.maximum.distance': '9999',
};

const BSC_BTS_OBJECT_PREFIX = 'DeviceGSM.Bts.';
const ERROR_NOTIFICATION_DURATION = 4;
const TASK_POLL_TIMEOUT_MS = 60000;
const TASK_POLL_INTERVAL_MS = 1000;

interface BscBtsAddModalProps {
  open: boolean;
  deviceId: string;
  locale: 'zh-CN' | 'en-US';
  /** 用于查找 leaf 的元数据(titleZh/titleEn/required/min/max/enumOptions/...)。 */
  configGroup?: QuickSettingsGroup;
  handoverGroup?: QuickSettingsGroup;
  onClose: () => void;
  /** Add + SetParameterValues 全部成功后回调(刷新外层列表/切到新实例)。 */
  onSuccess?: (instanceNumber: number) => void;
}

/** 从两个 group 合并出 leaf → param 的索引(优先 config,缺时退回 handover)。 */
function buildLeafIndex(
  configGroup?: QuickSettingsGroup,
  handoverGroup?: QuickSettingsGroup,
): Map<string, QuickSettingsParam> {
  const idx = new Map<string, QuickSettingsParam>();
  for (const grp of [configGroup, handoverGroup]) {
    if (!grp) continue;
    for (const p of grp.params) {
      // 该组里的 leaf 名取自 standardPath 的末段(BSC.xml 全部用 standardPath 表达)。
      const path = p.standardPath ?? '';
      const dot = path.lastIndexOf('.');
      const leafFromPath = dot >= 0 ? path.slice(dot + 1) : path;
      const candidates = [p.name, leafFromPath, p.leaf ?? ''].filter(Boolean);
      for (const key of candidates) {
        if (!idx.has(key)) idx.set(key, p);
      }
    }
  }
  return idx;
}

/** 计算该字段在 SetParameterValues 里要拼的 leaf 路径段。
 *  BSC 几个 handover 字段本身带点(如 handover1.window.rxlev.averaging),不能只取 standardPath
 *  最后一段,必须取 `{i}.` 后面的所有段。 */
function leafPathSegment(param: QuickSettingsParam | undefined, fallbackLeaf: string): string {
  const sp = param?.standardPath ?? '';
  if (sp) {
    const m = sp.match(/\{i\}\.(.+)$/);
    if (m) return m[1];
    const dot = sp.lastIndexOf('.');
    if (dot >= 0) return sp.slice(dot + 1);
  }
  return fallbackLeaf;
}

/** 按字段类型校验。返回 '' 表示通过。 */
function validateField(param: QuickSettingsParam | undefined, value: string): string {
  if (!param) return '';
  if (param.required && value.trim() === '') return '不能为空';
  if (value === '') return '';
  if (param.type === 'int') {
    const n = Number(value);
    if (!Number.isInteger(n)) return '请输入整数';
    if (param.minValue !== undefined && n < param.minValue) return `不能小于 ${param.minValue}`;
    if (param.maxValue !== undefined && n > param.maxValue) return `不能大于 ${param.maxValue}`;
  }
  return '';
}

export function serializeBscBtsAddDeviceValue(param: QuickSettingsParam | undefined, value: string): string {
  const raw = String(value ?? '').trim();
  if (!param) return raw;
  if (param.type === 'multiCheckbox') {
    if (isBscCodecSupportParam(param.standardPath ?? param.name)) {
      return serializeBscCodecSupportValue(raw);
    }
    if (raw === '') return raw;
    return serializeQuickSettingsMultiCheckboxValue(raw);
  }
  if (raw === '') return raw;
  if (param.type === 'enum' && param.enumOptions && param.enumOptions.length > 0) {
    const direct = param.enumOptions.find((o) => o.value === raw);
    if (direct) return direct.value;
    const ci = raw.toLowerCase();
    const byValue = param.enumOptions.find((o) => o.value.toLowerCase() === ci);
    if (byValue) return byValue.value;
    const byLabel = param.enumOptions.find((o) => String(o.label ?? '').toLowerCase() === ci);
    if (byLabel) return byLabel.value;
  }
  return raw;
}

export function resolveBscBtsAddParameterType(param: QuickSettingsParam | undefined): ParameterUpdateRequest['parameterType'] {
  return resolveQuickSettingsParameterType(param?.type);
}

/** Range hint:与截图风格一致 "Range: lo-hi Integer"。 */
function formatRangeHint(param: QuickSettingsParam | undefined): string {
  if (!param || param.type !== 'int') return '';
  const min = param.minValue;
  const max = param.maxValue;
  if (min === undefined && max === undefined) return '';
  return `Range: ${min ?? '-∞'}-${max ?? '∞'} Integer`;
}

/** 提取后端错误信息,优先 axios response.data.message / .msg,然后 details, 最后 Error.message。 */
function extractErrMsg(e: unknown): string {
  if (e instanceof AxiosError) {
    const data = e.response?.data as { message?: string; msg?: string; details?: unknown } | undefined;
    const base = data?.message ?? data?.msg;
    if (base) {
      let detailText = '';
      if (data?.details) {
        try {
          detailText = ` (${typeof data.details === 'string' ? data.details : JSON.stringify(data.details)})`;
        } catch {
          detailText = '';
        }
      }
      return `${base}${detailText}`;
    }
    if (e.response?.status) return `HTTP ${e.response.status}: ${e.message}`;
  }
  if (e instanceof Error) return e.message;
  return String(e);
}

async function waitForTaskTerminal(taskId: string) {
  const deadline = Date.now() + TASK_POLL_TIMEOUT_MS;
  while (Date.now() < deadline) {
    const task = await deviceTaskApi.getTask(taskId);
    if (isDeviceTaskTerminal(task.status)) return task;
    await new Promise((r) => window.setTimeout(r, TASK_POLL_INTERVAL_MS));
  }
  throw new Error('等待任务完成超时');
}

export default function BscBtsAddModal({
  open,
  deviceId,
  locale,
  configGroup,
  handoverGroup,
  onClose,
  onSuccess,
}: BscBtsAddModalProps) {
  const leafIndex = useMemo(
    () => buildLeafIndex(configGroup, handoverGroup),
    [configGroup, handoverGroup],
  );

  // 取 ConfigProvider/<App> 上下文里的 scoped 实例 — antd v5 下静态 message/notification
  // 在某些场景脱离 context,导致 toast/notification 不出现。
  const { message, notification } = App.useApp();

  // BSC 顶部 Tag 用同一份 feedback store(与 Trx 行级 Tag 一致),Modal 内只负责
  // 写入"已入队"状态;终态由 QuickSettingsTab 顶层的 useDeviceTaskStatus 轮询展示。
  const setFeedback = useQuickSettingsFeedbackStore((s) => s.setFeedback);
  const fbKey = useMemo(
    () => feedbackKey(deviceId, BSC_BTS_FEEDBACK_GROUP_ID, 0),
    [deviceId],
  );

  const addMutation = useAddObject();
  const updateMutation = useUpdateParameters();
  const [values, setValues] = useState<Record<string, string>>(() => ({ ...DEFAULT_VALUES }));
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [apiError, setApiError] = useState<string>('');

  const resetState = useCallback(() => {
    setValues({ ...DEFAULT_VALUES });
    setErrors({});
    setApiError('');
  }, []);

  const handleCancel = useCallback(() => {
    if (addMutation.isPending || updateMutation.isPending) return;
    resetState();
    onClose();
  }, [addMutation.isPending, updateMutation.isPending, onClose, resetState]);

  const setValue = useCallback(
    (leaf: string, value: string) => {
      setValues((prev) => ({ ...prev, [leaf]: value }));
      const err = validateField(leafIndex.get(leaf), value);
      setErrors((prev) => ({ ...prev, [leaf]: err }));
    },
    [leafIndex],
  );

  const handleConfirm = useCallback(async () => {
    setApiError('');
    // 校验
    const nextErrors: Record<string, string> = {};
    for (const leaf of VISIBLE_LEAVES) {
      const err = validateField(leafIndex.get(leaf), values[leaf] ?? '');
      if (err) nextErrors[leaf] = err;
    }
    if (Object.keys(nextErrors).length > 0) {
      setErrors(nextErrors);
      message.error({ content: '校验失败,请修正后再保存', duration: ERROR_NOTIFICATION_DURATION });
      return;
    }

    // 1) AddObject
    let newInstanceNumber: number | undefined;
    let addTaskId: string | undefined;
    try {
      const addResp = await addMutation.mutateAsync({
        deviceId,
        objectPath: BSC_BTS_OBJECT_PREFIX,
      });
      addTaskId = addResp.taskId;
      const task = await waitForTaskTerminal(addResp.taskId);
      if (task.status !== 'completed') {
        throw new Error(task.errorMessage || `AddObject ${task.status}`);
      }
      const raw = task.result?.instance_number;
      if (typeof raw === 'number' && raw > 0) {
        newInstanceNumber = raw;
      }
    } catch (err) {
      const errMsg = extractErrMsg(err);
      setApiError(`BTS 新增失败:${errMsg}`);
      notification.error({
        message: 'BTS 新增失败',
        description: errMsg,
        duration: ERROR_NOTIFICATION_DURATION,
      });
      return;
    }

    if (!newInstanceNumber) {
      const errMsg = 'AddObject 已完成,但未能识别新实例号,请刷新页面后重试';
      setApiError(errMsg);
      notification.error({
        message: 'BTS 新增异常',
        description: errMsg,
        duration: ERROR_NOTIFICATION_DURATION,
      });
      return;
    }

    // 2) SetParameterValues:仅下发用户填了值的字段(空字符串跳过,避免覆盖设备侧默认值)
    const updates: ParameterUpdateRequest[] = [];
    for (const leaf of VISIBLE_LEAVES) {
      const param = leafIndex.get(leaf);
      const value = serializeBscBtsAddDeviceValue(param, values[leaf] ?? '');
      if (value === '') continue;
      const seg = leafPathSegment(param, leaf);
      const path = `${BSC_BTS_OBJECT_PREFIX}${newInstanceNumber}.${seg}`;
      const type = resolveBscBtsAddParameterType(param);
      updates.push({ parameterPath: path, parameterValue: value, parameterType: type });
    }

    if (updates.length === 0) {
      // 没填任何字段:AddObject 任务本身已 completed,写入 store 让顶部 Tag 立即显示
      // "新增 成功 · BTS #N"。
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'add',
        submitStatus: 'queued',
        taskId: addTaskId,
        detail: `BTS #${newInstanceNumber}`,
        at: Date.now(),
        instanceNumber: newInstanceNumber,
        addedInstIds: [String(newInstanceNumber)],
        readbackObjectPath: BSC_BTS_OBJECT_PREFIX,
      });
      resetState();
      onClose();
      onSuccess?.(newInstanceNumber);
      return;
    }

    try {
      const spvResp = await updateMutation.mutateAsync({ deviceId, parameters: updates });
      // 不在 Modal 里等 SPV 终态 —— 顶部 Tag 通过 useDeviceTaskStatus(spvResp.taskId)
      // 自动轮询展示 pending / sent / completed / failed,并在 failed 时把 errorMessage
      // 透给 formatDeviceFaultBrief 拆出被拒原因。
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'add',
        submitStatus: 'queued',
        taskId: spvResp.taskId,
        detail: `BTS #${newInstanceNumber} · ${updates.length} 项参数`,
        at: Date.now(),
        instanceNumber: newInstanceNumber,
        addedInstIds: [String(newInstanceNumber)],
        readbackObjectPath: BSC_BTS_OBJECT_PREFIX,
        expectedReadback: Object.fromEntries(
          updates.map((update) => [update.parameterPath, update.parameterValue]),
        ),
      });
      resetState();
      onClose();
      onSuccess?.(newInstanceNumber);
    } catch (err) {
      // SPV 入队失败(网络/校验返 400):此前已 AddObject 完成,实例存在;
      // 仍把状态写入 store 让顶部 Tag 显示 "新增 入队失败",同时 Modal 顶部 Alert + notification
      // 提示细节。
      const errMsg = extractErrMsg(err);
      setFeedback(fbKey, {
        kind: 'multi',
        action: 'add',
        submitStatus: 'failed_to_queue',
        detail: `BTS #${newInstanceNumber} · SPV 入队失败:${errMsg}`,
        at: Date.now(),
        instanceNumber: newInstanceNumber,
      });
      notification.error({
        message: 'BTS 参数下发失败',
        description: `实例 #${newInstanceNumber} 已创建,但 ${updates.length} 项参数下发入队失败:${errMsg}`,
        duration: ERROR_NOTIFICATION_DURATION,
      });
      // 实例已创建,关闭弹窗让用户在表单里继续调整
      resetState();
      onClose();
      onSuccess?.(newInstanceNumber);
    }
  }, [addMutation, deviceId, fbKey, leafIndex, message, notification, onClose, onSuccess, resetState, setFeedback, updateMutation, values]);

  const titleText = locale === 'zh-CN' ? '新增 BTS' : 'Add BTS';
  const okText = locale === 'zh-CN' ? '确认新增' : 'Add';
  const cancelText = locale === 'zh-CN' ? '取消' : 'Cancel';

  return (
    <Modal
      title={titleText}
      open={open}
      onOk={() => void handleConfirm()}
      onCancel={handleCancel}
      okText={okText}
      cancelText={cancelText}
      confirmLoading={addMutation.isPending || updateMutation.isPending}
      mask={{ closable: false }}
      destroyOnHidden
      width={760}
    >
      {apiError && (
        <Alert
          type="error"
          message={apiError}
          showIcon
          closable
          onClose={() => setApiError('')}
          style={{ marginBottom: 12 }}
        />
      )}
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(2, minmax(0, 1fr))',
          columnGap: 16,
          rowGap: 16,
          width: '100%',
        }}
      >
        {VISIBLE_LEAVES.map((leaf) => {
          const param = leafIndex.get(leaf);
          const label = (locale === 'zh-CN' ? param?.titleZh : param?.titleEn) || leaf;
          const required = param?.required === true;
          const value = values[leaf] ?? '';
          const err = errors[leaf] || '';
          const hint = formatRangeHint(param);

          const labelNode = (
            <div style={{ marginBottom: 6, fontWeight: 500 }}>
              {required && <span style={{ color: '#ff4d4f', marginRight: 4 }}>*</span>}
              <span>{label}</span>
            </div>
          );
          const hintNode = hint ? (
            <Text type="secondary" style={{ fontSize: 12, display: 'block', marginTop: 4 }}>
              {hint}
            </Text>
          ) : null;
          const errNode = err ? (
            <div style={{ color: '#ff4d4f', fontSize: 12, marginTop: 4 }}>{err}</div>
          ) : null;

          // multiCheckbox(CodecSupport)
          if (param?.type === 'multiCheckbox') {
            const opts = param.checkboxOptions ?? [];
            const selected = value
              ? isBscCodecSupportParam(param.standardPath ?? param.name)
                ? normalizeBscCodecSupportValue(value)
                : parseQuickSettingsMultiCheckboxValue(value)
              : [];
            const isCodecSupport = isBscCodecSupportParam(param.standardPath ?? param.name);
            return (
              <div key={leaf} style={{ minWidth: 0 }}>
                {labelNode}
                <Checkbox.Group
                  value={selected}
                  options={opts.map((v) => ({ value: v, label: v, disabled: isCodecSupport && v === 'fr' }))}
                  onChange={(vals) => setValue(
                    leaf,
                    (isCodecSupport ? normalizeBscCodecSupportValue(vals) : (vals as string[])).join('-'),
                  )}
                />
                {hintNode}
                {errNode}
              </div>
            );
          }

          // enum
          const enumOpts = param?.enumOptions ?? [];
          if (enumOpts.length > 0 || param?.type === 'enum') {
            return (
              <div key={leaf} style={{ minWidth: 0 }}>
                {labelNode}
                <Select
                  value={value || undefined}
                  onChange={(v) => setValue(leaf, String(v))}
                  style={{ width: '100%' }}
                  status={err ? 'error' : undefined}
                  options={enumOpts.map((o) => ({ value: o.value, label: o.label || o.value }))}
                  allowClear
                />
                {hintNode}
                {errNode}
              </div>
            );
          }

          // string / int
          return (
            <div key={leaf} style={{ minWidth: 0 }}>
              {labelNode}
              <Input
                value={value}
                onChange={(e) => setValue(leaf, e.target.value)}
                status={err ? 'error' : undefined}
              />
              {hintNode}
              {errNode}
            </div>
          );
        })}
      </div>
      <Space style={{ marginTop: 12 }}>
        <Text type="secondary" style={{ fontSize: 12 }}>
          {locale === 'zh-CN'
            ? '说明:点击确认后会先 AddObject 创建实例,再下发已填写的参数。各阶段结果会以提示框显示。'
            : 'Note: AddObject creates the instance first, then SetParameterValues delivers the filled fields. Both stages report success / failure via notification.'}
        </Text>
      </Space>
    </Modal>
  );
}
