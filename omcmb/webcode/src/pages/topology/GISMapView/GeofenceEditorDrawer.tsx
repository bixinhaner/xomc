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
  const message = appContext.message?.success
    ? appContext.message
    : staticMessage;
  const [form] = Form.useForm<EditorFormValues>();
  const createMutation = useCreateGeofenceDefinition();
  const createDraftMutation = useCreateGeofenceDraftVersion();
  const publishMutation = usePublishGeofenceDraft();
  const renameMutation = useRenameGeofenceDefinition();
  const existingGeometry =
    item?.currentVersion?.geometry.type === 'Polygon'
      ? item.currentVersion.geometry
      : undefined;
  const geometry = drawnGeometry ?? existingGeometry;

  useEffect(() => {
    if (!open) return;
    const policy = item?.currentVersion?.policy;
    form.setFieldsValue(
      item
        ? {
            name: item.definition.name,
            carrier: item.definition.carrier,
            exitAction:
              policy?.exitAction === 'manual_review'
                ? 'manual_review'
                : 'notify_only',
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
      if (item) {
        if (values.name.trim() !== item.definition.name) {
          await renameMutation.mutateAsync({
            id: item.definition.id,
            name: values.name.trim(),
          });
        }
        const draft = await createDraftMutation.mutateAsync({
          id: item.definition.id,
          input: { geometry, policy },
        });
        await publishMutation.mutateAsync({
          id: item.definition.id,
          versionId: draft.id,
        });
      } else {
        const created = await createMutation.mutateAsync({
          name: values.name.trim(),
          carrier: values.carrier,
          ruleType: 'polygon_allow_zone',
          geometry,
          policy,
        });
        await publishMutation.mutateAsync({
          id: created.definition.id,
          versionId: created.draftVersion.id,
        });
      }
      void message.success(
        intl.formatMessage({ id: 'geofence.message.published' }),
      );
      close();
    } catch (error) {
      if (error instanceof Error) {
        void message.error(error.message);
      }
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
              id: 'geofence.action.saveAndPublish',
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
          ]}
        >
          <Input maxLength={128} />
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
          rules={[{ required: true }]}
        >
          <Select
            options={['notifyOnly', 'manualReview'].map((key) => ({
              value:
                key === 'notifyOnly'
                  ? 'notify_only'
                  : 'manual_review',
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
