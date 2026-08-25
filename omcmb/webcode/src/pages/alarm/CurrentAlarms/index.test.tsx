import { App } from 'antd';
import { render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter, useLocation } from 'react-router-dom';
import CurrentAlarms from './index';

const mocks = vi.hoisted(() => ({
  useAlarmById: vi.fn(),
  mutation: () => ({ isPending: false, mutateAsync: vi.fn() }),
  alarm: {
    id: '892d12c0-ec1a-4fd1-8070-902d3aaf84e9',
    deviceId: 'device-1',
    deviceSn: 'SN-1',
    severity: 'critical',
    alarmIdentifier: 'CELL_DOWN',
    alarmName: 'Cell unavailable',
    description: 'Cell unavailable',
    dealState: '0',
    eventType: 'device',
    eventTime: '2026-08-24T09:00:00Z',
    updTime: '2026-08-24T09:00:00Z',
    unread: '1',
  },
}));

vi.mock('@core/hooks/api/useAlarms', () => ({
  useCurrentAlarms: () => ({ data: { items: [], total: 0 }, isLoading: false, refetch: vi.fn() }),
  useAlarmById: mocks.useAlarmById,
  useAcknowledgeAlarms: mocks.mutation,
  useClearAlarms: mocks.mutation,
  useMarkAlarmRead: mocks.mutation,
  useUnacknowledgeAlarms: mocks.mutation,
  useAlarmCount: () => ({ data: { total_active: 0, critical: 0, major: 0, minor: 0, warning: 0, unacknowledged: 0, unread: 0 } }),
}));
vi.mock('@core/hooks/api/useSystemLicense', () => ({
  useSystemLicense: () => ({ data: undefined, isLoading: false }),
}));

vi.mock('@/hooks/useT', () => ({ useT: () => (key: string) => key }));
vi.mock('@/components/Layout/ListPageLayout', () => ({ default: ({ children }: { children: React.ReactNode }) => <main>{children}</main> }));
vi.mock('@/components/DataTable', () => ({ default: () => <div data-testid="alarm-table" /> }));
vi.mock('@/components/FilterBar', () => ({ default: () => <div data-testid="filter-bar" /> }));
vi.mock('../AlarmDetail', () => ({
  default: ({ alarm, open, onClose }: { alarm: { id: string } | null; open: boolean; onClose: () => void }) => (
    <div data-testid="alarm-detail" data-alarm-id={alarm?.id ?? ''} data-open={String(open)}>
      <button onClick={onClose}>close-detail</button>
    </div>
  ),
}));
vi.mock('../components/AutoRefreshDropdown', () => ({ default: () => <div /> }));
vi.mock('../components/ConfirmWithNoteModal', () => ({ default: () => null }));
vi.mock('./ExportModal', () => ({ default: () => null }));
vi.mock('../hooks/useAlarmListExport', () => ({
  useAlarmListExport: () => ({
    clearSelection: vi.fn(), exportLoading: false, exportOpen: false,
    handleExportConfirm: vi.fn(), handleExportTrigger: vi.fn(), handleSelectAllFiltered: vi.fn(),
    selectAllLoading: false, setExportOpen: vi.fn(),
  }),
}));

function LocationProbe() {
  const location = useLocation();
  return <div data-testid="location-search">{location.search}</div>;
}

describe('CurrentAlarms attention deep link', () => {
  beforeEach(() => {
    mocks.useAlarmById.mockReset().mockReturnValue({ data: mocks.alarm, isFetched: true, isError: false });
  });

  it('loads the alarm by ID, opens existing detail, and removes only alarmId on close', async () => {
    const user = userEvent.setup();
    render(<App><MemoryRouter initialEntries={[
      `/alarm/current?alarmId=${mocks.alarm.id}&severity=critical&from=dashboard`,
    ]}><CurrentAlarms /><LocationProbe /></MemoryRouter></App>);

    await waitFor(() => expect(screen.getByTestId('alarm-detail')).toHaveAttribute('data-open', 'true'));
    expect(screen.getByTestId('alarm-detail')).toHaveAttribute('data-alarm-id', mocks.alarm.id);
    expect(mocks.useAlarmById).toHaveBeenCalledWith(mocks.alarm.id);

    await user.click(screen.getByRole('button', { name: 'close-detail' }));
    await waitFor(() => expect(screen.getByTestId('location-search')).toHaveTextContent('?severity=critical&from=dashboard'));
  });
});
