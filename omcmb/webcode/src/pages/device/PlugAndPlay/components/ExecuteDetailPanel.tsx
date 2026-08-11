import React, { useMemo, useCallback, useState } from 'react';
import { App, Button, Drawer, Space, Table, Tag, Typography } from 'antd';
import type { TableColumnsType } from 'antd';
import {
  CheckCircleOutlined,
  CloseCircleOutlined,
  LoadingOutlined,
  ClockCircleOutlined,
  ForwardOutlined,
} from '@ant-design/icons';
import { useT } from '@/hooks/useT';
import { provisionApi } from '@core/services/api/provisionApi';
import { buildProvisioningSteps } from '../provisioningSteps';

interface Props {
  taskId: string;
  taskData?: {
    status: string;
    executeProcedure: string;
    failureReason: string;
    startTime: string;
    endTime: string;
    currentStep: number;
    totalSteps: number;
    currentStepName?: string;
    xmlFileId?: string | null;
  };
  onClose: () => void;
}

interface TaskRecord {
  id: string;
  stepName: string;
  status: '0' | '1' | '2' | '3' | '4';
  startTime: string;
  endTime: string;
  failureReason: string;
}

const STATUS_CONFIG: Record<string, { color: string; icon: React.ReactNode }> = {
  '0': { color: 'success', icon: <CheckCircleOutlined /> },
  '1': { color: 'error', icon: <CloseCircleOutlined /> },
  '2': { color: 'processing', icon: <LoadingOutlined /> },
  '3': { color: 'default', icon: <ClockCircleOutlined /> },
  '4': { color: 'warning', icon: <ForwardOutlined /> },
};

export default function ExecuteDetailPanel({ taskData, onClose }: Props) {
  const t = useT();
  const { message } = App.useApp();
  const [downloading, setDownloading] = useState(false);
  const currentStepLabel = taskData?.currentStepName === 'completed'
    ? t('status.success')
    : taskData?.currentStepName === 'download_xml'
      ? `${t('common.download')} XML`
      : taskData?.currentStepName || '-';

  const getStatusText = useCallback((status: string) => {
    const map: Record<string, string> = {
      '0': t('status.success'),
      '1': t('status.failed'),
      '2': t('status.running'),
      '3': t('provision.pending'),
      '4': t('provision.skipped'),
    };
    return map[status] || status;
  }, [t]);

  const translateFailureReason = useCallback((reason: string) => {
    if (!reason) return '-';
    const map: Record<string, string> = {
      'license_download_failed': t('provision.licenseDownloadFailed'),
    };
    return map[reason] || reason;
  }, [t]);

  const stepNameMap: Record<string, string> = useMemo(() => ({
    'software_upgrade': t('provision.softwareUpgrade'),
    'license': t('provision.license'),
    'self_config': t('provision.selfConfig'),
    'match_policy': t('provision.matchPolicy'),
    'prepare_context': t('provision.prepareContext'),
    'generate_xml': t('provision.generateXml'),
    'upload_file': t('provision.uploadFile'),
    'download_xml': t('provision.downloadXml'),
    'wait_transfer_complete': t('provision.waitTransferComplete'),
    'wait_startup_stage': t('provision.waitStartupStage'),
    'parameter_validation': t('provision.parameterValidation'),
    'parameter_configuration': t('provision.parameterConfiguration'),
    'cell_activation': t('provision.cellActivation'),
    'wait_startup_result': t('provision.waitStartupResult'),
    'verify_online': t('provision.verifyOnline'),
    'completed': t('provision.orchestrationCompleted'),
  }), [t]);

  const downloadXML = useCallback(async () => {
    if (!taskData?.xmlFileId) return;
    setDownloading(true);
    try {
      await provisionApi.downloadXML(taskData.xmlFileId);
    } catch {
      void message.error(t('common.downloadFailed'));
    } finally {
      setDownloading(false);
    }
  }, [message, t, taskData?.xmlFileId]);

  // Parse executeProcedure into step records
  const records: TaskRecord[] = useMemo(() => {
    if (!taskData) return [];

    const procedure = taskData.executeProcedure || '';
    const taskStatus = taskData.status;
    const failureReason = taskData.failureReason;
    const taskStartTime = taskData.startTime;
    const taskEndTime = taskData.endTime;

    const orchestrationSteps = buildProvisioningSteps({
      status: taskStatus,
      currentStep: taskData.currentStep,
      totalSteps: taskData.totalSteps,
      failureReason,
    });
    if (orchestrationSteps.length > 0) {
      return orchestrationSteps.map((step, idx) => ({
        ...step,
        stepName: stepNameMap[step.stepName] || step.stepName,
        startTime: idx === 0 ? taskStartTime : '',
        endTime: idx === orchestrationSteps.length - 1 ? taskEndTime : '',
      }));
    }

    // Handle special procedure texts
    if (procedure === 'waiting') {
      return [{ id: '1', stepName: t('provision.waitingExecute'), status: '3', startTime: '', endTime: '', failureReason: '' }];
    }
    if (procedure === 'skipped_latest') {
      return [{ id: '1', stepName: t('provision.skippedLatestVersion'), status: '4', startTime: taskStartTime, endTime: taskEndTime, failureReason: '' }];
    }
    if (procedure.endsWith('_running')) {
      const baseStep = procedure.replace('_running', '');
      return [{ id: '1', stepName: stepNameMap[baseStep] || baseStep, status: '2', startTime: taskStartTime, endTime: '', failureReason: '' }];
    }

    // Parse normal procedure steps: "software_upgrade > license > self_config"
    const steps = procedure.split(' > ').filter(Boolean);
    const failedIdx = taskStatus === '1' ? steps.length - 1 : -1;

    return steps.map((step, idx) => ({
      id: String(idx + 1),
      stepName: stepNameMap[step] || step,
      status: idx < failedIdx ? '0' as const
        : idx === failedIdx ? '1' as const
        : taskStatus === '0' ? '0' as const : '0' as const,
      startTime: idx === 0 ? taskStartTime : '',
      endTime: idx === steps.length - 1 ? taskEndTime : '',
      failureReason: idx === failedIdx ? failureReason : '',
    }));
  }, [taskData, stepNameMap, t]);

  // Table columns
  const columns: TableColumnsType<TaskRecord> = useMemo(() => [
    {
      title: t('provision.progress'),
      dataIndex: 'stepName',
      key: 'stepName',
      width: 150,
    },
    {
      title: t('table.status'),
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: string) => {
        const cfg = STATUS_CONFIG[status] || STATUS_CONFIG['3'];
        return (
          <Tag color={cfg.color} icon={cfg.icon}>
            {getStatusText(status)}
          </Tag>
        );
      },
    },
    {
      title: t('provision.startTime'),
      dataIndex: 'startTime',
      key: 'startTime',
      width: 150,
      render: (val: string) => val || '-',
    },
    {
      title: t('provision.endTime'),
      dataIndex: 'endTime',
      key: 'endTime',
      width: 150,
      render: (val: string) => val || '-',
    },
    {
      title: t('provision.failureReason'),
      dataIndex: 'failureReason',
      key: 'failureReason',
      render: (reason: string) => reason ? translateFailureReason(reason) : '-',
    },
  ], [t, getStatusText, translateFailureReason]);

  return (
    <Drawer
      title={t('provision.executeDetail')}
      placement="right"
      size={600}
      open={true}
      onClose={onClose}
      styles={{ body: { padding: 16 } }}
    >
      {taskData?.xmlFileId && (
        <Space style={{ marginBottom: 16 }}>
          <Typography.Text>
            {t('provision.stepProgress')}: {currentStepLabel}
          </Typography.Text>
          <Button
            type="link"
            loading={downloading}
            onClick={() => void downloadXML()}
          >
            {t('common.download')}
          </Button>
        </Space>
      )}
      <Table
        columns={columns}
        dataSource={records}
        rowKey="id"
        size="small"
        pagination={false}
      />
    </Drawer>
  );
}
