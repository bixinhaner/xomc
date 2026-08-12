import { useCallback, useEffect, useRef, useState } from 'react';
import {
  App,
  Alert,
  Button,
  Descriptions,
  Divider,
  Drawer,
  Form,
  Input,
  Modal,
  Segmented,
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
  useGeofenceDefinitions,
  useGeofenceManualBindItems,
  useGeofenceManualBindJob,
  usePreviewGeofenceManualBindings,
  useRemoveGeofenceBinding,
  useResumeGeofenceBinding,
  useSuspendGeofenceBinding,
} from '@core/hooks/api/useGeofence';
import type {
  GeofenceBindingDetail,
  GeofenceControlAction,
  GeofenceJobStatus,
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
  kind: 'suspend' | 'resume' | 'remove';
  binding: GeofenceBindingDetail;
}

interface ManualBindJobTracking {
  jobId: string;
  targetGeofenceId: string;
  affectedGeofenceIds: string[];
  status?: GeofenceJobStatus;
}

interface ManualBindJobTrackerProps {
  tracking: ManualBindJobTracking;
  visible: boolean;
  onStatusChange: (
    targetGeofenceId: string,
    jobId: string,
    status: GeofenceJobStatus,
  ) => void;
}

function ManualBindJobTracker({
  tracking,
  visible,
  onStatusChange,
}: ManualBindJobTrackerProps) {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = typeof (appContext.message as { success?: unknown }).success === 'function'
    ? appContext.message
    : staticMessage;
  const notifiedTerminalJobRef = useRef<string | undefined>(undefined);
  const jobQuery = useGeofenceManualBindJob(
    tracking.jobId,
    tracking.affectedGeofenceIds,
  );
  const itemsQuery = useGeofenceManualBindItems(
    tracking.jobId,
    { page: 1, pageSize: 50 },
    { enabled: visible },
  );
  const job = jobQuery.data;

  useEffect(() => {
    if (!job) return;
    onStatusChange(
      tracking.targetGeofenceId,
      tracking.jobId,
      job.status,
    );
    if (
      !['succeeded', 'failed', 'canceled'].includes(job.status) ||
      notifiedTerminalJobRef.current === `${job.id}:${job.status}`
    ) {
      return;
    }
    notifiedTerminalJobRef.current = `${job.id}:${job.status}`;
    if (job.status === 'succeeded') {
      void message.success(
        intl.formatMessage(
          { id: 'geofence.message.bindingJobCompleted' },
          {
            succeeded: job.progress.succeeded,
            skipped: job.progress.skipped,
            failed: job.progress.failed,
          },
        ),
      );
    } else if (job.status === 'failed') {
      void message.error(
        intl.formatMessage({ id: 'geofence.message.bindingJobFailed' }),
      );
    } else {
      void message.warning(
        intl.formatMessage({ id: 'geofence.message.bindingJobCanceled' }),
      );
    }
  }, [
    intl,
    job,
    message,
    onStatusChange,
    tracking.jobId,
    tracking.targetGeofenceId,
  ]);

  if (!visible) return null;

  const completed = job
    ? job.progress.total - job.progress.pending
    : 0;
  return (
    <>
      {jobQuery.isError && (
        <Alert
          type="warning"
          showIcon
          style={{ marginTop: 20 }}
          title={intl.formatMessage({
            id: 'geofence.message.bindingJobStatusRetrying',
          })}
        />
      )}
      {job && (
        <Descriptions
          column={1}
          size="small"
          title={intl.formatMessage({ id: 'geofence.job.title' })}
          style={{ marginTop: 20 }}
        >
          <Descriptions.Item
            label={intl.formatMessage({
              id: `geofence.job.status.${job.status}`,
            })}
          >
            {intl.formatMessage(
              { id: 'geofence.job.progress' },
              {
                completed,
                total: job.progress.total,
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
    </>
  );
}

const PREVIEW_REASON_MESSAGE_IDS: Record<string, string> = {
  device_unavailable: 'geofence.binding.reason.deviceUnavailable',
  geofence_not_enabled: 'geofence.binding.reason.geofenceNotEnabled',
  carrier_mismatch: 'geofence.binding.reason.carrierMismatch',
  baseline_owner_mismatch: 'geofence.binding.reason.baselineOwnerMismatch',
  already_bound: 'geofence.binding.reason.alreadyBound',
  binding_suspended: 'geofence.binding.reason.bindingSuspended',
  active_rule_conflict: 'geofence.binding.reason.activeRuleConflict',
  binding_changed: 'geofence.binding.reason.bindingChanged',
  reassigned: 'geofence.binding.reason.reassigned',
  geofence_version_changed: 'geofence.binding.reason.geofenceVersionChanged',
};

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

interface ControlParameterRow {
  path: string;
  before?: string;
  requested?: string;
  verified?: string;
}

function controlParameterRows(action: GeofenceControlAction): ControlParameterRow[] {
  const rows = new Map<string, ControlParameterRow>();
  const merge = (field: 'before' | 'requested' | 'verified', path: string, value: string) => {
    rows.set(path, { ...rows.get(path), path, [field]: value });
  };
  action.beforeState.forEach(({ path, value }) => merge('before', path, value));
  action.requestedState.forEach(({ path, value }) => merge('requested', path, value));
  action.verifiedState.forEach(({ path, value }) => merge('verified', path, value));
  return Array.from(rows.values()).filter((row) => row.requested !== undefined || row.verified !== undefined);
}

function controlParameterName(path: string, format: (id: string, values?: Record<string, string>) => string) {
  const rf = path.match(/FAPService\.(\d+)\..*RFTxStatus$/i);
  if (rf) return format('geofence.control.rfInstance', { instance: rf[1] });
  const ipsec = path.match(/Ipsec\.(\d+)\..*(?:TUNNEL_ENABLE|TUNNELENABLE)$/i);
  if (ipsec) return format('geofence.control.ipsecInstance', { instance: ipsec[1] });
  return path.split('.').slice(-2).join('.');
}

export default function GeofenceBindingsDrawer({
  open,
  item,
  initialDeviceSNs,
  onClose,
}: GeofenceBindingsDrawerProps) {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = typeof (appContext.message as { success?: unknown }).success === 'function'
    ? appContext.message
    : staticMessage;
  const [inputText, setInputText] = useState('');
  const [reason, setReason] = useState('');
  const [preview, setPreview] =
    useState<GeofenceManualBindingPreview>();
  const [jobTrackings, setJobTrackings] = useState<
    ManualBindJobTracking[]
  >([]);
  const [bindingView, setBindingView] = useState<'current' | 'history'>(
    'current',
  );
  const [bindingAction, setBindingAction] =
    useState<BindingAction>();
  const [bindingActionReason, setBindingActionReason] =
    useState('');
  const geofenceId = item?.definition.id;
  const currentBindingsQuery = useGeofenceBindings(
    geofenceId,
    { status: 'current', page: 1, pageSize: 50 },
    { enabled: open },
  );
  const historyBindingsQuery = useGeofenceBindings(
    geofenceId,
    { status: 'removed', page: 1, pageSize: 50 },
    { enabled: open },
  );
  const definitionsQuery = useGeofenceDefinitions({}, { enabled: open });
  const actionsQuery = useGeofenceControlActions(
    geofenceId,
    { enabled: open },
  );
  const previewMutation = usePreviewGeofenceManualBindings();
  const createJobMutation = useCreateGeofenceManualBindJob();
  const exportMutation = useExportGeofenceBindings();
  const currentJobTracking = jobTrackings.find(
    (tracking) => tracking.targetGeofenceId === geofenceId,
  );
  const suspendMutation = useSuspendGeofenceBinding();
  const resumeMutation = useResumeGeofenceBinding();
  const removeMutation = useRemoveGeofenceBinding();
  const deviceSNs = parseDeviceSNs(inputText);
  const executableCount = preview
    ? preview.eligibleCount + preview.moveCount
    : 0;
  const jobInProgress = Boolean(
    currentJobTracking &&
      (!currentJobTracking.status ||
        ['pending', 'running', 'zombie'].includes(
          currentJobTracking.status,
        )),
  );
  const handleJobStatusChange = useCallback(
    (
      targetGeofenceId: string,
      jobId: string,
      status: GeofenceJobStatus,
    ) => {
      setJobTrackings((current) => {
        const trackedJob = current.find(
          (tracking) =>
            tracking.targetGeofenceId === targetGeofenceId &&
            tracking.jobId === jobId,
        );
        if (!trackedJob || trackedJob.status === status) {
          return current;
        }
        return current.map((tracking) =>
          tracking === trackedJob ? { ...tracking, status } : tracking,
        );
      });
    },
    [],
  );
  const currentBindings = (currentBindingsQuery.data?.items ?? []).filter(
    (binding) =>
      binding.status === 'active' || binding.status === 'suspended',
  );
  const historyBindings = (historyBindingsQuery.data?.items ?? []).filter(
    (binding) => binding.status === 'removed',
  );
  const visibleBindings =
    bindingView === 'current' ? currentBindings : historyBindings;
  const currentBindingCount = currentBindingsQuery.data?.total ?? 0;
  const historyBindingCount = historyBindingsQuery.data?.total ?? 0;
  const selectedBindingsQuery =
    bindingView === 'current' ? currentBindingsQuery : historyBindingsQuery;
  const geofenceNameByID = new Map(
    (definitionsQuery.data ?? []).map((geofence) => [
      geofence.id,
      geofence.name,
    ]),
  );

  useEffect(() => {
    if (!open) {
      setInputText('');
      setReason('');
      setPreview(undefined);
      setBindingView('current');
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
      } else if (bindingAction.kind === 'resume') {
        await resumeMutation.mutateAsync(input);
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
      !reason.trim() ||
      jobInProgress
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
      const affectedGeofenceIds = Array.from(
        new Set([
          geofenceId,
          ...preview.items.flatMap((previewItem) =>
            previewItem.sourceGeofenceId
              ? [previewItem.sourceGeofenceId]
              : [],
          ),
        ]),
      );
      setJobTrackings((current) => [
        ...current.filter(
          (tracking) => tracking.targetGeofenceId !== geofenceId,
        ),
        {
          jobId: accepted.jobId,
          targetGeofenceId: geofenceId,
          affectedGeofenceIds,
        },
      ]);
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
        filter: { status: 'current' },
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
        <Space size="small">
          <Segmented
            size="small"
            value={bindingView}
            options={[
              {
                value: 'current',
                label: `${intl.formatMessage({ id: 'geofence.binding.current' })} ${currentBindingCount}`,
              },
              {
                value: 'history',
                label: `${intl.formatMessage({ id: 'geofence.binding.history' })} ${historyBindingCount}`,
              },
            ]}
            onChange={(value) =>
              setBindingView(value as 'current' | 'history')
            }
          />
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
        </Space>
      </div>
      {selectedBindingsQuery.isLoading ? (
        <Spin style={{ display: 'block', margin: 24 }} />
      ) : selectedBindingsQuery.isError ? (
        <Alert
          type="error"
          showIcon
          title={intl.formatMessage({
            id: 'geofence.message.loadFailed',
          })}
          action={
            <Button
              size="small"
              onClick={() => void selectedBindingsQuery.refetch()}
            >
              {intl.formatMessage({ id: 'common.retry' })}
            </Button>
          }
        />
      ) : (
        <div className="geofence-binding-list">
          {visibleBindings.map((binding) => (
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
                        onClick={() =>
                          beginBindingAction(binding, 'resume')
                        }
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
          <span>{intl.formatMessage({ id: 'geofence.control.title' })}</span>
          <span className="geofence-control-results-count">
            {(actionsQuery.data ?? []).length}
          </span>
        </summary>
        {actionsQuery.isLoading ? (
          <div className="geofence-control-results-state">
            <Spin size="small" />
          </div>
        ) : actionsQuery.isError ? (
          <div className="geofence-control-results-state">
            <Typography.Text type="danger">
              {intl.formatMessage({ id: 'geofence.message.loadFailed' })}
            </Typography.Text>
          </div>
        ) : (actionsQuery.data ?? []).length === 0 ? (
          <div className="geofence-control-results-state">
            {intl.formatMessage({ id: 'geofence.control.empty' })}
          </div>
        ) : (
          (actionsQuery.data ?? []).map((action) => (
            <article
              className="geofence-control-action-card"
              key={action.id}
            >
              <header className="geofence-control-action-header">
                <div className="geofence-control-action-identity">
                  <strong>{action.deviceSN}</strong>
                  <Typography.Text type="secondary">
                    {intl.formatDate(
                      action.completedAt ?? action.createdAt,
                      {
                        year: 'numeric',
                        month: 'numeric',
                        day: 'numeric',
                        hour: '2-digit',
                        minute: '2-digit',
                        second: '2-digit',
                        hour12: false,
                      },
                    )}
                  </Typography.Text>
                </div>
                <Space size={6} wrap>
                  <Tag>
                    {action.actionType === 'activate'
                      ? intl.formatMessage({
                          id: 'geofence.control.activate',
                        })
                      : intl.formatMessage({
                          id: 'geofence.control.deactivate',
                        })}
                  </Tag>
                  <Tag
                    color={
                      action.status === 'verified'
                        ? 'success'
                        : action.status === 'failed' ||
                            action.status === 'partial_failed'
                          ? 'error'
                          : 'processing'
                    }
                  >
                    {intl.formatMessage({
                      id: `geofence.control.status.${action.status}`,
                    })}
                  </Tag>
                </Space>
              </header>
              <div
                className="geofence-control-parameter-table"
                role="table"
              >
                <div
                  className="geofence-control-parameter-row geofence-control-parameter-head"
                  role="row"
                >
                  <span>
                    {intl.formatMessage({
                      id: 'geofence.control.parameter',
                    })}
                  </span>
                  <span>
                    {intl.formatMessage({ id: 'geofence.control.before' })}
                  </span>
                  <span>
                    {intl.formatMessage({
                      id: 'geofence.control.requested',
                    })}
                  </span>
                  <span>
                    {intl.formatMessage({
                      id: 'geofence.control.verified',
                    })}
                  </span>
                  <span>
                    {intl.formatMessage({ id: 'geofence.control.result' })}
                  </span>
                </div>
                {controlParameterRows(action).map((parameter) => {
                  const matched =
                    parameter.requested !== undefined &&
                    parameter.verified === parameter.requested;
                  const missing =
                    parameter.requested !== undefined &&
                    parameter.verified === undefined;
                  return (
                    <div
                      className="geofence-control-parameter-row"
                      role="row"
                      key={parameter.path}
                    >
                      <span className="geofence-control-parameter-name">
                        <strong>
                          {controlParameterName(
                            parameter.path,
                            (id, values) =>
                              intl.formatMessage({ id }, values),
                          )}
                        </strong>
                        <Tooltip title={parameter.path} placement="topLeft">
                          <code>{parameter.path}</code>
                        </Tooltip>
                      </span>
                      <code>{parameter.before ?? '—'}</code>
                      <code>{parameter.requested ?? '—'}</code>
                      <code>{parameter.verified ?? '—'}</code>
                      <Tag
                        color={
                          matched ? 'success' : missing ? 'warning' : 'error'
                        }
                      >
                        {intl.formatMessage({
                          id: matched
                            ? 'geofence.control.match'
                            : missing
                              ? 'geofence.control.missing'
                              : 'geofence.control.mismatch',
                        })}
                      </Tag>
                    </div>
                  );
                })}
              </div>
              {action.lastError && (
                <Alert
                  className="geofence-control-action-error"
                  type="error"
                  showIcon
                  title={action.lastError}
                />
              )}
            </article>
          ))
        )}
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
          <div
            className="geofence-binding-preview-table"
            role="table"
            aria-label={intl.formatMessage({
              id: 'geofence.binding.previewTitle',
            })}
          >
            <div className="geofence-binding-preview-row geofence-binding-preview-head" role="row">
              <span role="columnheader">
                {intl.formatMessage({ id: 'geofence.binding.previewDevice' })}
              </span>
              <span role="columnheader">
                {intl.formatMessage({ id: 'geofence.binding.previewDecision' })}
              </span>
              <span role="columnheader">
                {intl.formatMessage({ id: 'geofence.binding.previewDetail' })}
              </span>
            </div>
            {preview.items.map((previewItem) => {
              const sourceName = previewItem.sourceGeofenceId
                ? geofenceNameByID.get(previewItem.sourceGeofenceId)
                : undefined;
              const reasonMessageID = previewItem.reasonCode
                ? PREVIEW_REASON_MESSAGE_IDS[previewItem.reasonCode]
                : undefined;
              return (
                <div
                  className="geofence-binding-preview-row"
                  role="row"
                  key={previewItem.inputKey}
                >
                  <Typography.Text ellipsis role="cell">
                    {previewItem.deviceSN ?? previewItem.input}
                  </Typography.Text>
                  <span role="cell">
                    <Tag
                      color={
                        previewItem.decision === 'eligible'
                          ? 'green'
                          : previewItem.decision === 'move'
                            ? 'blue'
                            : 'default'
                      }
                    >
                      {intl.formatMessage({
                        id: `geofence.binding.decision.${previewItem.decision}`,
                      })}
                    </Tag>
                  </span>
                  <Typography.Text type="secondary" role="cell">
                    {sourceName
                      ? intl.formatMessage(
                          { id: 'geofence.binding.sourceFence' },
                          { name: sourceName },
                        )
                      : reasonMessageID
                        ? intl.formatMessage({ id: reasonMessageID })
                        : intl.formatMessage({
                            id: 'geofence.binding.reason.ready',
                          })}
                  </Typography.Text>
                </div>
              );
            })}
          </div>
          <Form.Item
            required
            label={intl.formatMessage({ id: 'geofence.field.reason' })}
            validateStatus={!reason.trim() ? 'error' : undefined}
            help={
              !reason.trim()
                ? intl.formatMessage({
                    id: 'geofence.validation.reasonRequired',
                  })
                : undefined
            }
            style={{ marginTop: 16, marginBottom: 12 }}
          >
            <Input.TextArea
              aria-label={intl.formatMessage({
                id: 'geofence.field.reason',
              })}
              value={reason}
              rows={2}
              placeholder={intl.formatMessage({
                id: 'geofence.lifecycle.reasonRequired',
              })}
              onChange={(event) => setReason(event.target.value)}
            />
          </Form.Item>
          <Tooltip
            title={
              executableCount > 0 && !reason.trim()
                ? intl.formatMessage({
                    id: 'geofence.binding.reasonBeforeCreate',
                  })
                : undefined
            }
          >
            <span>
              <Button
                type="primary"
                disabled={
                  executableCount === 0 || !reason.trim() || jobInProgress
                }
                loading={createJobMutation.isPending}
                onClick={() => void createJob()}
              >
                {intl.formatMessage({
                  id: 'geofence.binding.createJob',
                })}
              </Button>
            </span>
          </Tooltip>
        </div>
      )}

      {jobTrackings.map((tracking) => (
        <ManualBindJobTracker
          key={tracking.jobId}
          tracking={tracking}
          visible={
            open && tracking.targetGeofenceId === geofenceId
          }
          onStatusChange={handleJobStatusChange}
        />
      ))}
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
          suspendMutation.isPending ||
          resumeMutation.isPending ||
          removeMutation.isPending
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
