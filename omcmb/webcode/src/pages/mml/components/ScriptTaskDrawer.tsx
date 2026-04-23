import { useCallback, useEffect, useState } from 'react';
import {
  Button,
  Checkbox,
  DatePicker,
  Divider,
  Drawer,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
  Space,
  Upload,
} from 'antd';
import { DownloadOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';

import { useCreateMMLTask, useCreateMMLScript } from '@core/hooks/api/useMML';
import type { MMLExecuteType } from '@core/types/mml';
import { useT } from '@/hooks/useT';
import { toast } from '@/utils/toast';

// -----------------------------------------------------------------------------
// "新建 MML 脚本任务" Drawer —— 被 ScriptTask / MML Console 两个入口复用，
// 布局严格对齐 docs/design/image-8.png。
//
// mode：
//   "task"   —— 提交到 mml_tasks（POST /api/v1/mml/tasks），用于 MML 控制台
//              保存脚本的"立即执行"以及一次性任务场景。
//   "script" —— 提交到 mml_scripts（POST /api/v1/mml/scripts），用于脚本任务
//              页面"+新增"（to-do-list 本轮 #1b）。执行策略/重试策略字段
//              不显示，只保留任务名 + 设备 SN + 文件。
// -----------------------------------------------------------------------------

export type ScriptTaskDrawerMode = 'task' | 'script';

export interface ScriptTaskDrawerProps {
  open: boolean;
  onClose: () => void;
  /** 决定确认按钮落到哪个表；默认 "task"（历史行为）。 */
  mode?: ScriptTaskDrawerMode;
  /**
   * MML Console 入口：直接把选中命令组装的命令行作为脚本内容。
   * 为 true 时不显示文件上传，内容只读展示并直接随任务提交。
   */
  prefillContent?: string;
  /** 预填任务名（例如由命令名 + 日期拼成） */
  prefillTaskName?: string;
  /**
   * 预填设备 SN 列表。Console 入口通常把左侧已勾选的设备全部带进来，
   * 避免用户在 Drawer 里重复输入一次。
   */
  prefillDeviceSns?: string[];
  /** 创建成功后回调（例如刷新外层列表、关闭父级 Modal 等） */
  onSuccess?: () => void;
}

interface TaskForm {
  taskName: string;
  fileName?: string;
  executeType: MMLExecuteType;
  scheduledAt?: Dayjs;
  periodRange?: [Dayjs, Dayjs];
  periodTime?: Dayjs;
  offlineRetryEnable: boolean;
  offlineRetryWaitTime: number;
  failedRetryEnable: boolean;
  failedRetryCount: number;
  failedRetryWaitTime: number;
}

const SECTION_DOT: React.CSSProperties = {
  display: 'inline-block',
  width: 4,
  height: 14,
  background: '#1890ff',
  borderRadius: 2,
  marginRight: 8,
  verticalAlign: 'middle',
};

const SECTION_HEADER: React.CSSProperties = {
  marginBottom: 8,
  fontWeight: 500,
  color: '#333',
};

export default function ScriptTaskDrawer({
  open,
  onClose,
  mode = 'task',
  prefillContent,
  prefillTaskName,
  prefillDeviceSns,
  onSuccess,
}: ScriptTaskDrawerProps) {
  const t = useT();
  const [form] = Form.useForm<TaskForm>();
  const executeType = Form.useWatch('executeType', form);
  const isScriptMode = mode === 'script';

  const [deviceSns, setDeviceSns] = useState<string[]>([]);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [parsedCommands, setParsedCommands] = useState<string[]>([]);

  const createTaskMutation = useCreateMMLTask();
  const createScriptMutation = useCreateMMLScript();
  const submitting = isScriptMode ? createScriptMutation.isPending : createTaskMutation.isPending;

  // 打开时初始化表单；Console 入口直接把 prefillContent 拆成命令数组。
  useEffect(() => {
    if (!open) return;
    form.resetFields();
    form.setFieldsValue({
      taskName: prefillTaskName || `MML任务_${dayjs().format('YYYY-MM-DD HH:mm:ss')}`,
      executeType: 'immediate',
      offlineRetryEnable: false,
      offlineRetryWaitTime: 60,
      failedRetryEnable: false,
      failedRetryCount: 3,
      failedRetryWaitTime: 5,
    });
    // 预填来自父组件的已选设备 SN；做一次排重避免重复项。
    setDeviceSns(prefillDeviceSns ? Array.from(new Set(prefillDeviceSns.filter(Boolean))) : []);
    setFileList([]);
    if (prefillContent) {
      setParsedCommands(splitScriptLines(prefillContent));
    } else {
      setParsedCommands([]);
    }
  }, [open, prefillContent, prefillTaskName, prefillDeviceSns, form]);

  const parseUploadedFile = useCallback((file: File) => {
    const reader = new FileReader();
    reader.onload = (event) => {
      const content = event.target?.result as string;
      setParsedCommands(splitScriptLines(content));
    };
    reader.readAsText(file);
  }, []);

  const handleDownloadTemplate = useCallback(() => {
    const templateContent = `# ============================================================
# MML 脚本任务模板 (to-do-list #6)
# ============================================================
#
# 【设备 SN 格式】
#   表单中的"设备 SN"字段支持批量输入，多个 SN 使用 "," 或 ";" 或
#   换行分隔，输入后自动识别为多个 Tag。示例：
#     CPE000001,CPE000002,CPE000003
#     或
#     CPE000001; CPE000002
#     或
#     CPE000001
#     CPE000002
#
# 【脚本命令格式】
#   - 一行一条 MML 命令
#   - 以 '#' 开头的行视为注释，不会被执行
#   - 空行将被忽略
#
# 【命令示例】
#   LST CELL                                # 查询小区列表
#   DSP EQUIPMENT                           # 显示设备信息
#   MOD CELL:modId={CellId=1,CellName=Foo}  # 修改小区参数
#   ACT CELL:CellId=1                       # 激活小区
#   RST DEVICE                              # 重启设备（慎用）
#
# ============================================================
# 在下方编写你的命令（删除本说明前的所有注释不会影响执行）：
# ============================================================

LST CELL
DSP EQUIPMENT
`;
    const blob = new Blob([templateContent], { type: 'text/plain;charset=utf-8' });
    const url = URL.createObjectURL(blob);
    const link = document.createElement('a');
    link.href = url;
    link.download = 'mml-script-template.txt';
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, []);

  // 返回 Promise 让 antd Modal/Drawer 可以 await，保证提交期间 UI 处于 loading
  // 状态、完成后再关闭；失败时保留弹窗供用户重试。toast 反馈走统一 toast util，
  // 避免 message 全局调用在某些场景下丢失（to-do-list 本轮 #2）。
  const handleSubmit = useCallback(() => {
    return form
      .validateFields()
      .then(async (values) => {
        // to-do-list 本轮 #1a：SN 与文件同时为空时，明确提示用户先填 SN。
        if (deviceSns.length === 0 && parsedCommands.length === 0) {
          toast.warning(t('mml.snRequired'));
          throw new Error('SN_REQUIRED');
        }

        if (isScriptMode) {
          // to-do-list 本轮 #1b：写 mml_scripts 表。执行策略字段不适用，保留
          // 最小集：script_name / description / content / tags(SNs) / status / type。
          await createScriptMutation.mutateAsync({
            scriptName: values.taskName.trim(),
            description: '',
            content: parsedCommands.join('\n'),
            creator: '',
            // 设备 SN 作为脚本标签附着，便于后续一键下发时复用。
            tags: deviceSns.map((sn) => `sn:${sn}`),
            status: 'active',
            type: 'manual',
            progress: 0,
          });
          toast.success(t('mml.scriptSaved'));
        } else {
          const payload = {
            taskName: values.taskName.trim(),
            deviceSns,
            commands: parsedCommands,
            creator: '',
            executeType: values.executeType,
            offlineRetry: values.offlineRetryEnable,
            offlineRetryWait: values.offlineRetryWaitTime,
            failedRetry: values.failedRetryEnable,
            failedRetryCount: values.failedRetryCount,
            failedRetryInterval: values.failedRetryWaitTime,
            scheduledAt:
              values.executeType === 'scheduled' && values.scheduledAt
                ? values.scheduledAt.toISOString()
                : undefined,
            periodStart:
              values.executeType === 'periodic' && values.periodRange?.[0]
                ? values.periodRange[0].toISOString()
                : undefined,
            periodEnd:
              values.executeType === 'periodic' && values.periodRange?.[1]
                ? values.periodRange[1].toISOString()
                : undefined,
            periodTime:
              values.executeType === 'periodic' && values.periodTime
                ? values.periodTime.format('HH:mm:ss')
                : undefined,
          };
          await createTaskMutation.mutateAsync(payload);
          toast.success(t('mml.taskCreated'));
        }

        onSuccess?.();
        onClose();
      })
      .catch((err) => {
        // 表单校验失败 / guard 抛出的业务前置错误会落到这里。antd 的
        // 校验错误 err 没有 message，静默即可；其它错误统一通过 toast 暴露。
        if (err && (err as { errorFields?: unknown }).errorFields) return;
        if (err instanceof Error && err.message === 'SN_REQUIRED') return;
        toast.error(err, t(isScriptMode ? 'mml.scriptSaveFailed' : 'mml.taskCreateFailedPrefix'));
      });
  }, [
    form,
    deviceSns,
    parsedCommands,
    isScriptMode,
    createTaskMutation,
    createScriptMutation,
    onSuccess,
    onClose,
    t,
  ]);

  return (
    <Drawer
      title={t(isScriptMode ? 'mml.newMmlScript' : 'mml.newMmlTask')}
      open={open}
      onClose={onClose}
      width={560}
      destroyOnClose
      footer={
        <div style={{ textAlign: 'right' }}>
          <Button onClick={onClose} style={{ marginRight: 8 }}>
            {t('common.cancel')}
          </Button>
          <Button type="primary" onClick={handleSubmit} loading={submitting}>
            {t('common.confirm')}
          </Button>
        </div>
      }
    >
      <Form form={form} layout="vertical">
        {/* ---- 基本信息 ---- */}
        <div style={SECTION_HEADER}>
          <span style={SECTION_DOT} />
          {t('mml.basicInfo')}
        </div>

        <Form.Item
          label={t('mml.taskName')}
          name="taskName"
          rules={[{ required: true, message: t('mml.inputTaskNameRequired') }]}
          style={{ marginLeft: 12 }}
        >
          <Input maxLength={50} placeholder={t('mml.inputTaskName')} style={{ width: '100%' }} />
        </Form.Item>

        <div style={{ marginLeft: 12, marginBottom: 16 }}>
          <label style={{ display: 'block', marginBottom: 4, fontSize: 14 }}>
            {t('mml.deviceSn')}
          </label>
          <Select
            mode="tags"
            value={deviceSns}
            onChange={setDeviceSns}
            placeholder={t('mml.inputDeviceSn')}
            style={{ width: '100%' }}
            tokenSeparators={[',', ';', '\n']}
            open={false}
          />
          <span style={{ color: '#999', fontSize: 12 }}>{t('mml.deviceSnTip')}</span>
        </div>

        {prefillContent !== undefined ? (
          // MML Console 入口：直接展示内容，不支持文件上传（命令已经在面板上选定）。
          <Form.Item label={t('mml.selectFile')} style={{ marginLeft: 12 }}>
            <Input.TextArea
              readOnly
              rows={5}
              value={prefillContent}
              style={{ fontFamily: "'SFMono-Regular', Consolas, monospace", fontSize: 12 }}
            />
            <span style={{ color: '#52c41a', fontSize: 12 }}>
              {t('mml.commandsParsed', { count: parsedCommands.length })}
            </span>
          </Form.Item>
        ) : (
          // ScriptTask 入口：文件上传
          // to-do-list #5：不再强制必填，用户可只输入 SN 提交空命令任务。
          // to-do-list #6：标签由"选择脚本"改为"选择文件"。
          <Form.Item
            label={t('mml.selectFile')}
            name="fileName"
            style={{ marginLeft: 12 }}
          >
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                <Upload
                  accept=".txt"
                  fileList={fileList}
                  beforeUpload={(file) => {
                    setFileList([file as unknown as UploadFile]);
                    form.setFieldValue('fileName', file.name);
                    parseUploadedFile(file);
                    return false;
                  }}
                  onRemove={() => {
                    setFileList([]);
                    form.setFieldValue('fileName', '');
                    setParsedCommands([]);
                  }}
                  maxCount={1}
                >
                  <Button icon={<UploadOutlined />}>{t('mml.selectFile')}</Button>
                </Upload>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.onlyTxtFormat')}</span>
              </Space>
              {parsedCommands.length > 0 && (
                <div style={{ color: '#52c41a', fontSize: 12 }}>
                  {t('mml.commandsParsed', { count: parsedCommands.length })}
                </div>
              )}
              <div>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.templateImportTip')}</span>
                <Button
                  type="link"
                  size="small"
                  icon={<DownloadOutlined />}
                  onClick={handleDownloadTemplate}
                >
                  {t('mml.exportTemplate')}
                </Button>
              </div>
            </Space>
          </Form.Item>
        )}

        {/* script 模式下不展示执行策略/重试策略，mml_scripts 表不持有这些字段 */}
        {!isScriptMode && (
        <>
        <Divider />

        {/* ---- 选择执行方式 ---- */}
        <div style={SECTION_HEADER}>
          <span style={SECTION_DOT} />
          {t('mml.selectExecuteMethod')}
        </div>

        <Form.Item name="executeType" style={{ marginBottom: 8, marginLeft: 12 }}>
          <Radio.Group>
            <Radio value="immediate">{t('mml.immediateExecute')}</Radio>
            <Radio value="suspended">{t('mml.suspended')}</Radio>
            <Space>
              <Radio value="scheduled">{t('mml.scheduledExecute')}</Radio>
              {executeType === 'scheduled' && (
                <Form.Item
                  name="scheduledAt"
                  noStyle
                  rules={[
                    {
                      required: executeType === 'scheduled',
                      message: t('mml.selectScheduledTime'),
                    },
                  ]}
                >
                  <DatePicker
                    showTime
                    format="YYYY-MM-DD HH:mm:ss"
                    disabledDate={(current) => current && current < dayjs().startOf('day')}
                    style={{ width: 185 }}
                  />
                </Form.Item>
              )}
            </Space>
          </Radio.Group>
        </Form.Item>

        <Form.Item style={{ marginBottom: 8, marginLeft: 12 }}>
          <Space align="start">
            <Radio
              value="periodic"
              checked={executeType === 'periodic'}
              onChange={() => form.setFieldValue('executeType', 'periodic')}
            >
              {t('mml.periodicTask')}
            </Radio>
            {executeType === 'periodic' && (
              <>
                <Form.Item
                  name="periodRange"
                  noStyle
                  rules={[
                    {
                      required: executeType === 'periodic',
                      message: t('mml.selectDateRange'),
                    },
                  ]}
                >
                  <DatePicker.RangePicker
                    disabledDate={(current) => current && current < dayjs().startOf('day')}
                    style={{ width: 240 }}
                  />
                </Form.Item>
                <span>:</span>
                <Form.Item
                  name="periodTime"
                  noStyle
                  rules={[
                    { required: executeType === 'periodic', message: t('mml.selectTime') },
                  ]}
                >
                  <DatePicker.TimePicker format="HH:mm:ss" style={{ width: 110 }} />
                </Form.Item>
              </>
            )}
          </Space>
        </Form.Item>

        <Divider />

        {/* ---- 执行策略 ---- */}
        <div style={SECTION_HEADER}>
          <span style={SECTION_DOT} />
          {t('mml.executionStrategy')}
        </div>

        <div style={{ marginLeft: 12, marginBottom: 16 }}>
          {t('mml.offlineDevice')}
          <Form.Item name="offlineRetryEnable" valuePropName="checked" noStyle>
            <Checkbox style={{ marginLeft: 8 }}>{t('mml.waitOnlineRetry')}</Checkbox>
          </Form.Item>
          <Form.Item name="offlineRetryWaitTime" noStyle>
            <InputNumber min={20} max={10080} style={{ width: 80, margin: '0 8px' }} />
          </Form.Item>
          {t('mml.minutes')}
        </div>

        <div style={{ marginLeft: 12, marginBottom: 8 }}>
          {t('mml.onlineDevice')}
          <Form.Item name="failedRetryEnable" valuePropName="checked" noStyle>
            <Checkbox style={{ marginLeft: 8 }}>{t('mml.failedRetry')}</Checkbox>
          </Form.Item>
          <Form.Item name="failedRetryCount" noStyle>
            <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
          </Form.Item>
          {t('mml.intervalRetry')}
          <Form.Item name="failedRetryWaitTime" noStyle>
            <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
          </Form.Item>
          {t('mml.minutes')}
        </div>
        </>
        )}
      </Form>
    </Drawer>
  );
}

// 把脚本内容按行拆分，剔除空行与 # 开头的注释行。
function splitScriptLines(content: string): string[] {
  return content
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter((line) => line && !line.startsWith('#'));
}
