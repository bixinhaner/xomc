import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n/index';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type {
  GeofenceMapDefinition,
  GeofencePolygonGeometry,
} from '@core/types/geofence';

const mocks = vi.hoisted(() => ({
  create: vi.fn(),
  createDraft: vi.fn(),
  publish: vi.fn(),
  rename: vi.fn(),
}));

vi.mock('@core/hooks/api/useGeofence', () => ({
  useCreateGeofenceDefinition: () => ({
    mutateAsync: mocks.create,
    isPending: false,
  }),
  useCreateGeofenceDraftVersion: () => ({
    mutateAsync: mocks.createDraft,
    isPending: false,
  }),
  usePublishGeofenceDraft: () => ({
    mutateAsync: mocks.publish,
    isPending: false,
  }),
  useRenameGeofenceDefinition: () => ({
    mutateAsync: mocks.rename,
    isPending: false,
  }),
}));

import GeofenceEditorDrawer from './GeofenceEditorDrawer';

const geometry: GeofencePolygonGeometry = {
  type: 'Polygon',
  coordinates: [[[120, 30], [121, 30], [121, 31], [120, 30]]],
};

const existing: GeofenceMapDefinition = {
  definition: {
    id: 'fence-1',
    name: '园区围栏',
    carrier: 'cmcc',
    ruleType: 'polygon_allow_zone',
    status: 'enabled',
    currentVersionId: 'version-1',
    createdBy: 'operator',
    updatedBy: 'operator',
    createdAt: '2026-07-31T08:00:00Z',
    updatedAt: '2026-07-31T08:00:00Z',
  },
  currentVersion: {
    id: 'version-1',
    geofenceId: 'fence-1',
    version: 1,
    status: 'published',
    geometry,
    boundingBox: {
      minLongitude: 120,
      minLatitude: 30,
      maxLongitude: 121,
      maxLatitude: 31,
    },
    policy: {
      exitAction: 'notify_only',
      exitConsecutiveSamples: 2,
    },
    createdBy: 'operator',
    publishedBy: 'operator',
    createdAt: '2026-07-31T08:00:00Z',
    publishedAt: '2026-07-31T08:00:00Z',
  },
};

function renderEditor({
  item,
  drawnGeometry = geometry,
}: {
  item?: GeofenceMapDefinition;
  drawnGeometry?: GeofencePolygonGeometry;
} = {}) {
  const onClose = vi.fn();
  const onStartDraw = vi.fn();
  const onStopDraw = vi.fn();
  return {
    onClose,
    onStartDraw,
    onStopDraw,
    user: userEvent.setup(),
    ...render(
      <IntlProvider
        locale="zh-CN"
        defaultLocale="zh-CN"
        messages={getMessages('zh-CN')}
      >
        <GeofenceEditorDrawer
          open
          item={item}
          drawnGeometry={drawnGeometry}
          onClose={onClose}
          onStartDraw={onStartDraw}
          onStopDraw={onStopDraw}
        />
      </IntlProvider>,
    ),
  };
}

describe('GeofenceEditorDrawer', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.create.mockResolvedValue({
      definition: existing.definition,
      draftVersion: {
        ...existing.currentVersion,
        id: 'draft-1',
        status: 'draft',
      },
    });
    mocks.createDraft.mockResolvedValue({
      ...existing.currentVersion,
      id: 'draft-2',
      status: 'draft',
    });
    mocks.publish.mockResolvedValue({ status: 'executed' });
    mocks.rename.mockResolvedValue({ status: 'executed' });
  });

  it('creates a polygon draft with WGS84 longitude/latitude order and publishes it', async () => {
    const { user, onClose } = renderEditor();

    await user.type(screen.getByLabelText('围栏名称'), '新围栏');
    await user.click(screen.getByRole('button', { name: '保存并发布' }));

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith({
        name: '新围栏',
        carrier: 'cmcc',
        ruleType: 'polygon_allow_zone',
        geometry,
        policy: {
          exitAction: 'notify_only',
          exitConsecutiveSamples: 2,
          reentryConsecutiveSamples: 2,
        },
      }),
    );
    expect(geometry.coordinates[0][0]).toEqual([120, 30]);
    expect(mocks.publish).toHaveBeenCalledWith({
      id: 'fence-1',
      versionId: 'draft-1',
    });
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('redraws an existing fence as a new immutable version', async () => {
    const { user, onStartDraw } = renderEditor({ item: existing });

    expect(screen.getByLabelText('围栏名称')).toHaveValue('园区围栏');
    await user.click(screen.getByRole('button', { name: '重新绘制区域' }));
    expect(onStartDraw).toHaveBeenCalledTimes(1);
    await user.click(screen.getByRole('button', { name: '保存并发布' }));

    await waitFor(() =>
      expect(mocks.createDraft).toHaveBeenCalledWith({
        id: 'fence-1',
        input: {
          geometry,
          policy: {
            exitAction: 'notify_only',
            exitConsecutiveSamples: 2,
            reentryConsecutiveSamples: 2,
          },
        },
      }),
    );
    expect(mocks.publish).toHaveBeenCalledWith({
      id: 'fence-1',
      versionId: 'draft-2',
    });
    expect(mocks.rename).not.toHaveBeenCalled();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it('cancels drawing without writing server state', async () => {
    const { user, onClose, onStopDraw } = renderEditor();

    await user.click(screen.getByRole('button', { name: /取\s*消/ }));

    expect(onStopDraw).toHaveBeenCalledTimes(1);
    expect(onClose).toHaveBeenCalledTimes(1);
    expect(mocks.create).not.toHaveBeenCalled();
    expect(mocks.createDraft).not.toHaveBeenCalled();
    expect(mocks.publish).not.toHaveBeenCalled();
  });
});
