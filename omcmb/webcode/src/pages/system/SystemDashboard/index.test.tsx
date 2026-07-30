import { cleanup, render, screen } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import SystemDashboard from './index';

const systemInfoMocks = vi.hoisted(() => ({
  useSystemInfo: vi.fn(),
}));

vi.mock('@core/hooks/api/useSystem', () => ({
  useSystemInfo: systemInfoMocks.useSystemInfo,
}));

vi.mock('@/components/Charts/GaugeChart', () => ({
  default: ({ value }: { value: number }) => <div data-testid="gauge-chart">{value}%</div>,
}));

vi.mock('@/hooks/useT', () => ({
  useT: () => (key: string) => key,
}));

describe('SystemDashboard disk usage', () => {
  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('does not fall back to the legacy disk value when typed metrics are unavailable', () => {
    systemInfoMocks.useSystemInfo.mockReturnValue({ data: {
      disks: [{ id: 'root', usedPercent: 58, status: 'available' }],
      storage: [{
        id: 'host-filesystem', kind: 'host_filesystem', label: 'Host filesystems',
        source: 'prometheus/node_exporter', status: 'unavailable', error: 'metrics missing',
      }],
    } });

    render(<SystemDashboard />);

    expect(screen.queryByText('58%')).not.toBeInTheDocument();
    expect(screen.getByText('metrics missing')).toBeInTheDocument();
    expect(screen.getAllByTestId('gauge-chart')).toHaveLength(2);
  });

  it('renders each source-specific metric without inventing a database percentage', () => {
    systemInfoMocks.useSystemInfo.mockReturnValue({
      data: {
        storage: [
          {
            id: 'root', kind: 'app_filesystem', label: '/', mountPath: '/', source: 'statfs/app',
            totalBytes: 409600, usedBytes: 307200, availableBytes: 81920, usedPercent: 75,
            collectedAt: '2026-07-20T12:00:00Z', status: 'available',
          },
          {
            id: 'minio-cluster', kind: 'minio_cluster', label: 'MinIO cluster', source: 'prometheus/minio',
            totalBytes: 2048, usedBytes: 1024, availableBytes: 1024, usedPercent: 50,
            collectedAt: '2026-07-20T12:00:00Z', status: 'available',
          },
          {
            id: 'database-postgres', kind: 'database_logical', label: 'postgres:5432',
            source: 'prometheus/otelcol', usedBytes: 1024, collectedAt: '2026-07-20T12:00:00Z',
            status: 'available',
          },
        ],
      },
    });

    render(<SystemDashboard />);

    expect(screen.getByText('system.dashboard.storage.kind.app_filesystem')).toBeInTheDocument();
    expect(screen.getByText('system.dashboard.storage.kind.minio_cluster')).toBeInTheDocument();
    expect(screen.getByText('system.dashboard.storage.kind.database_logical')).toBeInTheDocument();
    expect(screen.getByText('common.source: statfs/app')).toBeInTheDocument();
    expect(screen.getByText('common.source: prometheus/minio')).toBeInTheDocument();
    expect(screen.getByText('common.source: prometheus/otelcol')).toBeInTheDocument();
    expect(screen.getAllByTestId('gauge-chart')).toHaveLength(4);
    expect(screen.getByText('1 KiB')).toBeInTheDocument();
    expect(screen.queryByText('NaN%')).not.toBeInTheDocument();
  });

  it('renders stale source data as unavailable instead of showing a stale gauge', () => {
    systemInfoMocks.useSystemInfo.mockReturnValue({ data: {
      storage: [{
        id: 'host-filesystem', kind: 'host_filesystem', label: 'Host filesystems',
        source: 'prometheus/node_exporter', status: 'stale', error: 'sample is stale',
      }],
    } });

    render(<SystemDashboard />);

    expect(screen.getByText('system.dashboard.storage.stale')).toBeInTheDocument();
    expect(screen.getByText('sample is stale')).toBeInTheDocument();
    expect(screen.queryByText('NaN%')).not.toBeInTheDocument();
  });
});
