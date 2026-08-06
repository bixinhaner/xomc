import { useEffect, useState } from 'react';
import {
  App,
  Alert,
  Button,
  Descriptions,
  Divider,
  Drawer,
  Input,
  Modal,
  Space,
  Spin,
  Tag,
  Tooltip,
  Typography,
  message as staticMessage,
} from 'antd';
import {
  DeleteOutlined,
  DownloadOutlined,
  PauseCircleOutlined,
  PlayCircleOutlined,
} from '@ant-design/icons';
import { useIntl } from 'react-intl';
import {
  useCreateGeofenceManualBindJob,
  useExportGeofenceBindings,
  useGeofenceBindings,
  useGeofenceControlActions,
  useGeofenceManualBindItems,
  useGeofenceManualBindJob,
  usePreviewGeofenceManualBindings,
  useRemoveGeofenceBinding,
  useResumeGeofenceBinding,
  useSuspendGeofenceBinding,
} from '@core/hooks/api/useGeofence';
import type {
  GeofenceBindingDetail,
  GeofenceManualBindingPreview,
  GeofenceMapDefinition,
} from '@core/types/geofence';

interface GeofenceBindingsDrawerProps {
  open: boolean;
  item?: GeofenceMapDefinition;
  initialDeviceSNs?: string[];
  onClose: () => void;
}

interface BindingAction {
  kind: 'suspend' | 'remove';
  binding: GeofenceBindingDetail;
}

function parseDeviceSNs(value: string): string[] {
  return Array.from(
    new Set(
      value
        .split(/[\s,;]+/)
        .map((part) => part.trim())
        .filter(Boolean),
    ),
  ).slice(0, 1000);
}

export default function GeofenceBindingsDrawer({
  open,
  item,
  initialDeviceSNs,
  onClose,
}: GeofenceBindingsDrawerProps) {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = appContext.message?.success
    ? appContext.message
    : staticMessage;
  const [inputText, setInputText] = useState('');
  const [reason, setReason] = useState('');
  const [preview, setPreview] =
    useState<GeofenceManualBindingPreview>();
  const [jobId, setJobId] = useState<string>();
  const [bindingAction, setBindingAction] =
    useState<BindingAction>();
  const [bindingActionReason, setBindingActionReason] =
    useState('');
  const geofenceId = item?.definition.id;
  const bindingsQuery = useGeofenceBindings(
    geofenceId,
    { page: 1, pageSize: 50 },
    { enabled: open },
  );
  const actionsQuery = useGeofenceControlActions(
    geofenceId,
    { enabled: open },
  );
  const previewMutation = usePreviewGeofenceManualBindings();
  const createJobMutation = useCreateGeofenceManualBindJob();
  const exportMutation = useExportGeofenceBindings();
  const jobQuery = useGeofenceManualBindJob(jobId, geofenceId);
  const itemsQuery = useGeofenceManualBindItems(
    jobId,
    { page: 1, pageSize: 50 },
    { enabled: Boolean(jobId) },
  );
  const suspendMutation = useSuspendGeofenceBinding();
  const resumeMutation = useResumeGeofenceBinding();
  const removeMutation = useRemoveGeofenceBinding();
  const deviceSNs = parseDeviceSNs(inputText);
  const executableCount = preview
    ? preview.eligibleCount + preview.moveCount
    : 0;

  useEffect(() => {
    if (!open) {
      setInputText('');
      setPreview(undefined);
      return;
    }
    setInputText(initialDeviceSNs?.join('\n') ?? '');
    setPreview(undefined);
  }, [initialDeviceSNs, open, item?.definition.id]);

  const beginBindingAction = (
    binding: GeofenceBindingDetail,
    kind: BindingAction['kind'],
  ) => {
    setBindingAction({ binding, kind });
    setBindingActionReason('');
  };

  const runBindingAction = async () => {
    if (!bindingAction || !bindingActionReason.trim()) return;
    try {
      const input = {
        id: bindingAction.binding.id,
        reason: bindingActionReason.trim(),
      };
      if (bindingAction.kind === 'suspend') {
        await suspendMutation.mutateAsync(input);
      } else {
        await removeMutation.mutateAsync(input);
      }
      setBindingAction(undefined);
      setBindingActionReason('');
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({
              id: 'geofence.message.operationFailed',
            }),
      );
    }
  };

  const resumeBinding = async (binding: GeofenceBindingDetail) => {
    try {
      await resumeMutation.mutateAsync(binding.id);
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({
              id: 'geofence.message.operationFailed',
            }),
      );
    }
  };

  const runPreview = async () => {
    if (!geofenceId || deviceSNs.length === 0) return;
    try {
      setPreview(
        await previewMutation.mutateAsync({
          id: geofenceId,
          inputs: { deviceSNs },
        }),
      );
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({
              id: 'geofence.message.operationFailed',
            }),
      );
    }
  };

  const createJob = async () => {
    if (
      !geofenceId ||
      !preview ||
      executableCount === 0 ||
      !reason.trim()
    ) {
      return;
    }
    try {
      const accepted = await createJobMutation.mutateAsync({
        id: geofenceId,
        input: {
          deviceSNs,
          previewFingerprint: preview.previewFingerprint,
          reason: reason.trim(),
        },
      });
      setJobId(accepted.jobId);
      void message.success(
        intl.formatMessage({
          id: 'geofence.message.bindingJobCreated',
        }),
      );
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({
              id: 'geofence.message.operationFailed',
            }),
      );
    }
  };

  const exportBindings = async () => {
    if (!geofenceId) return;
    try {
      await exportMutation.mutateAsync({
        id: geofenceId,
        filter: {},
      });
      void message.success(
        intl.formatMessage({
          id: 'geofence.message.bindingsExported',
        }),
      );
    } catch (error) {
      void message.error(
        error instanceof Error
          ? error.message
          : intl.formatMessage({
              id: 'geofence.message.operationFailed',
            }),
      );
    }
  };

  const completed = jobQuery.data
    ? jobQuery.data.progress.total - jobQuery.data.progress.pending
    : 0;

  return (
    <Drawer
      rootClassName="geofence-drawer"
      open={open}
      size={560}
      title={`${intl.formatMessage({
        id: 'geofence.binding.title',
      })}${item ? ` · ${item.definition.name}` : ''}`}
      onClose={onClose}
    >
      <div className="geofence-binding-section-heading">
        <strong>
          {intl.formatMessage({
            id: 'geofence.binding.currentTitle',
          })}
        </strong>
        <Tooltip
          title={intl.formatMessage({
            id: 'geofence.action.exportBindings',
          })}
        >
          <Button
            type="text"
            size="small"
            icon={<DownloadOutlined aria-hidden />}
            aria-label={intl.formatMessage({
              id: 'geofence.action.exportBindings',
            })}
            loading={exportMutation.isPending}
            disabled={!geofenceId}
            onClick={() => void exportBindings()}
          />
        </Tooltip>
      </div>
      {bindingsQuery.isLoading ? (
        <Spin style={{ display: 'block', margin: 24 }} />
      ) : bindingsQuery.isError ? (
        <Alert
          type="error"
          showIcon
          title={intl.formatMessage({
            id: 'geofence.message.loadFailed',
          })}
          action={
            <Button
              size="small"
              onClick={() => void bindingsQuery.refetch()}
            >
              {intl.formatMessage({ id: 'common.retry' })}
            </Button>
          }
        />
      ) : (
        <div className="geofence-binding-list">
          {(bindingsQuery.data?.items ?? []).map((binding) => (
            <article className="geofence-binding-card" key={binding.id}>
              <div className="geofence-binding-card-header">
                <div>
                  <strong>{binding.deviceName}</strong>
                  <Typography.Text type="secondary">
                    {binding.deviceSN}
                  </Typography.Text>
                </div>
                <Space size="small" wrap>
                  <Tag color={binding.status === 'active' ? 'success' : 'default'}>
                    {intl.formatMessage({
                      id: `geofence.bindingStatus.${binding.status}`,
                    })}
                  </Tag>
                  <Tag
                    color={
                      binding.evaluation.lastObservedAt
                        ? binding.evaluation.confirmedState === 'inside'
                          ? 'green'
                          : binding.evaluation.confirmedState === 'outside'
                            ? 'red'
                            : 'default'
                        : 'default'
                    }
                  >
                    {intl.formatMessage({
                      id: binding.evaluation.lastObservedAt
                        ? `geofence.evaluation.confirmed.${binding.evaluation.confirmedState}`
                        : 'geofence.evaluation.notEvaluated',
                    })}
                  </Tag>
                  {binding.evaluation.evaluationHealth === 'failed' && (
                    <Tag color="error">
                      {intl.formatMessage({
                        id: 'geofence.evaluation.failed',
                      })}
                    </Tag>
                  )}
                  {binding.evaluation.candidateState && (
                    <Tag color="processing">
                      {intl.formatMessage(
                        {
                          id: `geofence.evaluation.candidate.${binding.evaluation.candidateState}`,
                        },
                        { count: binding.evaluation.candidateCount },
                      )}
                    </Tag>
                  )}
                </Space>
              </div>
              <div className="geofence-binding-card-footer">
                <Space size="small" wrap className="geofence-binding-card-meta">
                  {binding.evaluation.lastDistanceToBoundary !==
                    undefined && (
                    <Typography.Text type="secondary">
                      {intl.formatMessage(
                        {
                          id: 'geofence.evaluation.distance',
                        },
                        {
                          distance:
                            binding.evaluation.lastDistanceToBoundary.toFixed(
                              1,
                            ),
                        },
                      )}
                    </Typography.Text>
                  )}
                  {binding.evaluation.lastObservedAt && (
                    <Typography.Text type="secondary">
                      {intl.formatMessage(
                        {
                          id: 'geofence.evaluation.lastObservedAt',
                        },
                        {
                          time: intl.formatDate(
                            binding.evaluation.lastObservedAt,
                            {
                              year: 'numeric',
                              month: 'numeric',
                              day: 'numeric',
                              hour: '2-digit',
                              minute: '2-digit',
                              second: '2-digit',
                              hour12: false,
                            },
                          ),
                        },
                      )}
                    </Typography.Text>
                  )}
                  {binding.evaluation.lastObservationVersion !==
                    undefined && (
                    <Typography.Text type="secondary">
                      {intl.formatMessage(
                        {
                          id: 'geofence.evaluation.observationVersion',
                        },
                        {
                          version:
                            binding.evaluation.lastObservationVersion,
                        },
                      )}
                    </Typography.Text>
                  )}
                  {binding.evaluation.errorCode && (
                    <Typography.Text type="danger">
                      {intl.formatMessage(
                        {
                          id: 'geofence.evaluation.errorCode',
                        },
                        { code: binding.evaluation.errorCode },
                      )}
                    </Typography.Text>
                  )}
                </Space>
                <Space size={4} className="geofence-binding-card-actions">
                  {binding.status === 'active' ? (
                    <Tooltip
                      title={intl.formatMessage({
                        id: 'geofence.action.suspendBinding',
                      })}
                    >
                      <Button
                        type="text"
                        size="small"
                        icon={<PauseCircleOutlined aria-hidden />}
                        aria-label={intl.formatMessage({
                          id: 'geofence.action.suspendBinding',
                        })}
                        onClick={() =>
                          beginBindingAction(binding, 'suspend')
                        }
                      />
                    </Tooltip>
                  ) : binding.status === 'suspended' ? (
                    <Tooltip
                      title={intl.formatMessage({
                        id: 'geofence.action.resumeBinding',
                      })}
                    >
                      <Button
                        type="text"
                        size="small"
                        icon={<PlayCircleOutlined aria-hidden />}
                        aria-label={intl.formatMessage({
                          id: 'geofence.action.resumeBinding',
                        })}
                        onClick={() => void resumeBinding(binding)}
                      />
                    </Tooltip>
                  ) : null}
                  {(binding.status === 'active' ||
                    binding.status === 'suspended') && (
                    <Tooltip
                      title={intl.formatMessage({
                        id: 'geofence.action.removeBinding',
                      })}
                    >
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined aria-hidden />}
                        aria-label={intl.formatMessage({
                          id: 'geofence.action.removeBinding',
                        })}
                        onClick={() =>
                          beginBindingAction(binding, 'remove')
                        }
                      />
                    </Tooltip>
                  )}
              </Space>
              </div>
            </article>
          ))}
        </div>
      )}

      <details className="geofence-control-results">
        <summary>
          {intl.formatMessage({ id: 'geofence.control.title' })}
          <Tag>{(actionsQuery.data ?? []).length}</Tag>
        </summary>
        <Descriptions column={1} size="small">
        {actionsQuery.isLoading ? (
          <Descriptions.Item>
            <Spin size="small" />
          </Descriptions.Item>
        ) : actionsQuery.isError ? (
          <Descriptions.Item>
            <Typography.Text type="danger">
              {intl.formatMessage({ id: 'geofence.message.loadFailed' })}
            </Typography.Text>
          </Descriptions.Item>
        ) : (actionsQuery.data ?? []).length === 0 ? (
          <Descriptions.Item>
            {intl.formatMessage({ id: 'geofence.control.empty' })}
          </Descriptions.Item>
        ) : (
          (actionsQuery.data ?? []).map((action) => (
            <Descriptions.Item
              key={action.id}
              label={action.deviceSN}
            >
              <Space size="small" wrap>
                <Tag>{action.actionType === 'activate'
                  ? intl.formatMessage({ id: 'geofence.control.activate' })
                  : intl.formatMessage({ id: 'geofence.control.deactivate' })}</Tag>
                <Tag color={action.status === 'verified' ? 'success' : action.status === 'failed' || action.status === 'partial_failed' ? 'error' : 'processing'}>
                  {intl.formatMessage({
                    id: `geofence.control.status.${action.status}`,
                  })}
                </Tag>
                {action.requestedState.map((parameter) => (
                  <Typography.Text type="secondary" key={`requested-${parameter.path}`}>
                    {intl.formatMessage({ id: 'geofence.control.requested' })}:{' '}
                    {parameter.path} = {parameter.value}
                  </Typography.Text>
                ))}
                {action.verifiedState.map((parameter) => (
                  <Typography.Text type="secondary" key={`verified-${parameter.path}`}>
                    {intl.formatMessage({ id: 'geofence.control.verified' })}:{' '}
                    {parameter.path} = {parameter.value}
                  </Typography.Text>
                ))}
                {action.lastError && (
                  <Typography.Text type="danger">{action.lastError}</Typography.Text>
                )}
              </Space>
            </Descriptions.Item>
          ))
        )}
        </Descriptions>
      </details>

      <Divider
        className="geofence-binding-batch-divider"
        titlePlacement="start"
        plain
      >
        {intl.formatMessage({ id: 'geofence.binding.batchTitle' })}
      </Divider>

      <Input.TextArea
        aria-label={intl.formatMessage({
          id: 'geofence.binding.inputPlaceholder',
        })}
        value={inputText}
        rows={3}
        placeholder={intl.formatMessage({
          id: 'geofence.binding.inputPlaceholder',
        })}
        onChange={(event) => {
          setInputText(event.target.value);
          setPreview(undefined);
        }}
      />
      <Typography.Text type="secondary" style={{ display: 'block', marginTop: 6, fontSize: 12 }}>
        {intl.formatMessage({ id: 'geofence.binding.inputHint' })}
      </Typography.Text>
      <Button
        style={{ marginTop: 12 }}
        disabled={!geofenceId || deviceSNs.length === 0}
        loading={previewMutation.isPending}
        onClick={() => void runPreview()}
      >
        {intl.formatMessage({
          id: 'geofence.binding.previewAction',
        })}
      </Button>

      {preview && (
        <div style={{ marginTop: 16 }}>
          <Space>
            <Tag color="green">
              {intl.formatMessage({ id: 'geofence.binding.eligible' })}:{' '}
              {preview.eligibleCount}
            </Tag>
            <Tag color="blue">
              {intl.formatMessage({ id: 'geofence.binding.move' })}:{' '}
              {preview.moveCount}
            </Tag>
            <Tag>
              {intl.formatMessage({ id: 'geofence.binding.skipped' })}:{' '}
              {preview.skippedCount}
            </Tag>
          </Space>
          {preview.moveCount > 0 && (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 12 }}
              title={intl.formatMessage({
                id: 'geofence.binding.moveWarning',
              })}
            />
          )}
          {executableCount === 0 && (
            <Alert
              type="warning"
              showIcon
              style={{ marginTop: 12 }}
              title={intl.formatMessage({
                id: 'geofence.binding.noEligibleDevices',
              })}
            />
          )}
          <Input.TextArea
            aria-label={intl.formatMessage({
              id: 'geofence.field.reason',
            })}
            value={reason}
            rows={2}
            style={{ marginTop: 12 }}
            placeholder={intl.formatMessage({
              id: 'geofence.lifecycle.reasonRequired',
            })}
            onChange={(event) => setReason(event.target.value)}
          />
          <Button
            type="primary"
            style={{ marginTop: 12 }}
            disabled={executableCount === 0 || !reason.trim()}
            loading={createJobMutation.isPending}
            onClick={() => void createJob()}
          >
            {intl.formatMessage({
              id: 'geofence.binding.createJob',
            })}
          </Button>
        </div>
      )}

      {jobQuery.data && (
        <Descriptions
          column={1}
          size="small"
          title={intl.formatMessage({ id: 'geofence.job.title' })}
          style={{ marginTop: 20 }}
        >
          <Descriptions.Item
            label={intl.formatMessage({
              id: `geofence.job.status.${jobQuery.data.status}`,
            })}
          >
            {intl.formatMessage(
              { id: 'geofence.job.progress' },
              {
                completed,
                total: jobQuery.data.progress.total,
              },
            )}
          </Descriptions.Item>
          {itemsQuery.data?.items.map((jobItem) => (
            <Descriptions.Item
              key={jobItem.id}
              label={jobItem.deviceSN ?? jobItem.inputValue}
            >
              {intl.formatMessage({
                id: `geofence.job.itemStatus.${jobItem.status}`,
              })}
            </Descriptions.Item>
          ))}
        </Descriptions>
      )}
      <Modal
        open={Boolean(bindingAction)}
        title={intl.formatMessage({
          id: 'geofence.lifecycle.previewTitle',
        })}
        okText={intl.formatMessage({ id: 'common.confirm' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        okButtonProps={{
          disabled: !bindingActionReason.trim(),
        }}
        confirmLoading={
          suspendMutation.isPending || removeMutation.isPending
        }
        onCancel={() => setBindingAction(undefined)}
        onOk={() => void runBindingAction()}
      >
        {bindingAction?.kind === 'remove' && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 12 }}
            title={intl.formatMessage({
              id: 'geofence.message.removeBindingDoesNotRecover',
            })}
          />
        )}
        <Input.TextArea
          aria-label={intl.formatMessage({
            id: 'geofence.field.reason',
          })}
          value={bindingActionReason}
          rows={3}
          placeholder={intl.formatMessage({
            id: 'geofence.lifecycle.reasonRequired',
          })}
          onChange={(event) =>
            setBindingActionReason(event.target.value)
          }
        />
      </Modal>
    </Drawer>
  );
}
