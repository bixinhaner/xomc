import { fireEvent, render, screen } from '@testing-library/react';
import { ConfigProvider, theme } from 'antd';
import { IntlProvider } from 'react-intl';
import { enUS, zhCN } from '@core/i18n';
import GpsSyncConfirmModal from './GpsSyncConfirmModal';

const device = {
  longitude: 115.366462,
  latitude: 25.92416,
  gpsHeight: 174,
  locationSync: {
    status: 'pending' as const,
    accepted: { longitude: 115.366462, latitude: 25.92416 },
    reported: {
      longitude: 115.376,
      latitude: 25.9341,
      observedAt: '2026-07-17T10:00:00Z',
      version: 3,
      sourcePath: 'Device.FAP.GPS',
    },
    distanceMeters: 1500,
    heightDiffMeters: null,
  },
};

function renderDialog(
  locale: 'zh-CN' | 'en-US',
  onConfirm = () => {},
  targetDevice = device,
) {
  const messages = locale === 'en-US' ? enUS : zhCN;
  return render(
    <IntlProvider locale={locale} messages={messages}>
      <ConfigProvider theme={{ algorithm: theme.darkAlgorithm }}>
        <GpsSyncConfirmModal
          open
          device={targetDevice}
          onCancel={() => {}}
          onConfirm={onConfirm}
        />
      </ConfigProvider>
    </IntlProvider>,
  );
}

describe('GpsSyncConfirmModal', () => {
  it('renders the English title and original GPS confirmation content', () => {
    renderDialog('en-US');

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(screen.getByText('Operation Confirmation')).toBeInTheDocument();
    expect(screen.queryByText('操作确认')).not.toBeInTheDocument();
    expect(screen.getByText('Use device-reported coordinates to overwrite OMC coordinates?')).toBeInTheDocument();
    expect(screen.getByText('OMC coordinates: Longitude 115.366462, Latitude 25.92416')).toBeInTheDocument();
    expect(screen.getByText('Device-reported coordinates: Longitude 115.376, Latitude 25.9341, GPS Height(m) 174')).toBeInTheDocument();
    expect(screen.getByText('Horizontal difference: 1500 m')).toBeInTheDocument();
    expect(screen.getByText('Observed at: 2026-07-17 10:00:00')).toBeInTheDocument();
    expect(screen.getByText('Coordinate slot: 1')).toBeInTheDocument();
    const dialog = screen.getByRole('dialog');
    expect(dialog).not.toHaveTextContent('Source path');
    expect(dialog).not.toHaveTextContent('Device.FAP.GPS');
    expect(screen.getByRole('alert')).toHaveTextContent('Data inconsistent, Sure to synchronize?');
    expect(dialog.textContent).not.toMatch(/[\u3400-\u9fff]/u);
  });

  it('renders the Chinese title and preserves confirm behavior', () => {
    const onConfirm = vi.fn();
    renderDialog('zh-CN', onConfirm);

    expect(screen.getByText('操作确认')).toBeInTheDocument();
    expect(screen.getByText('采集时间：2026-07-17 10:00:00')).toBeInTheDocument();
    expect(screen.getByText('坐标槽位：第 1 组')).toBeInTheDocument();
    expect(screen.getByRole('dialog')).not.toHaveTextContent('参数来源');
    fireEvent.click(screen.getByRole('button', { name: '确 认' }));
    expect(onConfirm).toHaveBeenCalledOnce();
  });

  it('shows synchronized coordinates as information without a false warning or confirm action', () => {
    renderDialog('en-US', () => {}, {
      ...device,
      locationSync: {
        ...device.locationSync,
        status: 'in_sync',
        distanceMeters: 0,
      },
    });

    expect(screen.getByText('GPS Coordinate Details')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('GPS coordinates are synchronized');
    expect(screen.queryByText('Data inconsistent, Sure to synchronize?')).not.toBeInTheDocument();
    expect(screen.queryByRole('button', { name: 'Confirm' })).not.toBeInTheDocument();
    expect(screen.getByText('Close', { selector: '.ant-modal-footer button span' })).toBeInTheDocument();
  });
});
