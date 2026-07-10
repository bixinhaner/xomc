import { useCallback, useEffect, useMemo, useState } from 'react';
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
  Table,
  Upload,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import { DownloadOutlined, PlusOutlined, UploadOutlined } from '@ant-design/icons';
import type { UploadFile } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';

import { useCreateMMLTask } from '@core/hooks/api/useMML';
import type { MMLExecuteType, MMLTaskPlanItem } from '@core/types/mml';
import { parseMmlScriptPlan } from '@core/utils/mmlScriptPlanParser';
import type { MMLScriptPlanParseResult } from '@core/utils/mmlScriptPlanParser';
import { useT } from '@/hooks/useT';
import { toast } from '@/utils/toast';
import DeviceSelectModal from '../Console/components/DeviceSelectModal';

// -----------------------------------------------------------------------------
// "新建 MML 脚本任务" Drawer —— 由 任务记录（TaskRecord）页"新建任务"入口调用；
// 同时保留 prefill* 入参，供 MML 控制台后续接入复用。布局对齐 docs/design/image-10.png。
//
// 一律提交到 mml_tasks（POST /api/v1/mml/tasks），包含：
//   基本信息（任务名 + 设备 SN + 脚本）
// + 执行方式（立即 / 挂起 / 定时 / 周期）
// + 执行策略（离线/在线重试）
// -----------------------------------------------------------------------------

export interface ScriptTaskDrawerProps {
  open: boolean;
  onClose: () => void;
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
  /**
   * 脚本任务页「执行」入口传入已存脚本的 ID（UUID）。
   * 提交时随 commands 一同发送给后端，后端将其保存在 mml_tasks.script_id，
   * 用于脚本执行历史关联及 last_run_status 回写。
   * BUG-06 fix (#706)
   */
  scriptId?: string;
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
  color: 'var(--color-neutral-700)',
};

const EMPTY_PARSE_RESULT: MMLScriptPlanParseResult = {
  executeMode: 'common',
  commands: [],
  planItems: [],
  deviceSns: [],
  warnings: [],
  errors: [],
};

export default function ScriptTaskDrawer({
  open,
  onClose,
  prefillContent,
  prefillTaskName,
  prefillDeviceSns,
  scriptId,
  onSuccess,
}: ScriptTaskDrawerProps) {
  const t = useT();
  const [form] = Form.useForm<TaskForm>();
  const executeType = Form.useWatch('executeType', form);

  const [deviceSns, setDeviceSns] = useState<string[]>([]);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [parseResult, setParseResult] = useState<MMLScriptPlanParseResult>(EMPTY_PARSE_RESULT);
  // Console 入口预填的命令文本同样允许用户手动调整（to-do-list 当轮 #6）。
  // 文件入口下，scriptContent 仅在解析完成后用于本地展示，不直接提交。
  const [scriptContent, setScriptContent] = useState('');
  // BUG-19：设备选择弹框控制
  const [deviceSelectOpen, setDeviceSelectOpen] = useState(false);

  const createTaskMutation = useCreateMMLTask();
  const submitting = createTaskMutation.isPending;
  const parsedCommands = parseResult.commands;
  const parsedPlanItems = parseResult.planItems;
  const isDeviceBound = parseResult.executeMode === 'device_bound';

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
    const initialDeviceSns = prefillDeviceSns ? Array.from(new Set(prefillDeviceSns.filter(Boolean))) : [];
    setFileList([]);
    const initial = prefillContent ?? '';
    setScriptContent(initial);
    const nextParse = initial ? parseMmlScriptPlan(initial, { format: 'text' }) : EMPTY_PARSE_RESULT;
    setParseResult(nextParse);
    setDeviceSns(nextParse.executeMode === 'device_bound' ? nextParse.deviceSns : initialDeviceSns);
  }, [open, prefillContent, prefillTaskName, prefillDeviceSns, form]);

  // 用户在 Console 入口手动改命令时，实时同步解析结果。
  const handleScriptContentChange = useCallback((value: string) => {
    setScriptContent(value);
    const nextParse = parseMmlScriptPlan(value, { format: 'text' });
    setParseResult(nextParse);
    if (nextParse.executeMode === 'device_bound') {
      setDeviceSns(nextParse.deviceSns);
    }
  }, []);

  // BUG-19：设备选择弹框确认回调——把选中的 SN 追加进来（去重）。
  const handleDeviceSelect = useCallback((sns: string[], _productId: string) => {
    setDeviceSns((prev) => Array.from(new Set([...prev, ...sns])));
    setDeviceSelectOpen(false);
  }, []);

  const parseUploadedFile = useCallback((file: File) => {
    const reader = new FileReader();
    reader.onload = (event) => {
      const content = event.target?.result as string;
      const format = file.name.toLowerCase().endsWith('.csv') ? 'csv' : 'text';
      const nextParse = parseMmlScriptPlan(content, { format });
      setScriptContent(content);
      setParseResult(nextParse);
      if (nextParse.executeMode === 'device_bound') {
        setDeviceSns(nextParse.deviceSns);
      }
    };
    reader.readAsText(file);
  }, []);

  const planPreviewColumns = useMemo<ColumnsType<MMLTaskPlanItem>>(
    () => [
      {
        title: t('mml.planLine'),
        dataIndex: 'lineNo',
        width: 82,
        render: (value: number, item) => `${value || '-'} / ${item.order || '-'}`,
      },
      {
        title: t('mml.deviceSn'),
        dataIndex: 'deviceSn',
        width: 150,
        ellipsis: true,
      },
      {
        title: t('mml.planCommand'),
        key: 'command',
        ellipsis: true,
        render: (_, item) => item.command.commandCode,
      },
    ],
    [t],
  );

  // 模板内容同步自 docs/design 附带的 MMLTemplate.txt（to-do-list 本轮 #2），
  // 覆盖内建与自定义 MML 命令的书写格式，与 ACS 解析一致。
  const handleDownloadTemplate = useCallback(() => {
    const templateContent = `# This is a MML script example,'#' defines a comment line, if you need to excute the command please delete the character '#', and specify the parameter values.
# You need to pay attention that one mml script must be on the same line and best not begin with blank.
# CELL_INDEX is a cell number,can be removed,the default is 1.
# Supports built-in commands and custom commands.
#### The following are examples of the built-in MML command formats.
# LST EUTRANNFREQ;{Serial Number}
# ADD EUTRANNFREQ:LTE_INTER_FREQ_DL_EARFCN={41390};{Serial Number}
# MOD EUTRANNFREQ:CELL_INDEX={1},i={2},LTE_INTER_FREQ_DL_EARFCN={41390};{Serial Number}
# RMV EUTRANNFREQ:CELL_INDEX={1},i={1};{Serial Number}
# MOD REMOTE_DEVICE:i={1},CRAN_EU_RU_RFTxStatus={true};{Serial Number}
# REBOOT CELL;{Serial Number}
# REBOOT_STK CELL;{Serial Number}
# REBOOT_RU CELL:i={15};{Serial Number}
# RESET CELL;{Serial Number}
# COLD_REBOOT CELL;{Serial Number}
# CLEAR IMSI;{Serial Number}
# RADIO_OPEN CELL;{Serial Number}
# RADIO_CLOSE CELL;{Serial Number}
#### The following is an example of the custom MML command formats.
## Example of a custom LST-type MML: Suppose the MML command group is named test_lst and contains three parameters named path1,path2, and path3.
## You can execute this custom MML using either of the following methods,where v1 and v3 represent path1 and path3 respectively.
# test_lst;{Serial Number}
# test_lst:v1,v3;{Serial Number}
## Example of a custom MOD type MML: Suppose the MML command group is named test_mod, containing two parameters path1 and path2, with corresponding modification values value1 and value2.
## You can execute this command in the following three ways, where value1 and value2 can be non-custom built-in modification parameters.
# test_mod;{Serial Numbner}
# test_mod:v1=name,v2=3;{Serial Numbner}
# test_mod:v1={name,name2},v2={3};{Serial Number}
## The following are custom ADD and RMV MML commands.
# test_add;{Serial Number}
# test_rmv;{Serial Number}
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
        if (isDeviceBound && parsedPlanItems.length === 0) {
          toast.warning(t('mml.noPlanItems'));
          throw new Error('PLAN_REQUIRED');
        }
        if (isDeviceBound && parseResult.warnings.length > 0) {
          toast.warning(t('mml.executeModeAllRowsRequireSn', {
            mode: t('mml.executeModeDeviceBound'),
          }));
          throw new Error('PLAN_MIXED');
        }
        if (!isDeviceBound && deviceSns.length === 0) {
          toast.warning(t('mml.snRequired'));
          throw new Error('SN_REQUIRED');
        }
        if (!isDeviceBound && parsedCommands.length === 0) {
          toast.warning(t('mml.selectFileFirst'));
          throw new Error('COMMAND_REQUIRED');
        }

        const payload = {
          taskName: values.taskName.trim(),
          deviceSns: isDeviceBound ? parseResult.deviceSns : deviceSns,
          commands: parsedCommands,
          executeMode: parseResult.executeMode,
          planItems: isDeviceBound ? parsedPlanItems : undefined,
          // BUG-06 fix (#706)：脚本任务页传入 scriptId 时随 commands 一同提交，
          // 后端保存 mml_tasks.script_id 用于历史关联与 last_run_status 回写。
          scriptId: scriptId ?? undefined,
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
        await createTaskMutation.mutateAsync(payload as Parameters<typeof createTaskMutation.mutateAsync>[0]);
        toast.success(t('mml.taskCreated'));

        onSuccess?.();
        onClose();
      })
      .catch((err) => {
        // 表单校验失败 / guard 抛出的业务前置错误会落到这里。antd 的
        // 校验错误 err 没有 message，静默即可；其它错误统一通过 toast 暴露。
        if (err && (err as { errorFields?: unknown }).errorFields) return;
        if (err instanceof Error && ['SN_REQUIRED', 'PLAN_REQUIRED', 'PLAN_MIXED', 'COMMAND_REQUIRED'].includes(err.message)) return;
        toast.error(err, t('mml.taskCreateFailedPrefix'));
      });
  }, [
    form,
    deviceSns,
    parsedCommands,
    parsedPlanItems,
    parseResult,
    isDeviceBound,
    scriptId,
    createTaskMutation,
    onSuccess,
    onClose,
    t,
  ]);

  return (
    <>
    <Drawer
      title={t('mml.newMmlTask')}
      open={open}
      onClose={onClose}
      size={760}
      destroyOnHidden
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
          rules={[
            { required: true, message: t('mml.inputTaskNameRequired') },
            { max: 128, message: t('mml.taskNameMaxLength') },
          ]}
          style={{ marginLeft: 12 }}
        >
          <Input
            maxLength={128}
            showCount
            placeholder={t('mml.inputTaskName')}
            style={{ width: '100%' }}
          />
        </Form.Item>

        <div style={{ marginLeft: 12, marginBottom: 16 }}>
          <label style={{ display: 'block', marginBottom: 4, fontSize: 14 }}>
            {t('mml.executeMode')}
          </label>
          <div style={{ border: '1px solid #d9d9d9', borderRadius: 8, padding: '12px 14px' }}>
            <div style={{ fontSize: 14, fontWeight: 600 }}>
              {isDeviceBound ? t('mml.executeModeDeviceBound') : t('mml.executeModeCommon')}
            </div>
            <div style={{ color: '#666', fontSize: 12, lineHeight: 1.6, marginTop: 4 }}>
              {isDeviceBound
                ? t('mml.executeModeDeviceBoundDescription')
                : t('mml.executeModeCommonDescription')}
            </div>
            <div style={{ color: '#999', fontSize: 12, marginTop: 6 }}>
              {isDeviceBound
                ? `${t('mml.planItemCount', { count: parsedPlanItems.length })} / ${t('mml.planDeviceCount', { count: parseResult.deviceSns.length })}`
                : t('mml.commandsParsed', { count: parsedCommands.length })}
            </div>
          </div>
        </div>

        {isDeviceBound ? (
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4, fontSize: 14 }}>
              {t('mml.deviceSn')}
            </label>
            <div style={{ color: '#999', fontSize: 12 }}>
              {t('mml.executeModeParsedDevices', {
                mode: t('mml.executeModeDeviceBound'),
                count: parseResult.deviceSns.length,
              })}
            </div>
          </div>
        ) : (
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            <label style={{ display: 'block', marginBottom: 4, fontSize: 14 }}>
              {t('mml.deviceSn')}
            </label>
            <Space.Compact style={{ width: '100%' }}>
              <Select
                mode="tags"
                value={deviceSns}
                onChange={setDeviceSns}
                placeholder={t('mml.inputDeviceSn')}
                style={{ flex: 1 }}
                tokenSeparators={[',', ';', '\n']}
                open={false}
              />
              <Button icon={<PlusOutlined />} onClick={() => setDeviceSelectOpen(true)}>
                {t('mml.selectDevice')}
              </Button>
            </Space.Compact>
            <span style={{ color: '#999', fontSize: 12 }}>{t('mml.deviceSnTip')}</span>
          </div>
        )}

        {prefillContent !== undefined ? (
          // MML Console 入口：预填的命令也允许用户手动微调（to-do-list 当轮 #6）。
          <Form.Item label={t('mml.scriptContent')} style={{ marginLeft: 12 }}>
            <Input.TextArea
              rows={5}
              value={scriptContent}
              onChange={(e) => handleScriptContentChange(e.target.value)}
              placeholder={t('mml.scriptDescTip')}
              style={{ fontFamily: "'SFMono-Regular', Consolas, monospace", fontSize: 12 }}
            />
            <div style={{ color: '#999', fontSize: 12, marginTop: 4 }}>
              {t('mml.scriptDescTip')}
            </div>
            <div style={{ color: '#52c41a', fontSize: 12 }}>
              {isDeviceBound
                ? t('mml.planItemsParsed', { count: parsedPlanItems.length })
                : t('mml.commandsParsed', { count: parsedCommands.length })}
            </div>
          </Form.Item>
        ) : (
          // ScriptTask 入口：文件上传（标签文案 "选择脚本"，image-10）。
          // 不强制必填，用户可只输入 SN 提交空命令任务。
          <Form.Item
            label={t('mml.selectScript')}
            name="fileName"
            style={{ marginLeft: 12 }}
          >
            <Space orientation="vertical" style={{ width: '100%' }}>
              <Space>
                <Upload
                  accept=".txt,.csv"
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
                    setParseResult(EMPTY_PARSE_RESULT);
                    setScriptContent('');
                  }}
                  maxCount={1}
                >
                  <Button icon={<UploadOutlined />}>{t('mml.selectFile')}</Button>
                </Upload>
                <span style={{ color: '#999', fontSize: 12 }}>{t('mml.txtOrCsvFormat')}</span>
              </Space>
              <div style={{ color: '#999', fontSize: 12 }}>{t('mml.scriptDescTip')}</div>
              {parsedCommands.length > 0 && (
                <div style={{ color: '#52c41a', fontSize: 12 }}>
                  {isDeviceBound
                    ? t('mml.planItemsParsed', { count: parsedPlanItems.length })
                    : t('mml.commandsParsed', { count: parsedCommands.length })}
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

        {isDeviceBound && (
          <Form.Item label={t('mml.planPreview')} style={{ marginLeft: 12 }}>
            <Table<MMLTaskPlanItem>
              size="small"
              rowKey={(item) => `${item.lineNo}-${item.deviceSn}-${item.order}`}
              columns={planPreviewColumns}
              dataSource={parsedPlanItems}
              pagination={parsedPlanItems.length > 8 ? { pageSize: 8, size: 'small' } : false}
            />
          </Form.Item>
        )}

        <Divider />

        {/* ---- 执行方式 ---- */}
        <div style={SECTION_HEADER}>
          <span style={SECTION_DOT} />
          {t('mml.executeMethod')}
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
            <Checkbox style={{ marginLeft: 8 }}>{t('common.enable')}</Checkbox>
          </Form.Item>
          <Form.Item name="offlineRetryWaitTime" noStyle>
            <InputNumber min={1} max={10080} style={{ width: 80, margin: '0 8px' }} />
          </Form.Item>
          {t('mml.minutesOnlineExecute')}
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
      </Form>
    </Drawer>

    {/* BUG-19：设备选择弹框 */}
    <DeviceSelectModal
      open={deviceSelectOpen}
      value={deviceSns}
      onCancel={() => setDeviceSelectOpen(false)}
      onConfirm={handleDeviceSelect}
    />
  </>
  );
}
