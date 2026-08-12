import { fireEvent, render, screen } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { describe, expect, it, vi } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import type { AntennaSector, MapDevice } from '@core/types/map';
import MapPopup from './MapPopup';

vi.mock('@/hooks/useThemeToken', () => ({
  useThemeToken: () => ({
    colorPrimary: '#1677ff',
    colorPrimaryHover: '#4096ff',
  }),
}));

const device: MapDevice = {
  id: 'device-1',
  lat: 30.1,
  lng: 120.1,
  name: '测试设备',
  status: 'onlineActive',
  sn: '1215000058258TB0026',
  alarmCount: 0,
};

const validSector: AntennaSector = {
  number: 1,
  azimuth: 120,
  antennaHeight: 27,
  mechanicalDowntilt: 1,
  horizontalBeamwidth: 0.01,
  verticalBeamwidth: 1,
  nearRadiusMeters: 1031,
  farRadiusMeters: 3094,
  fieldSources: {},
  directionAvailable: true,
  coverageAvailable: true,
  coverageStatus: 'available',
  missingFields: [],
};

function renderPanel(sectors: AntennaSector[], extra: Partial<React.ComponentProps<typeof MapPopup>> = {}) {
  return render(
    <IntlProvider locale="zh-CN" messages={zhCN}>
      <MapPopup
        device={device}
        variant="panel"
        antennaSectors={sectors}
        activeSectorNumber={sectors[0]?.number}
        {...extra}
      />
    </IntlProvider>,
  );
}

describe('MapPopup panel', () => {
  it('在右侧面板显示窄波瓣提示并支持关闭', () => {
    const onClose = vi.fn();
    renderPanel([validSector], { activeSectorRenderMode: 'narrow', onClose });

    expect(screen.getByText(/当前波瓣过窄/)).toBeInTheDocument();
    expect(screen.getByText('1031 - 3094 m')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('天线图层尚未启用时提示继续放大', () => {
    renderPanel([validSector], { activeSectorRenderMode: 'unavailable' });

    expect(screen.getByText('请将地图放大到 13 级以上以显示天线扇区')).toBeInTheDocument();
    expect(screen.queryByText(/当前波瓣过窄/)).not.toBeInTheDocument();
  });

  it('区分无效几何并阻止保存', () => {
    const invalidSector: AntennaSector = {
      ...validSector,
      mechanicalDowntilt: 1,
      verticalBeamwidth: 3,
      nearRadiusMeters: undefined,
      farRadiusMeters: undefined,
      coverageAvailable: false,
      coverageStatus: 'invalid_geometry',
      coverageIssue: 'far_angle_not_positive',
    };
    renderPanel([invalidSector], { onAntennaSave: vi.fn() });

    expect(screen.getByText(/机械下倾角必须大于垂直波宽的一半/)).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '编辑' }));
    expect(screen.getByRole('button', { name: /保\s*存/ })).toBeDisabled();
  });
});
