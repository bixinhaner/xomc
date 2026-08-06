import { useState } from 'react';
import {
  App,
  Alert,
  Button,
  Checkbox,
  Descriptions,
  Input,
  Modal,
  Select,
  Space,
  Spin,
  Switch,
  Tag,
  Tooltip,
  message as staticMessage,
} from 'antd';
import {
  AimOutlined,
  CheckCircleOutlined,
  DeleteOutlined,
  EditOutlined,
  LinkOutlined,
  PauseCircleOutlined,
  PlusOutlined,
  PoweroffOutlined,
  RadarChartOutlined,
  SafetyCertificateOutlined,
  SearchOutlined,
} from '@ant-design/icons';
import { useIntl } from 'react-intl';
import {
  useGeofenceMap,
  useGeofenceSettings,
  usePreviewGeofenceCandidates,
  usePreviewGeofenceSettings,
  usePreviewGeofenceLifecycle,
  useTransitionGeofenceLifecycle,
  useUpdateGeofenceSettings,
} from '@core/hooks/api/useGeofence';
import type {
  GeofenceLifecycleImpact,
  GeofenceLifecycleTarget,
  GeofenceMapDefinition,
  GeofenceSettingsPreview,
  UpdateGeofenceSettingsInput,
} from '@core/types/geofence';
import { geofenceSettingsToInput } from '@core/utils/geofenceSettings';
import { geofenceErrorMessage } from './geofenceErrorMessage';

interface GeofencePanelProps {
  open: boolean;
  canManage: boolean;
  selectedId?: string;
  onLocate: (item: GeofenceMapDefinition) => void;
  onCreate: () => void;
  onEdit: (item: GeofenceMapDefinition) => void;
  onBind: (item: GeofenceMapDefinition, deviceSNs?: string[]) => void;
}

interface PendingTransition {
  item: GeofenceMapDefinition;
  target: GeofenceLifecycleTarget;
  impact: GeofenceLifecycleImpact;
}

interface PendingSettingsToggle {
  enabled: boolean;
  input: UpdateGeofenceSettingsInput;
  preview: GeofenceSettingsPreview;
}

export default function GeofencePanel({
  open,
  canManage,
  selectedId,
  onLocate,
  onCreate,
  onEdit,
  onBind,
}: GeofencePanelProps) {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = typeof (appContext.message as { success?: unknown }).success === 'function'
    ? appContext.message
    : staticMessage;
  const [searchDraft, setSearchDraft] = useState('');
  const [searchName, setSearchName] = useState('');
  const [reason, setReason] = useState('');
  const [pending, setPending] = useState<PendingTransition>();
  const [selectedCarrier, setSelectedCarrier] = useState<string>();
  const settingsQuery = useGeofenceSettings();
  const carrierSettings = settingsQuery.data?.carriers ?? [];
  const effectiveSelectedCarrier =
    selectedCarrier ?? carrierSettings[0]?.carrier;
  const mapQuery = useGeofenceMap(
    {
      name: searchName || undefined,
      carrier: effectiveSelectedCarrier,
    },
    { enabled: open },
  );
  const previewMutation = usePreviewGeofenceLifecycle();
  const transitionMutation = useTransitionGeofenceLifecycle();
  const candidateMutation = usePreviewGeofenceCandidates();
  const settingsPreviewMutation = usePreviewGeofenceSettings();
  const settingsUpdateMutation = useUpdateGeofenceSettings();
  const [pendingSettingsToggle, setPendingSettingsToggle] =
    useState<PendingSettingsToggle>();
  const [candidatePreview, setCandidatePreview] = useState<
    Awaited<ReturnType<typeof candidateMutation.mutateAsync>>
  >();
  const [selectedCandidateSNs, setSelectedCandidateSNs] = useState<string[]>([]);
  const activeCarrier =
    carrierSettings.find(
      (setting) => setting.carrier === effectiveSelectedCarrier,
    ) ??
    carrierSettings[0];
  const carrierEnabled = activeCarrier?.mode !== 'off';
  const items = (mapQuery.data?.items ?? []).filter(
    ({ definition }) =>
      definition.status !== 'archived' &&
      (!activeCarrier || definition.carrier === activeCarrier.carrier),
  );

  if (!open) return null;

  const startTransition = async (
    item: GeofenceMapDefinition,
    target: GeofenceLifecycleTarget,
  ) => {
    try {
      const impact = await previewMutation.mutateAsync({
        id: item.definition.id,
        target,
      });
      setReason('');
      setPending({ item, target, impact });
    } catch (error) {
      void message.error(geofenceErrorMessage(intl, error));
    }
  };

  const confirmTransition = async () => {
    if (!pending || !reason.trim()) return;
    try {
      await transitionMutation.mutateAsync({
        id: pending.item.definition.id,
        target: pending.target,
        input: {
          reason: reason.trim(),
          previewFingerprint: pending.impact.previewFingerprint,
        },
      });
      void message.success(
        intl.formatMessage({
          id: 'geofence.message.transitioned',
        }),
      );
      setPending(undefined);
      setReason('');
    } catch (error) {
      void message.error(geofenceErrorMessage(intl, error));
    }
  };

  const previewCandidates = async (id: string) => {
    try {
      setCandidatePreview(await candidateMutation.mutateAsync(id));
      setSelectedCandidateSNs([]);
    } catch (error) {
      void message.error(geofenceErrorMessage(intl, error));
    }
  };

  const previewMasterToggle = async (enabled: boolean) => {
    if (!settingsQuery.data || !activeCarrier || !canManage) return;
    const input = {
      ...geofenceSettingsToInput(settingsQuery.data),
      carriers: settingsQuery.data.carriers.map((setting) => ({
        carrier: setting.carrier,
        mode:
          setting.carrier === activeCarrier.carrier
            ? enabled
              ? 'enforce'
              : 'off'
            : setting.mode,
        defaultBaselineRadiusMeters:
          setting.defaultBaselineRadiusMeters,
      })),
    } satisfies UpdateGeofenceSettingsInput;
    try {
      const preview = await settingsPreviewMutation.mutateAsync(input);
      setPendingSettingsToggle({ enabled, input, preview });
    } catch (error) {
      void message.error(geofenceErrorMessage(intl, error));
    }
  };

  const confirmMasterToggle = async () => {
    if (!pendingSettingsToggle) return;
    try {
      await settingsUpdateMutation.mutateAsync(pendingSettingsToggle.input);
      setPendingSettingsToggle(undefined);
      void message.success(
        intl.formatMessage({ id: 'geofence.settings.carrierUpdated' }),
      );
    } catch (error) {
      void message.error(geofenceErrorMessage(intl, error));
    }
  };

  return (
    <>
      <section
        aria-label={intl.formatMessage({ id: 'geofence.title' })}
        className="geofence-panel"
      >
        <div className="geofence-panel-header">
          <div className="geofence-panel-heading">
            <span className="geofence-panel-heading-icon" aria-hidden="true">
              <SafetyCertificateOutlined />
            </span>
            <div>
              <strong>{intl.formatMessage({ id: 'geofence.title' })}</strong>
              <span className="geofence-panel-kicker">
                {intl.formatMessage(
                  { id: 'geofence.map.fenceCount' },
                  { count: items.length },
                )}
              </span>
            </div>
          </div>
          <Button
            size="small"
            type="text"
            className="geofence-create-button"
            aria-label={intl.formatMessage({ id: 'geofence.action.create' })}
            icon={<PlusOutlined aria-hidden="true" />}
            onClick={onCreate}
            disabled={!canManage || !carrierEnabled}
          >
            <span className="geofence-create-button-label">
              {intl.formatMessage({ id: 'geofence.action.create' })}
            </span>
          </Button>
        </div>
        <div className="geofence-master-switch">
          <span className="geofence-master-switch-icon" aria-hidden="true">
            <PoweroffOutlined />
          </span>
          <div className="geofence-master-switch-copy">
            <strong>
              {intl.formatMessage({ id: 'geofence.settings.carrierSwitch' })}
            </strong>
            <span>
              {intl.formatMessage({
                id: carrierEnabled
                  ? 'geofence.settings.carrierEnabled'
                  : 'geofence.settings.carrierDisabled',
              })}
            </span>
          </div>
          <Select
            className="geofence-carrier-select"
            aria-label={intl.formatMessage({ id: 'geofence.settings.carrier' })}
            value={effectiveSelectedCarrier}
            options={carrierSettings.map((setting) => ({
              value: setting.carrier,
              label: intl.formatMessage({
                id: `geofence.carrier.${setting.carrier}`,
              }),
            }))}
            onChange={setSelectedCarrier}
            disabled={!canManage || carrierSettings.length === 0}
          />
          <Switch
            className="geofence-master-toggle"
            aria-label={intl.formatMessage({
              id: 'geofence.settings.carrierSwitch',
            })}
            checked={carrierEnabled}
            disabled={
              !canManage ||
              !settingsQuery.data ||
              settingsQuery.isError ||
              !activeCarrier
            }
            loading={
              settingsQuery.isLoading ||
              settingsPreviewMutation.isPending ||
              settingsUpdateMutation.isPending
            }
            onChange={(checked) => void previewMasterToggle(checked)}
          />
        </div>
        <div className="geofence-panel-toolbar">
          <Input
            className="geofence-search-input"
            aria-label={intl.formatMessage({
              id: 'geofence.map.searchPlaceholder',
            })}
            value={searchDraft}
            allowClear
            prefix={<SearchOutlined aria-hidden="true" />}
            placeholder={intl.formatMessage({
              id: 'geofence.map.searchPlaceholder',
            })}
            onChange={(event) => {
              const value = event.target.value;
              setSearchDraft(value);
              if (!value) setSearchName('');
            }}
            onPressEnter={() => setSearchName(searchDraft.trim())}
          />
        </div>

        {mapQuery.isLoading ? (
          <div className="geofence-panel-empty">
            <Spin />
          </div>
        ) : mapQuery.isError ? (
          <Alert
            type="error"
            showIcon
            title={intl.formatMessage({
              id: 'geofence.message.loadFailed',
            })}
            action={
              <Button size="small" onClick={() => void mapQuery.refetch()}>
                {intl.formatMessage({ id: 'common.retry' })}
              </Button>
            }
          />
        ) : (
          items.length === 0 ? (
            <div className="geofence-panel-empty">
              {intl.formatMessage({ id: 'geofence.map.empty' })}
            </div>
          ) : (
            <ul className="geofence-list">
              {items.map((item) => {
              const { definition } = item;
              const canLocate =
                definition.status === 'enabled' &&
                Boolean(item.currentVersion);
              return (
                <li
                  key={definition.id}
                  data-testid={`geofence-row-${definition.id}`}
                  className={selectedId === definition.id ? 'geofence-row is-selected' : 'geofence-row'}
                >
                  <div className="geofence-row-content">
                    <div className="geofence-row-title">
                      <span className="geofence-row-name">
                        <span
                          className={`geofence-row-status-dot is-${definition.status}`}
                          aria-hidden="true"
                        />
                        <span>{definition.name}</span>
                      </span>
                      <Tag
                        variant="filled"
                        color={
                          definition.status === 'enabled'
                            ? 'success'
                            : definition.status === 'draft'
                              ? 'processing'
                              : 'default'
                        }
                      >
                        {intl.formatMessage({
                          id: `geofence.status.${definition.status}`,
                        })}
                      </Tag>
                    </div>
                    <div className="geofence-row-subtitle">
                      <SafetyCertificateOutlined aria-hidden="true" />
                      {intl.formatMessage({
                        id:
                          definition.ruleType === 'polygon_allow_zone'
                            ? 'geofence.ruleType.polygonAllowZone'
                            : 'geofence.ruleType.baselineRadius',
                      })}
                    </div>
                    <div className="geofence-row-actions">
                      <div className="geofence-row-quick-actions">
                        <Tooltip
                          title={intl.formatMessage({
                            id: 'geofence.map.zoomToFence',
                          })}
                        >
                          <Button
                            size="small"
                            type="text"
                            icon={<AimOutlined />}
                            aria-label={intl.formatMessage({
                              id: 'geofence.map.zoomToFence',
                            })}
                            disabled={!canLocate}
                            onClick={() => onLocate(item)}
                          />
                        </Tooltip>
                        <Tooltip
                          title={intl.formatMessage({
                            id: 'geofence.action.edit',
                          })}
                        >
                          <Button
                            size="small"
                            type="text"
                            icon={<EditOutlined />}
                            aria-label={intl.formatMessage({
                              id: 'geofence.action.edit',
                            })}
                            disabled={!canManage}
                            onClick={() => onEdit(item)}
                          />
                        </Tooltip>
                        <Tooltip
                          title={intl.formatMessage({
                            id: 'geofence.action.previewCandidates',
                          })}
                        >
                          <Button
                            size="small"
                            type="text"
                            icon={<RadarChartOutlined />}
                            aria-label={intl.formatMessage({
                              id: 'geofence.action.previewCandidates',
                            })}
                            loading={candidateMutation.isPending}
                            onClick={() => void previewCandidates(definition.id)}
                          />
                        </Tooltip>
                      </div>
                      <div className="geofence-row-primary-actions">
                        {definition.status === 'enabled' && (
                          <Button
                            size="small"
                            type="text"
                            icon={<LinkOutlined aria-hidden="true" />}
                            disabled={!canManage || !carrierEnabled}
                            onClick={() => onBind(item)}
                          >
                            {intl.formatMessage({
                              id: 'geofence.action.bindDevices',
                            })}
                          </Button>
                        )}
                        {definition.status === 'enabled' && (
                          <Button
                            size="small"
                            type="text"
                            danger
                            icon={<PauseCircleOutlined aria-hidden="true" />}
                            disabled={!canManage}
                            onClick={() =>
                              void startTransition(item, 'disabled')
                            }
                          >
                            {intl.formatMessage({
                              id: 'geofence.action.disable',
                            })}
                          </Button>
                        )}
                        {definition.status === 'disabled' && (
                          <>
                            <Button
                              size="small"
                              type="text"
                              className="geofence-row-enable-action"
                              icon={<CheckCircleOutlined aria-hidden="true" />}
                              disabled={!canManage || !carrierEnabled}
                              onClick={() =>
                                void startTransition(item, 'enabled')
                              }
                            >
                              {intl.formatMessage({
                                id: 'geofence.action.enable',
                              })}
                            </Button>
                            <Button
                              size="small"
                              type="text"
                              danger
                              icon={<DeleteOutlined aria-hidden="true" />}
                              disabled={!canManage}
                              onClick={() =>
                                void startTransition(item, 'archived')
                              }
                            >
                              {intl.formatMessage({
                                id: 'geofence.action.archive',
                              })}
                            </Button>
                          </>
                        )}
                      </div>
                    </div>
                  </div>
                </li>
              );
              })}
            </ul>
          )
        )}
      </section>

      <Modal
        className="geofence-modal"
        open={Boolean(candidatePreview)}
        title={intl.formatMessage({ id: 'geofence.candidate.previewTitle' })}
        footer={null}
        onCancel={() => setCandidatePreview(undefined)}
      >
        {candidatePreview && (
          <>
          <div className="geofence-candidate-summary">
            {[
              ['geofence.candidate.inside', candidatePreview.inside.length],
              ['geofence.candidate.outside', candidatePreview.outside.length],
              ['geofence.candidate.noLocation', candidatePreview.noLocation.length],
            ].map(([id, count]) => (
              <div className="geofence-candidate-summary-item" key={id}>
                <span>{intl.formatMessage({ id: String(id) })}</span>
                <strong>{count}</strong>
              </div>
            ))}
          </div>
          <div className="geofence-candidate-list">
            {candidatePreview.inside.map((device) => (
              <div key={device.id} className="geofence-candidate-item">
                <Checkbox
                  checked={selectedCandidateSNs.includes(device.serialNumber)}
                  onChange={(event) => {
                    setSelectedCandidateSNs((current) => event.target.checked
                      ? [...current, device.serialNumber]
                      : current.filter((sn) => sn !== device.serialNumber));
                  }}
                >
                  {device.serialNumber}{device.name ? ` · ${device.name}` : ''}
                </Checkbox>
              </div>
            ))}
          </div>
          <Button
            type="primary"
            disabled={!canManage || selectedCandidateSNs.length === 0}
            className="geofence-candidate-bind-button"
            onClick={() => {
              const candidateItem = items.find(
                (entry) => entry.definition.id === candidatePreview.geofenceId,
              );
              if (candidateItem) onBind(candidateItem, selectedCandidateSNs);
              setCandidatePreview(undefined);
            }}
          >
            {intl.formatMessage({ id: 'geofence.candidate.bindSelected' })}
          </Button>
          </>
        )}
      </Modal>

      <Modal
        className="geofence-modal"
        open={Boolean(pendingSettingsToggle)}
        title={intl.formatMessage({
          id: 'geofence.settings.carrierConfirmTitle',
        })}
        okText={intl.formatMessage({ id: 'common.confirm' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        confirmLoading={settingsUpdateMutation.isPending}
        onCancel={() => setPendingSettingsToggle(undefined)}
        onOk={() => void confirmMasterToggle()}
      >
        {pendingSettingsToggle && (
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <Alert
              type={pendingSettingsToggle.enabled ? 'info' : 'warning'}
              showIcon
              title={intl.formatMessage({
                id: pendingSettingsToggle.enabled
                  ? 'geofence.settings.carrierEnableNotice'
                  : 'geofence.settings.carrierDisableNotice',
              })}
            />
            <Descriptions
              column={1}
              size="small"
              items={[
                {
                  key: 'enabled',
                  label: intl.formatMessage({
                    id: 'geofence.settings.enabledGeofences',
                  }),
                  children: pendingSettingsToggle.preview.enabledGeofences,
                },
                {
                  key: 'bindings',
                  label: intl.formatMessage({
                    id: 'geofence.settings.activeBindings',
                  }),
                  children: pendingSettingsToggle.preview.activeBindings,
                },
                {
                  key: 'observed',
                  label: intl.formatMessage({
                    id: 'geofence.settings.newlyObservedDevices',
                  }),
                  children: pendingSettingsToggle.preview.newlyObservedDevices,
                },
              ]}
            />
          </Space>
        )}
      </Modal>

      <Modal
        className="geofence-modal"
        open={Boolean(pending)}
        title={intl.formatMessage({
          id: 'geofence.lifecycle.previewTitle',
        })}
        okText={intl.formatMessage({ id: 'common.confirm' })}
        cancelText={intl.formatMessage({ id: 'common.cancel' })}
        okButtonProps={{
          disabled:
            !reason.trim() ||
            (pending?.target === 'archived' &&
              pending.impact.activeBatchJobCount > 0),
        }}
        confirmLoading={transitionMutation.isPending}
        onCancel={() => setPending(undefined)}
        onOk={() => void confirmTransition()}
      >
        {pending && (
          <Space orientation="vertical" size="middle" style={{ width: '100%' }}>
            <Descriptions column={1} size="small">
              <Descriptions.Item
                label={intl.formatMessage({
                  id: 'geofence.lifecycle.bindingCount',
                })}
              >
                {pending.impact.bindingCount}
              </Descriptions.Item>
              <Descriptions.Item
                label={intl.formatMessage({
                  id: 'geofence.lifecycle.deviceCount',
                })}
              >
                {pending.impact.deviceCount}
              </Descriptions.Item>
              <Descriptions.Item
                label={intl.formatMessage({
                  id: 'geofence.lifecycle.activeJobs',
                })}
              >
                {pending.impact.activeBatchJobCount}
              </Descriptions.Item>
            </Descriptions>
            {pending.target === 'disabled' && (
              <Alert
                type="warning"
                showIcon
                title={intl.formatMessage({
                  id: 'geofence.message.disableDoesNotRecover',
                })}
              />
            )}
            {pending.target === 'archived' && (
              <Alert
                type="warning"
                showIcon
                title={intl.formatMessage({
                  id: 'geofence.message.archiveRemovesBindings',
                })}
              />
            )}
            {pending.target === 'archived' &&
              pending.impact.activeBatchJobCount > 0 && (
                <Alert
                  type="error"
                  showIcon
                  title={intl.formatMessage({
                    id: 'geofence.lifecycle.activeJobsBlockArchive',
                  })}
                />
              )}
            <Input.TextArea
              aria-label={intl.formatMessage({
                id: 'geofence.field.reason',
              })}
              value={reason}
              rows={3}
              placeholder={intl.formatMessage({
                id: 'geofence.lifecycle.reasonRequired',
              })}
              onChange={(event) => setReason(event.target.value)}
            />
          </Space>
        )}
      </Modal>
    </>
  );
}
