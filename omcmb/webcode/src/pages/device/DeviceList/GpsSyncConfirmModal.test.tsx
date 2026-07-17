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
    expect(screen.getByText('Longitude: 115.366462 Latitude: 25.92416 GPS Height(m): 174')).toBeInTheDocument();
    expect(screen.getByRole('alert')).toHaveTextContent('Data inconsistent, Sure to synchronize?');
  });

  it('renders the Chinese title and preserves confirm behavior', () => {
    const onConfirm = vi.fn();
    renderDialog('zh-CN', onConfirm);

    expect(screen.getByText('操作确认')).toBeInTheDocument();
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
