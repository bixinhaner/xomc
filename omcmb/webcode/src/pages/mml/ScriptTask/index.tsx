import { useState, useMemo, useCallback } from 'react';
import {
  Button,
  Checkbox,
  DatePicker,
  Divider,
  Drawer,
  Dropdown,
  Form,
  Input,
  InputNumber,
  Modal,
  Radio,
  Space,
  Tag,
  Upload,
  message,
} from 'antd';
import {
  PlusOutlined,
  EyeOutlined,
  MoreOutlined,
  PlayCircleOutlined,
  PauseCircleOutlined,
  StopOutlined,
  DeleteOutlined,
  InfoCircleOutlined,
  UploadOutlined,
  DownloadOutlined,
} from '@ant-design/icons';
import type { MenuProps, UploadFile } from 'antd';
import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import ListPageLayout from '@/components/Layout/ListPageLayout';
import DataTable from '@/components/DataTable';
import type { DataTableColumn } from '@/components/DataTable';
import FilterBar from '@/components/FilterBar';
import type { FilterField } from '@/components/FilterBar';
import { useMMLScripts } from '@/hooks/api/useMML';
import { useT } from '@/hooks/useT';

// 任务类型枚举
type CreateStatus = 'active' | 'suspend' | 'timing' | 'period';
// 任务状态枚举
type TaskStatus = 'waiting' | 'running' | 'paused' | 'completed' | 'terminated' | 'exception';
// 任务结果枚举
type TaskResult = 'success' | 'partial' | 'failed';

// 添加任务表单接口
interface AddTaskForm {
  taskName: string;
  fileName: string;
  status: CreateStatus;
  time: Dayjs | null;
  periodStartTime: Dayjs | null;
  periodEndTime: Dayjs | null;
  periodTime: Dayjs | null;
  offlineRetryEnable: boolean;
  offlineRetryWaitTime: number;
  failedRetryEnable: boolean;
  failedRetryCount: number;
  failedRetryWaitTime: number;
}

interface ScriptRow extends Record<string, unknown> {
  TASK_ID: string;
  TASK_NAME: string;
  CREATE_USER: string;
  CREATE_TIME: string;
  CREATE_STATUS: CreateStatus;
  TASK_STATUS: TaskStatus;
  TASK_PROGRESS: string;
  TASK_RESULT: TaskResult;
  START_TIME: string;
  END_TIME: string;
}

// 任务类型映射
const CREATE_STATUS_MAP: Record<CreateStatus, { color: string; text: string }> = {
  active: { color: 'green', text: '立即执行' },
  suspend: { color: 'orange', text: '挂起' },
  timing: { color: 'blue', text: '定时执行' },
  period: { color: 'purple', text: '周期任务' },
};

// 任务状态映射
const TASK_STATUS_MAP: Record<TaskStatus, { color: string; text: string }> = {
  waiting: { color: 'default', text: '等待中' },
  running: { color: 'processing', text: '执行中' },
  paused: { color: 'warning', text: '已暂停' },
  completed: { color: 'success', text: '已完成' },
  terminated: { color: 'error', text: '已终止' },
  exception: { color: 'error', text: '异常' },
};

// 任务结果映射
const TASK_RESULT_MAP: Record<TaskResult, { color: string; text: string }> = {
  success: { color: 'success', text: '成功' },
  partial: { color: 'warning', text: '部分成功' },
  failed: { color: 'error', text: '失败' },
};

// 任务类型选项
const CREATE_STATUS_OPTIONS = [
  { label: '立即执行', value: 'active' },
  { label: '挂起', value: 'suspend' },
  { label: '定时执行', value: 'timing' },
  { label: '周期任务', value: 'period' },
];

// 任务状态选项
const TASK_STATUS_OPTIONS = [
  { label: '等待中', value: 'waiting' },
  { label: '执行中', value: 'running' },
  { label: '已暂停', value: 'paused' },
  { label: '已完成', value: 'completed' },
  { label: '已终止', value: 'terminated' },
  { label: '异常', value: 'exception' },
];

// 任务结果选项
const TASK_RESULT_OPTIONS = [
  { label: '成功', value: 'success' },
  { label: '部分成功', value: 'partial' },
  { label: '失败', value: 'failed' },
];

const mockData: ScriptRow[] = [
  { TASK_ID: '1', TASK_NAME: '批量激活小区脚本', CREATE_USER: 'admin', CREATE_TIME: '2026-03-01 14:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-03-01 14:30:00', END_TIME: '2026-03-01 14:35:00' },
  { TASK_ID: '2', TASK_NAME: '5G频率配置脚本', CREATE_USER: 'operator1', CREATE_TIME: '2026-02-28 09:00:00', CREATE_STATUS: 'timing', TASK_STATUS: 'waiting', TASK_PROGRESS: '0%', TASK_RESULT: 'success', START_TIME: '2026-03-05 10:00:00', END_TIME: '' },
  { TASK_ID: '3', TASK_NAME: '告警清除脚本', CREATE_USER: 'operator2', CREATE_TIME: '2026-03-02 08:00:00', CREATE_STATUS: 'period', TASK_STATUS: 'running', TASK_PROGRESS: '45%', TASK_RESULT: 'partial', START_TIME: '2026-03-02 09:00:00', END_TIME: '' },
  { TASK_ID: '4', TASK_NAME: '邻区关系批量添加', CREATE_USER: 'admin', CREATE_TIME: '2026-03-01 15:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'exception', TASK_PROGRESS: '60%', TASK_RESULT: 'failed', START_TIME: '2026-03-01 16:00:00', END_TIME: '2026-03-01 16:45:00' },
  { TASK_ID: '5', TASK_NAME: '参数查询脚本', CREATE_USER: 'admin', CREATE_TIME: '2026-03-03 10:00:00', CREATE_STATUS: 'suspend', TASK_STATUS: 'paused', TASK_PROGRESS: '30%', TASK_RESULT: 'success', START_TIME: '2026-03-03 10:30:00', END_TIME: '' },
  { TASK_ID: '6', TASK_NAME: '基站重启脚本-北京区域', CREATE_USER: 'sysadmin', CREATE_TIME: '2026-03-10 08:00:00', CREATE_STATUS: 'timing', TASK_STATUS: 'waiting', TASK_PROGRESS: '0%', TASK_RESULT: 'success', START_TIME: '2026-03-15 02:00:00', END_TIME: '' },
  { TASK_ID: '7', TASK_NAME: '小区状态巡检脚本', CREATE_USER: 'operator1', CREATE_TIME: '2026-03-08 09:30:00', CREATE_STATUS: 'period', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-03-08 10:00:00', END_TIME: '2026-03-08 10:15:00' },
  { TASK_ID: '8', TASK_NAME: 'PCI冲突检测与修复', CREATE_USER: 'netadmin', CREATE_TIME: '2026-03-07 14:20:00', CREATE_STATUS: 'active', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'partial', START_TIME: '2026-03-07 15:00:00', END_TIME: '2026-03-07 15:30:00' },
  { TASK_ID: '9', TASK_NAME: '性能指标采集脚本', CREATE_USER: 'pmadmin', CREATE_TIME: '2026-03-06 11:00:00', CREATE_STATUS: 'period', TASK_STATUS: 'running', TASK_PROGRESS: '75%', TASK_RESULT: 'success', START_TIME: '2026-03-06 12:00:00', END_TIME: '' },
  { TASK_ID: '10', TASK_NAME: '4G/5G互操作参数配置', CREATE_USER: 'admin', CREATE_TIME: '2026-03-05 16:45:00', CREATE_STATUS: 'active', TASK_STATUS: 'terminated', TASK_PROGRESS: '20%', TASK_RESULT: 'failed', START_TIME: '2026-03-05 17:00:00', END_TIME: '2026-03-05 17:10:00' },
  { TASK_ID: '11', TASK_NAME: '批量修改发射功率', CREATE_USER: 'operator2', CREATE_TIME: '2026-03-04 13:30:00', CREATE_STATUS: 'active', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-03-04 14:00:00', END_TIME: '2026-03-04 14:25:00' },
  { TASK_ID: '12', TASK_NAME: 'TA配置同步脚本', CREATE_USER: 'netadmin', CREATE_TIME: '2026-03-03 18:00:00', CREATE_STATUS: 'timing', TASK_STATUS: 'waiting', TASK_PROGRESS: '0%', TASK_RESULT: 'success', START_TIME: '2026-03-10 03:00:00', END_TIME: '' },
  { TASK_ID: '13', TASK_NAME: '基站版本升级检查', CREATE_USER: 'sysadmin', CREATE_TIME: '2026-03-02 10:15:00', CREATE_STATUS: 'period', TASK_STATUS: 'exception', TASK_PROGRESS: '55%', TASK_RESULT: 'failed', START_TIME: '2026-03-02 11:00:00', END_TIME: '2026-03-02 11:45:00' },
  { TASK_ID: '14', TASK_NAME: 'X2接口配置脚本', CREATE_USER: 'operator1', CREATE_TIME: '2026-03-01 09:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'paused', TASK_PROGRESS: '40%', TASK_RESULT: 'partial', START_TIME: '2026-03-01 09:30:00', END_TIME: '' },
  { TASK_ID: '15', TASK_NAME: '负载均衡参数调整', CREATE_USER: 'admin', CREATE_TIME: '2026-02-28 15:30:00', CREATE_STATUS: 'suspend', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-02-28 16:00:00', END_TIME: '2026-02-28 16:20:00' },
  { TASK_ID: '16', TASK_NAME: 'PDCP参数优化脚本', CREATE_USER: 'optadmin', CREATE_TIME: '2026-03-09 10:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-03-09 10:30:00', END_TIME: '2026-03-09 11:00:00' },
  { TASK_ID: '17', TASK_NAME: 'RLC层配置批量更新', CREATE_USER: 'netadmin', CREATE_TIME: '2026-03-08 14:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'running', TASK_PROGRESS: '85%', TASK_RESULT: 'success', START_TIME: '2026-03-08 15:00:00', END_TIME: '' },
  { TASK_ID: '18', TASK_NAME: 'S1接口链路检测', CREATE_USER: 'operator2', CREATE_TIME: '2026-03-07 09:00:00', CREATE_STATUS: 'period', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'partial', START_TIME: '2026-03-07 09:30:00', END_TIME: '2026-03-07 09:45:00' },
  { TASK_ID: '19', TASK_NAME: 'QoS策略配置下发', CREATE_USER: 'qosadmin', CREATE_TIME: '2026-03-06 16:00:00', CREATE_STATUS: 'timing', TASK_STATUS: 'waiting', TASK_PROGRESS: '0%', TASK_RESULT: 'success', START_TIME: '2026-03-12 04:00:00', END_TIME: '' },
  { TASK_ID: '20', TASK_NAME: 'MAC层调度参数调整', CREATE_USER: 'admin', CREATE_TIME: '2026-03-05 11:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'paused', TASK_PROGRESS: '55%', TASK_RESULT: 'success', START_TIME: '2026-03-05 11:30:00', END_TIME: '' },
  { TASK_ID: '21', TASK_NAME: 'ANR自动邻区关系配置', CREATE_USER: 'netadmin', CREATE_TIME: '2026-03-04 08:00:00', CREATE_STATUS: 'period', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-03-04 08:30:00', END_TIME: '2026-03-04 09:00:00' },
  { TASK_ID: '22', TASK_NAME: 'PRB利用率统计采集', CREATE_USER: 'pmadmin', CREATE_TIME: '2026-03-03 15:00:00', CREATE_STATUS: 'period', TASK_STATUS: 'exception', TASK_PROGRESS: '30%', TASK_RESULT: 'failed', START_TIME: '2026-03-03 15:30:00', END_TIME: '2026-03-03 16:00:00' },
  { TASK_ID: '23', TASK_NAME: '切换参数优化脚本', CREATE_USER: 'optadmin', CREATE_TIME: '2026-03-02 13:00:00', CREATE_STATUS: 'suspend', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'partial', START_TIME: '2026-03-02 14:00:00', END_TIME: '2026-03-02 14:30:00' },
  { TASK_ID: '24', TASK_NAME: '干扰检测与上报脚本', CREATE_USER: 'operator1', CREATE_TIME: '2026-03-01 11:00:00', CREATE_STATUS: 'period', TASK_STATUS: 'running', TASK_PROGRESS: '90%', TASK_RESULT: 'success', START_TIME: '2026-03-01 12:00:00', END_TIME: '' },
  { TASK_ID: '25', TASK_NAME: '基站时钟同步检查', CREATE_USER: 'sysadmin', CREATE_TIME: '2026-02-28 10:00:00', CREATE_STATUS: 'active', TASK_STATUS: 'completed', TASK_PROGRESS: '100%', TASK_RESULT: 'success', START_TIME: '2026-02-28 10:30:00', END_TIME: '2026-02-28 10:45:00' },
];

export default function ScriptTask() {
  const t = useT();
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(20);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingRow, setEditingRow] = useState<ScriptRow | null>(null);
  const [resultModalVisible, setResultModalVisible] = useState(false);
  const [resultTaskName, setResultTaskName] = useState<string>('');
  const [filterParams, setFilterParams] = useState<Record<string, unknown>>({});
  const [addModalVisible, setAddModalVisible] = useState(false);
  const [fileList, setFileList] = useState<UploadFile[]>([]);
  const [addForm] = Form.useForm<AddTaskForm>();
  const executeType = Form.useWatch('status', addForm);

  const { data, isLoading, refetch } = useMMLScripts({ page, pageSize });

  // 根据筛选条件过滤数据
  const filteredData = useMemo(() => {
    const source = (data?.items ?? mockData) as unknown as ScriptRow[];
    return source.filter((row) => {
      if (filterParams.taskName && typeof filterParams.taskName === 'string') {
        if (!row.TASK_NAME.toLowerCase().includes(filterParams.taskName.toLowerCase())) return false;
      }
      if (filterParams.startTime && Array.isArray(filterParams.startTime) && filterParams.startTime.length === 2) {
        const [start, end] = filterParams.startTime as [string, string];
        if (row.START_TIME) {
          if (start && row.START_TIME < start) return false;
          if (end && row.START_TIME > end) return false;
        }
      }
      if (filterParams.createStatus && filterParams.createStatus !== 'all') {
        if (row.CREATE_STATUS !== filterParams.createStatus) return false;
      }
      if (filterParams.taskStatus && filterParams.taskStatus !== 'all') {
        if (row.TASK_STATUS !== filterParams.taskStatus) return false;
      }
      if (filterParams.taskResult && filterParams.taskResult !== 'all') {
        if (row.TASK_RESULT !== filterParams.taskResult) return false;
      }
      return true;
    });
  }, [data?.items, filterParams]);

  const filterFields: FilterField[] = useMemo(() => [
    { name: 'taskName', label: '任务名称', type: 'input', placeholder: '请输入任务名称' },
    { name: 'startTime', label: '开始时间', type: 'date-range' },
    { name: 'createStatus', label: '类型', type: 'select', placeholder: '请选择类型', options: [{ label: '全部', value: 'all' }, ...CREATE_STATUS_OPTIONS] },
    { name: 'taskStatus', label: '状态', type: 'select', placeholder: '请选择状态', options: [{ label: '全部', value: 'all' }, ...TASK_STATUS_OPTIONS] },
    { name: 'taskResult', label: '结果', type: 'select', placeholder: '请选择结果', options: [{ label: '全部', value: 'all' }, ...TASK_RESULT_OPTIONS] },
  ], []);

  const handleSearch = useCallback((values: Record<string, unknown>) => {
    setFilterParams(values);
    setPage(1);
  }, []);

  const handleReset = useCallback(() => {
    setFilterParams({});
    setPage(1);
  }, []);

  const showResult = (row: ScriptRow) => {
    setResultTaskName(row.TASK_NAME);
    setResultModalVisible(true);
  };

  const startTask = (row: ScriptRow) => {
    void message.success(`任务 ${row.TASK_NAME} 已启动`);
    void refetch();
  };

  const suspendTask = (row: ScriptRow) => {
    void message.success(`任务 ${row.TASK_NAME} 已暂停`);
    void refetch();
  };

  const terminateTask = (row: ScriptRow) => {
    void message.success(`任务 ${row.TASK_NAME} 已终止`);
    void refetch();
  };

  const deleteTask = (row: ScriptRow) => {
    void message.success(`任务 ${row.TASK_NAME} 已删除`);
    void refetch();
  };

  const viewTaskInfo = (row: ScriptRow) => {
    setEditingRow(row);
    setModalVisible(true);
  };

  const openAddModal = () => {
    addForm.resetFields();
    addForm.setFieldsValue({
      status: 'active',
      offlineRetryEnable: false,
      offlineRetryWaitTime: 60,
      failedRetryEnable: false,
      failedRetryCount: 3,
      failedRetryWaitTime: 5,
    });
    setFileList([]);
    setAddModalVisible(true);
  };

  const handleAddTask = () => {
    addForm.validateFields().then((values) => {
      console.log('Add task:', values, fileList);
      void message.success('任务创建成功');
      setAddModalVisible(false);
      void refetch();
    }).catch(() => undefined);
  };

  const handleDownloadTemplate = () => {
    void message.info('模板下载功能开发中...');
  };

  const getActionMenu = (row: ScriptRow): MenuProps['items'] => {
    const status = row.TASK_STATUS;
    return [
      { key: 'info', icon: <InfoCircleOutlined />, label: '信息', onClick: () => viewTaskInfo(row) },
      { key: 'start', icon: <PlayCircleOutlined />, label: '开始', disabled: status !== 'paused', onClick: () => startTask(row) },
      { key: 'wait', icon: <PauseCircleOutlined />, label: '暂停', disabled: status !== 'running', onClick: () => suspendTask(row) },
      { key: 'end', icon: <StopOutlined />, label: '终止任务', disabled: !['running', 'waiting'].includes(status), onClick: () => terminateTask(row) },
      { key: 'del', icon: <DeleteOutlined />, label: '删除', danger: true, disabled: status === 'running', onClick: () => deleteTask(row) },
    ];
  };

  const columns: DataTableColumn<ScriptRow>[] = useMemo(() => [
    {
      key: 'operation', title: '操作', dataIndex: 'TASK_ID', width: 70, fixed: 'left',
      render: (_, record) => (
        <Space size={4}>
          <Button type="link" size="small" icon={<EyeOutlined />} onClick={() => showResult(record)} title="查看结果" />
          <Dropdown menu={{ items: getActionMenu(record) }} trigger={['click']}>
            <Button type="link" size="small" icon={<MoreOutlined />} />
          </Dropdown>
        </Space>
      ),
    },
    { key: 'TASK_NAME', title: '任务名称', dataIndex: 'TASK_NAME', ellipsis: true },
    { key: 'CREATE_USER', title: '创建者', dataIndex: 'CREATE_USER', width: 100 },
    { key: 'CREATE_TIME', title: '创建时间', dataIndex: 'CREATE_TIME', width: 160 },
    { key: 'CREATE_STATUS', title: '类型', dataIndex: 'CREATE_STATUS', width: 100, render: (val: CreateStatus) => <Tag color={CREATE_STATUS_MAP[val]?.color}>{CREATE_STATUS_MAP[val]?.text}</Tag> },
    { key: 'TASK_STATUS', title: '状态', dataIndex: 'TASK_STATUS', width: 100, render: (val: TaskStatus) => <Tag color={TASK_STATUS_MAP[val]?.color}>{TASK_STATUS_MAP[val]?.text}</Tag> },
    { key: 'TASK_PROGRESS', title: '进度', dataIndex: 'TASK_PROGRESS', width: 80 },
    { key: 'TASK_RESULT', title: '结果', dataIndex: 'TASK_RESULT', width: 100, render: (val: TaskResult) => <Tag color={TASK_RESULT_MAP[val]?.color}>{TASK_RESULT_MAP[val]?.text}</Tag> },
    { key: 'START_TIME', title: '开始时间', dataIndex: 'START_TIME', width: 140 },
    { key: 'END_TIME', title: '结束时间', dataIndex: 'END_TIME', width: 140 },
  ], [t]);

  return (
    <ListPageLayout
      title={t('nav.mml.script')}
      extra={
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={openAddModal}>{t('common.add')}</Button>
        </Space>
      }
    >
      <FilterBar filterId="mml-script-task" fields={filterFields} onSearch={handleSearch} onReset={handleReset} />

      <DataTable<ScriptRow>
        tableId="script-task" columns={columns} dataSource={filteredData} loading={isLoading} rowKey="TASK_ID"
        total={filteredData.length} currentPage={page} pageSize={pageSize}
        onPageChange={(p, s) => { setPage(p); setPageSize(s); }} onRefresh={() => void refetch()} scroll={{ x: 1400 }}
      />

      {/* 任务详情弹窗 */}
      <Modal title="任务详情" open={modalVisible} onCancel={() => setModalVisible(false)} footer={null} width={680}>
        {editingRow && (
          <div style={{ padding: '16px 0' }}>
            <p><strong>任务名称：</strong>{editingRow.TASK_NAME}</p>
            <p><strong>创建者：</strong>{editingRow.CREATE_USER}</p>
            <p><strong>创建时间：</strong>{editingRow.CREATE_TIME}</p>
            <p><strong>类型：</strong>{CREATE_STATUS_MAP[editingRow.CREATE_STATUS]?.text}</p>
            <p><strong>状态：</strong>{TASK_STATUS_MAP[editingRow.TASK_STATUS]?.text}</p>
            <p><strong>进度：</strong>{editingRow.TASK_PROGRESS}</p>
            <p><strong>结果：</strong>{TASK_RESULT_MAP[editingRow.TASK_RESULT]?.text}</p>
            <p><strong>开始时间：</strong>{editingRow.START_TIME || '-'}</p>
            <p><strong>结束时间：</strong>{editingRow.END_TIME || '-'}</p>
          </div>
        )}
      </Modal>

      {/* 查看结果弹窗 */}
      <Modal title={`执行结果 - ${resultTaskName}`} open={resultModalVisible} onCancel={() => setResultModalVisible(false)} footer={null} width={900}>
        <div style={{ padding: '16px 0' }}>
          <p style={{ color: '#999' }}>任务执行结果列表将在此显示...</p>
        </div>
      </Modal>

      {/* 新建任务抽屉 */}
      <Drawer
        title="新建MML脚本任务"
        open={addModalVisible}
        onClose={() => setAddModalVisible(false)}
        width={560}
        destroyOnClose
        footer={
          <div style={{ textAlign: 'right' }}>
            <Button onClick={() => setAddModalVisible(false)} style={{ marginRight: 8 }}>取消</Button>
            <Button type="primary" onClick={handleAddTask}>确定</Button>
          </div>
        }
      >
        <Form form={addForm} layout="vertical">
          {/* 基本信息 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            基本信息
          </div>
          <Form.Item label="任务名称" name="taskName" rules={[{ required: true, message: '请输入任务名称' }]} style={{ marginLeft: 12 }}>
            <Input maxLength={50} placeholder="请输入新建任务名称" style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item label="选择脚本" name="fileName" rules={[{ required: true, message: '请先选择文件' }]} style={{ marginLeft: 12 }}>
            <Space direction="vertical" style={{ width: '100%' }}>
              <Space>
                <Upload
                  accept=".txt"
                  fileList={fileList}
                  beforeUpload={(file) => {
                    setFileList([file as unknown as UploadFile]);
                    addForm.setFieldValue('fileName', file.name);
                    return false;
                  }}
                  onRemove={() => {
                    setFileList([]);
                    addForm.setFieldValue('fileName', '');
                  }}
                  maxCount={1}
                >
                  <Button icon={<UploadOutlined />}>选择文件</Button>
                </Upload>
                <span style={{ color: '#999', fontSize: 12 }}>( 仅支持 .txt 格式 )</span>
              </Space>
              <div>
                <span style={{ color: '#999', fontSize: 12 }}>使用模板导入提示：支持使用模板导入</span>
                <Button type="link" size="small" icon={<DownloadOutlined />} onClick={handleDownloadTemplate}>导出模板</Button>
              </div>
            </Space>
          </Form.Item>

          <Divider />

          {/* 选择执行方式 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            选择执行方式
          </div>
          <Form.Item name="status" style={{ marginBottom: 8, marginLeft: 12 }}>
            <Radio.Group>
              <Radio value="active">立即执行</Radio>
              <Radio value="suspend">挂起</Radio>
              <Space>
                <Radio value="timing">定时执行</Radio>
                {executeType === 'timing' && (
                  <Form.Item name="time" noStyle rules={[{ required: executeType === 'timing', message: '请选择定时执行时间' }]}>
                    <DatePicker showTime format="YYYY-MM-DD HH:mm:ss" disabledDate={(current) => current && current < dayjs().startOf('day')} style={{ width: 185 }} />
                  </Form.Item>
                )}
              </Space>
            </Radio.Group>
          </Form.Item>
          <Form.Item style={{ marginBottom: 8, marginLeft: 12 }}>
            <Space align="start">
              <Radio value="period" checked={executeType === 'period'} onChange={() => addForm.setFieldValue('status', 'period')}>周期任务</Radio>
              {executeType === 'period' && (
                <>
                  <Form.Item name="periodDateRange" noStyle rules={[{ required: executeType === 'period', message: '请选择日期范围' }]}>
                    <DatePicker.RangePicker disabledDate={(current) => current && current < dayjs().startOf('day')} style={{ width: 240 }} />
                  </Form.Item>
                  <span>:</span>
                  <Form.Item name="periodTime" noStyle rules={[{ required: executeType === 'period', message: '请选择时间' }]}>
                    <DatePicker.TimePicker format="HH:mm:ss" style={{ width: 110 }} />
                  </Form.Item>
                </>
              )}
            </Space>
          </Form.Item>

          <Divider />

          {/* 执行策略 */}
          <div style={{ marginBottom: 8, fontWeight: 500, color: '#333' }}>
            <span style={{ display: 'inline-block', width: 4, height: 14, background: '#1890ff', borderRadius: 2, marginRight: 8, verticalAlign: 'middle' }} />
            执行策略
          </div>
          <div style={{ marginLeft: 12, marginBottom: 16 }}>
            离线设备
            <Form.Item name="offlineRetryEnable" valuePropName="checked" noStyle>
              <Checkbox style={{ marginLeft: 8 }}>等待设备上线重试</Checkbox>
            </Form.Item>
            <Form.Item name="offlineRetryWaitTime" noStyle>
              <InputNumber min={20} max={10080} style={{ width: 80, margin: '0 8px' }} />
            </Form.Item>
            分钟
          </div>
          <div style={{ marginLeft: 12, marginBottom: 8 }}>
            在线设备
            <Form.Item name="failedRetryEnable" valuePropName="checked" noStyle>
              <Checkbox style={{ marginLeft: 8 }}>配置失败重试</Checkbox>
            </Form.Item>
            <Form.Item name="failedRetryCount" noStyle>
              <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
            </Form.Item>
            间隔次数重试
            <Form.Item name="failedRetryWaitTime" noStyle>
              <InputNumber min={1} style={{ width: 70, margin: '0 8px' }} />
            </Form.Item>
            分钟
          </div>
        </Form>
      </Drawer>
    </ListPageLayout>
  );
}
