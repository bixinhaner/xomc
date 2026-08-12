import { useEffect } from 'react';
import {
  App,
  Alert,
  Button,
  Drawer,
  Form,
  Input,
  InputNumber,
  Select,
  Space,
  message as staticMessage,
} from 'antd';
import { useIntl } from 'react-intl';
import {
  useCreateGeofenceDefinition,
  useCreateGeofenceDraftVersion,
  usePublishGeofenceDraft,
  useRenameGeofenceDefinition,
} from '@core/hooks/api/useGeofence';
import type {
  GeofenceExitAction,
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
} from '@core/types/geofence';
import { geofenceErrorMessage } from './geofenceErrorMessage';

interface EditorFormValues {
  name: string;
  carrier: string;
  exitAction: GeofenceExitAction;
  exitConsecutiveSamples: number;
  reentryConsecutiveSamples: number;
}

interface GeofenceEditorDrawerProps {
  open: boolean;
  item?: GeofenceMapDefinition;
  drawnGeometry?: GeofencePolygonGeometry;
  onClose: () => void;
  onStartDraw: () => void;
  onStopDraw: () => void;
}

const DEFAULT_VALUES: EditorFormValues = {
  name: '',
  carrier: 'cmcc',
  exitAction: 'notify_only',
  exitConsecutiveSamples: 2,
  reentryConsecutiveSamples: 2,
};

function samePosition(first: number[], second: number[]): boolean {
  return first[0] === second[0] && first[1] === second[1];
}

function isFormValidationError(
  error: unknown,
): error is { errorFields: unknown[] } {
  return (
    typeof error === 'object' &&
    error !== null &&
    Array.isArray((error as { errorFields?: unknown }).errorFields)
  );
}

export function normalizePolygonGeometry(
  geometry: GeofencePolygonGeometry,
): GeofencePolygonGeometry {
  const ring = geometry.coordinates[0] ?? [];
  const normalized = ring.reduce<number[][]>((points, point) => {
    if (
      points.length === 0 ||
      !samePosition(points[points.length - 1], point)
    ) {
      points.push([...point]);
    }
    return points;
  }, []);
  while (
    normalized.length > 1 &&
    samePosition(normalized[0], normalized[normalized.length - 1])
  ) {
    normalized.pop();
  }
  if (normalized.length > 0) {
    normalized.push([...normalized[0]]);
  }
  return { ...geometry, coordinates: [normalized] };
}

export default function GeofenceEditorDrawer({
  open,
  item,
  drawnGeometry,
  onClose,
  onStartDraw,
  onStopDraw,
}: GeofenceEditorDrawerProps) {
  const intl = useIntl();
  const appContext = App.useApp();
  const message = typeof (appContext.message as { success?: unknown }).success === 'function'
    ? appContext.message
    : staticMessage;
  const [form] = Form.useForm<EditorFormValues>();
  const watchedExitAction = Form.useWatch('exitAction', form);
  const selectedExitAction =
    watchedExitAction ??
    (item
      ? item.currentVersion?.policy.exitAction === 'manual_review'
        ? undefined
        : item.currentVersion?.policy.exitAction ?? 'notify_only'
      : DEFAULT_VALUES.exitAction);
  const createMutation = useCreateGeofenceDefinition();
  const createDraftMutation = useCreateGeofenceDraftVersion();
  const publishMutation = usePublishGeofenceDraft();
  const renameMutation = useRenameGeofenceDefinition();
  const existingGeometry =
    item?.currentVersion?.geometry.type === 'Polygon'
      ? item.currentVersion.geometry
      : undefined;
  const geometry = drawnGeometry ?? existingGeometry;
  const hasLegacyManualReview =
    item?.currentVersion?.policy.exitAction === 'manual_review';

  useEffect(() => {
    if (!open) return;
    const policy = item?.currentVersion?.policy;
    const exitAction = item
      ? policy?.exitAction === 'manual_review'
        ? undefined
        : policy?.exitAction ?? 'notify_only'
      : DEFAULT_VALUES.exitAction;
    form.setFieldsValue(
      item
        ? {
            name: item.definition.name,
            carrier: item.definition.carrier,
            exitAction,
            exitConsecutiveSamples:
              policy?.exitConsecutiveSamples ?? 2,
            reentryConsecutiveSamples:
              policy?.reentryConsecutiveSamples ?? 2,
          }
        : DEFAULT_VALUES,
    );
  }, [form, item, open]);

  const close = () => {
    onStopDraw();
    onClose();
  };

  const save = async () => {
    try {
      const values = await form.validateFields();
      if (!geometry) {
        void message.error(
          intl.formatMessage({
            id: 'geofence.validation.geometryRequired',
          }),
        );
        return;
      }
      const policy = {
        exitAction: values.exitAction,
        exitConsecutiveSamples: values.exitConsecutiveSamples,
        reentryConsecutiveSamples: values.reentryConsecutiveSamples,
      };
      const normalizedGeometry = normalizePolygonGeometry(geometry);
      if (item) {
        const normalizedName = values.name.trim();
        const currentPolicy = item.currentVersion?.policy;
        const nameChanged = normalizedName !== item.definition.name;
        const policyChanged =
          values.exitAction !== (currentPolicy?.exitAction ?? 'notify_only') ||
          values.exitConsecutiveSamples !==
            (currentPolicy?.exitConsecutiveSamples ?? 2) ||
          values.reentryConsecutiveSamples !==
            (currentPolicy?.reentryConsecutiveSamples ?? 2);
        // 历史人工复核策略必须通过新版本显式收敛到受支持的动作，
        // 不能走“围栏信息没有变化”的快速返回分支。
        const versionChanged =
          Boolean(drawnGeometry) || hasLegacyManualReview || policyChanged;
        if (!nameChanged && !versionChanged) {
          void message.info(
            intl.formatMessage({ id: 'geofence.editor.noChanges' }),
          );
          return;
        }
        if (nameChanged) {
          await renameMutation.mutateAsync({
            id: item.definition.id,
            name: normalizedName,
          });
        }
        if (versionChanged) {
          const draft = await createDraftMutation.mutateAsync({
            id: item.definition.id,
            input: { geometry: normalizedGeometry, policy },
          });
          await publishMutation.mutateAsync({
            id: item.definition.id,
            versionId: draft.id,
          });
        }
        void message.success(
          intl.formatMessage({
            id: versionChanged
              ? 'geofence.message.published'
              : 'geofence.message.saved',
          }),
        );
      } else {
        const created = await createMutation.mutateAsync({
          name: values.name.trim(),
          carrier: values.carrier,
          ruleType: 'polygon_allow_zone',
          geometry: normalizedGeometry,
          policy,
        });
        await publishMutation.mutateAsync({
          id: created.definition.id,
          versionId: created.draftVersion.id,
        });
        void message.success(
          intl.formatMessage({ id: 'geofence.message.published' }),
        );
      }
      close();
    } catch (error) {
      // Ant Design 已在对应字段下展示明确的校验文案，不再叠加一个
      // “电子围栏操作失败”的通用 toast，避免把输入错误误报成服务异常。
      if (isFormValidationError(error)) return;
      void message.error(geofenceErrorMessage(intl, error));
    }
  };

  const busy =
    createMutation.isPending ||
    createDraftMutation.isPending ||
    publishMutation.isPending ||
    renameMutation.isPending;

  return (
    <Drawer
      rootClassName="geofence-drawer"
      open={open}
      mask={false}
      size={480}
      title={intl.formatMessage({
        id: item
          ? 'geofence.editor.editTitle'
          : 'geofence.editor.createTitle',
      })}
      onClose={close}
      footer={
        <Space style={{ width: '100%', justifyContent: 'flex-end' }}>
          <Button onClick={close}>
            {intl.formatMessage({ id: 'common.cancel' })}
          </Button>
          <Button
            type="primary"
            loading={busy}
            disabled={!geometry}
            onClick={() => void save()}
          >
            {intl.formatMessage({
              id: item ? 'common.save' : 'geofence.action.saveAndPublish',
            })}
          </Button>
        </Space>
      }
    >
      <Form form={form} layout="vertical" className="geofence-drawer-form">
        {item && (
          <Alert
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
            title={intl.formatMessage({
              id: 'geofence.editor.versionNotice',
            })}
          />
        )}
        {hasLegacyManualReview && (
          <Alert
            type="warning"
            showIcon
            style={{ marginBottom: 16 }}
            title={intl.formatMessage({
              id: 'geofence.policy.legacyManualReviewWarning',
            })}
          />
        )}
        <Form.Item
          name="name"
          label={intl.formatMessage({ id: 'geofence.field.name' })}
          rules={[
            {
              required: true,
              whitespace: true,
              message: intl.formatMessage({
                id: 'geofence.validation.nameRequired',
              }),
            },
            {
              max: 128,
              message: intl.formatMessage({
                id: 'geofence.validation.nameTooLong',
              }),
            },
          ]}
        >
          <Input showCount />
        </Form.Item>
        <Form.Item
          name="carrier"
          label={intl.formatMessage({ id: 'geofence.field.carrier' })}
          rules={[{ required: true }]}
        >
          <Select
            disabled={Boolean(item)}
            options={['cmcc', 'ctcc', 'cucc'].map((carrier) => ({
              value: carrier,
              label: intl.formatMessage({
                id: `geofence.carrier.${carrier}`,
              }),
            }))}
          />
        </Form.Item>
        <Form.Item
          name="exitAction"
          label={intl.formatMessage({
            id: 'geofence.policy.exitAction',
          })}
          extra={
            selectedExitAction === 'notify_only'
              ? intl.formatMessage({
                  id: 'geofence.policy.notifyOnlyDescription',
                })
              : selectedExitAction === 'deactivate'
                ? intl.formatMessage({
                    id: 'geofence.policy.deactivateDescription',
                  })
                : undefined
          }
          rules={[
            {
              required: true,
              message: intl.formatMessage({
                id: 'geofence.validation.exitActionRequired',
              }),
            },
          ]}
        >
          <Select
            placeholder={intl.formatMessage({
              id: 'geofence.validation.exitActionRequired',
            })}
            options={[
              ['notifyOnly', 'notify_only'],
              ['deactivate', 'deactivate'],
            ].map(([key, value]) => ({
              value,
              label: intl.formatMessage({
                id: `geofence.policy.${key}`,
              }),
            }))}
          />
        </Form.Item>
        <Form.Item
          name="exitConsecutiveSamples"
          label={intl.formatMessage({
            id: 'geofence.policy.exitConsecutiveSamples',
          })}
          rules={[{ required: true, type: 'number', min: 1, max: 20 }]}
        >
          <InputNumber min={1} max={20} style={{ width: '100%' }} />
        </Form.Item>
        <Form.Item
          name="reentryConsecutiveSamples"
          label={intl.formatMessage({
            id: 'geofence.policy.reentryConsecutiveSamples',
          })}
          rules={[{ required: true, type: 'number', min: 1, max: 20 }]}
        >
          <InputNumber min={1} max={20} style={{ width: '100%' }} />
        </Form.Item>
        <div className="geofence-geometry-section">
          <div className="geofence-geometry-section-copy">
            <strong>
              {intl.formatMessage({
                id: geometry
                  ? 'geofence.editor.geometryReady'
                  : 'geofence.validation.geometryRequired',
              })}
            </strong>
          </div>
          <Button onClick={onStartDraw}>
            {intl.formatMessage({
              id: item
                ? 'geofence.action.redraw'
                : 'geofence.action.draw',
            })}
          </Button>
        </div>
      </Form>
    </Drawer>
  );
}
