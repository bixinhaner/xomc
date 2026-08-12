import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n/index';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const access = vi.hoisted(() => ({
  available: true,
  permissions: new Set([
    'topology:gis-map:geofence:view',
    'topology:gis-map:geofence:manage',
  ]),
}));

vi.mock('react-router-dom', () => ({
  useNavigate: () => vi.fn(),
}));

vi.mock('@core/store/tabStore', () => ({
  useTabStore: () => vi.fn(),
}));

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorPrimary: '#1677ff',
  }),
}));

vi.mock('@/components/GISMap', () => ({
  default: () => <div data-testid="gis-map" />,
}));

vi.mock('@/components/GISMap/useMapConfig', () => ({
  useMapConfig: () => ({
    status: 'error',
    metadata: undefined,
    isUsingDefault: true,
  }),
}));

vi.mock('./GeofenceBindingsDrawer', () => ({
  default: () => null,
}));

vi.mock('@core/hooks/api/useTopology', () => ({
  useDomainTree: () => ({ data: [], isLoading: false }),
  useDeviceAntennaSectors: () => ({ data: [] }),
  useMapDevicesGeo: () => ({ data: { items: [] } }),
  useMapStats: () => ({
    data: {
      total: 0,
      statusCount: {
        onlineActive: 0,
        onlineInactive: 0,
        offline: 0,
      },
      alarmCount: 0,
    },
  }),
}));

vi.mock('@core/hooks/useDeviceSearch', () => ({
  useDeviceSearch: () => ({
    keyword: '',
    handleChange: vi.fn(),
    handleClear: vi.fn(),
    results: [],
    isLoading: false,
    expanded: false,
    setExpanded: vi.fn(),
  }),
}));

vi.mock('@core/hooks/useAntennaSectorEditor', () => ({
  useAntennaSectorEditor: () => ({
    previewSectors: [],
    updatePreview: vi.fn(),
    discardPreview: vi.fn(),
    saveSector: vi.fn(),
    isSaving: false,
  }),
}));

vi.mock('@core/hooks/usePermission', () => ({
  usePermission: (key: string) => access.permissions.has(key),
}));

vi.mock('@core/hooks/api/useGeofence', () => ({
  useGeofenceAvailability: () => ({ data: { enabled: access.available } }),
  useGeofenceMap: () => ({ data: { items: [] } }),
  useGeofenceSettings: () => ({
    data: {
      systemMode: 'observe',
      carriers: [
        {
          carrier: 'cmcc',
          mode: 'observe',
          effectiveMode: 'observe',
          defaultBaselineRadiusMeters: 1000,
          updatedAt: '2026-07-31T08:00:00Z',
        },
      ],
    },
    isLoading: false,
    isFetching: false,
    isError: false,
    refetch: vi.fn(),
  }),
  usePreviewGeofenceSettings: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
    reset: vi.fn(),
  }),
  useUpdateGeofenceSettings: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePreviewGeofenceLifecycle: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useTransitionGeofenceLifecycle: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePreviewGeofenceCandidates: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useCreateGeofenceDefinition: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useCreateGeofenceDraftVersion: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  usePublishGeofenceDraft: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
  useRenameGeofenceDefinition: () => ({
    mutateAsync: vi.fn(),
    isPending: false,
  }),
}));

vi.mock('@core/services/api/topologyApi', () => ({
  topologyApi: {
    searchDevices: vi.fn(),
  },
}));

import GISMapView from './index';

describe('GISMapView geofence settings integration', () => {
  beforeEach(() => {
    access.available = true;
    access.permissions = new Set([
      'topology:gis-map:geofence:view',
      'topology:gis-map:geofence:manage',
    ]);
  });

  it('hides the GIS entry while the system switch is off', () => {
    access.available = false;
    render(
      <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
        <GISMapView />
      </IntlProvider>,
    );

    expect(
      screen.queryByRole('button', { name: '电子围栏' }),
    ).not.toBeInTheDocument();
  });

  it('hides the GIS entry without geofence view permission', () => {
    access.permissions.clear();
    render(
      <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
        <GISMapView />
      </IntlProvider>,
    );

    expect(
      screen.queryByRole('button', { name: '电子围栏' }),
    ).not.toBeInTheDocument();
  });

  it('opens settings from the enabled geofence tool', async () => {
    const user = userEvent.setup();
    render(
      <IntlProvider
        locale="zh-CN"
        defaultLocale="zh-CN"
        messages={getMessages('zh-CN')}
      >
        <GISMapView />
      </IntlProvider>,
    );

    const geofenceButton = screen.getByRole('button', {
      name: '电子围栏',
    });
    expect(geofenceButton).not.toHaveClass('is-active');
    expect(geofenceButton).toHaveAttribute('aria-pressed', 'false');
    await user.hover(geofenceButton);
    expect(
      screen.queryByRole('tooltip', { name: '电子围栏' }),
    ).not.toBeInTheDocument();
    await user.click(geofenceButton);
    expect(geofenceButton).toHaveClass('is-active');
    expect(geofenceButton).toHaveAttribute('aria-pressed', 'true');
    expect(
      screen.getByRole('region', { name: '电子围栏' }),
    ).toBeInTheDocument();
    await user.click(
      screen.getByRole('button', { name: '新增围栏' }),
    );
    expect(
      screen.getByText('请在地图上绘制保护区域，完成后将打开新增围栏面板'),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '取消绘制' }));
    await user.click(
      screen.getByRole('button', { name: '电子围栏设置' }),
    );

    expect(
      screen.getByRole('dialog'),
    ).toBeInTheDocument();
    expect(screen.queryByText('系统运行模式')).not.toBeInTheDocument();
    expect(screen.getByText('已开启，按运营商开关生效')).toBeInTheDocument();
  });
});
