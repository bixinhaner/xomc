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

function renderEditor(options: {
  item?: GeofenceMapDefinition;
  drawnGeometry?: GeofencePolygonGeometry;
} = {}) {
  const { item } = options;
  const drawnGeometry = Object.prototype.hasOwnProperty.call(
    options,
    'drawnGeometry',
  )
    ? options.drawnGeometry
    : item
      ? undefined
      : geometry;
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

  it('normalizes duplicate drawing-control closure points before creating', async () => {
    const drawnGeometry: GeofencePolygonGeometry = {
      type: 'Polygon',
      coordinates: [[
        [120, 30],
        [121, 30],
        [121, 30],
        [121, 31],
        [120, 30],
        [120, 30],
      ]],
    };
    const { user } = renderEditor({ drawnGeometry });

    await user.type(screen.getByLabelText('围栏名称'), '闭合围栏');
    await user.click(screen.getByRole('button', { name: '保存并发布' }));

    await waitFor(() =>
      expect(mocks.create).toHaveBeenCalledWith(
        expect.objectContaining({ geometry }),
      ),
    );
  });

  it('offers and preserves the deactivate policy', async () => {
    const deactivateFence: GeofenceMapDefinition = {
      ...existing,
      currentVersion: {
        ...existing.currentVersion!,
        policy: {
          ...existing.currentVersion!.policy,
          exitAction: 'deactivate',
        },
      },
    };
    const { user } = renderEditor({ item: deactivateFence });

    expect(screen.getByText('去激活')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() => expect(mocks.createDraft).not.toHaveBeenCalled());
  });

  it('starts an explicit redraw without writing server state', async () => {
    const { user, onStartDraw } = renderEditor({ item: existing });

    expect(screen.getByLabelText('围栏名称')).toHaveValue('园区围栏');
    await user.click(screen.getByRole('button', { name: '重新绘制区域' }));
    expect(onStartDraw).toHaveBeenCalledTimes(1);
    expect(mocks.createDraft).not.toHaveBeenCalled();
    expect(mocks.publish).not.toHaveBeenCalled();
  });

  it('publishes a redrawn existing fence as a new immutable version', async () => {
    const redrawnGeometry: GeofencePolygonGeometry = {
      type: 'Polygon',
      coordinates: [[[122, 30], [123, 30], [123, 31], [122, 30]]],
    };
    const { user } = renderEditor({
      item: existing,
      drawnGeometry: redrawnGeometry,
    });

    await user.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(mocks.createDraft).toHaveBeenCalledWith({
        id: 'fence-1',
        input: {
          geometry: redrawnGeometry,
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

  it('renames without creating or publishing a geometry version', async () => {
    const { user, onClose } = renderEditor({ item: existing });

    expect(
      screen.getByText(
        '仅修改名称不会创建版本；调整保护区域或越界策略时会创建新版本，并在发布成功后切换当前生效版本。',
      ),
    ).toBeInTheDocument();

    const nameInput = screen.getByLabelText('围栏名称');
    await user.clear(nameInput);
    await user.type(nameInput, '园区围栏新名称');
    await user.click(screen.getByRole('button', { name: /保\s*存/ }));

    await waitFor(() =>
      expect(mocks.rename).toHaveBeenCalledWith({
        id: 'fence-1',
        name: '园区围栏新名称',
      }),
    );
    expect(mocks.createDraft).not.toHaveBeenCalled();
    expect(mocks.publish).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it('rejects a 129-character name before calling the API', async () => {
    const { user } = renderEditor();

    await user.type(screen.getByLabelText('围栏名称'), '围'.repeat(129));
    await user.click(screen.getByRole('button', { name: '保存并发布' }));

    expect(
      await screen.findByText('围栏名称不能超过 128 个字符'),
    ).toBeInTheDocument();
    expect(screen.queryByText('电子围栏操作失败')).not.toBeInTheDocument();
    expect(mocks.create).not.toHaveBeenCalled();
  });

  it('shows only the field error for an empty required name', async () => {
    const { user } = renderEditor();

    await user.click(screen.getByRole('button', { name: '保存并发布' }));

    expect(await screen.findByText('请输入围栏名称')).toBeInTheDocument();
    expect(screen.queryByText('电子围栏操作失败')).not.toBeInTheDocument();
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
