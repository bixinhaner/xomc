import { Form } from 'antd';
import { fireEvent, render, screen, within } from '@testing-library/react';
import { IntlProvider } from 'react-intl';
import { describe, expect, it } from 'vitest';
import zhCN from '@core/i18n/zh-CN';
import { ParameterConfigFields } from './CommonParameterConfigPanel';
import { toParamConfigFormValues } from './paramConfigDetail';

function Harness({ readOnly }: { readOnly: boolean }) {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={{
          sheetParameters: {
            DEVICE: [{
              'Serial Number': 'SN-ISSUE-348',
              'NTP Server1': '192.0.2.1',
              'NTP Server2': '192.0.2.2',
              'NTP Server3': '192.0.2.3',
              'NTP Server4': '192.0.2.4',
              'NTP Server5': '192.0.2.5',
              'Local Time Zone': 'UTC+08:00',
              URL: 'http://acs.example.test',
              'Periodic Inform Enable': '1',
              'Periodic Inform Time': '2026-08-20T16:00:00Z',
              'Periodic Inform Interval': '300',
              PpsTimeMode: 'GPS_PPS',
            }],
          },
        }}
      >
        <ParameterConfigFields deviceType="gNB" scope="device" readOnly={readOnly} />
      </Form>
    </IntlProvider>
  );
}

function MultiCellHarness() {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={toParamConfigFormValues({
          deviceType: 'gNB',
          serialNumber: 'SN-MULTI-CELL',
          sheetParameters: {
            CELL: [
              {
                'Serial Number': 'SN-MULTI-CELL',
                'Cell Index': 1,
                'Num Of Tx Antenna': '2',
                'Num Of Rx Antenna': '4',
                'Pattern1 DL Slots': '66',
              },
              {
                'Serial Number': 'SN-MULTI-CELL',
                'Cell Index': 2,
                'Num Of Tx Antenna': '6',
                'Num Of Rx Antenna': '8',
                'Pattern1 DL Slots': '166',
              },
            ],
          },
        })}
      >
        <ParameterConfigFields deviceType="gNB" scope="device" readOnly />
      </Form>
    </IntlProvider>
  );
}

function BlankCellHarness() {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={toParamConfigFormValues({
          deviceType: 'gNB',
          serialNumber: 'SN-BLANK-CELL',
          sheetParameters: {
            CELL: [{ 'Serial Number': 'SN-BLANK-CELL', 'Cell Index': 1 }],
          },
        })}
      >
        <ParameterConfigFields deviceType="gNB" scope="device" readOnly={false} />
      </Form>
    </IntlProvider>
  );
}

function CommonGnbCellHarness() {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={toParamConfigFormValues({
          deviceType: 'gNB',
          serialNumber: 'SN-COMMON-CELL',
          sheetParameters: {
            CELL: [{ 'Serial Number': 'SN-COMMON-CELL', 'Cell Index': 1, '*PCI': 10 }],
          },
        })}
      >
        <ParameterConfigFields deviceType="gNB" scope="common" readOnly={false} />
      </Form>
    </IntlProvider>
  );
}

function EnbCellHarness() {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={toParamConfigFormValues({
          deviceType: 'eNB',
          serialNumber: 'SN-ENB-CELL',
          sheetParameters: {
            CELL: [{
              '*SERIAL_NUMBER': 'SN-ENB-CELL',
              '*CELL_NUMBER': 1,
              '*BAND': '40',
              '*BANDWIDTH_DL': 'n50',
              X_COM_MaxTxPowerExpanded: '18',
              MaxTxPower: '20',
              SUBFRAME_ASSIGNMENT: '1',
            }],
          },
        })}
      >
        <ParameterConfigFields deviceType="eNB" scope="device" readOnly={false} />
      </Form>
    </IntlProvider>
  );
}

function GsmCellHarness() {
  const [form] = Form.useForm();
  return (
    <IntlProvider locale="zh-CN" defaultLocale="zh-CN" messages={zhCN}>
      <Form
        form={form}
        initialValues={toParamConfigFormValues({
          deviceType: 'GSM',
          serialNumber: 'SN-GSM-BTS',
          productClass: 'Example BSC Product',
          sheetParameters: {
            GSM: [{
              'Serial Number': 'SN-GSM-BTS',
              'BTS Index': 1,
              IPA: '6969',
              'Unit ID': '7',
              'WAN IP': '192.0.2.10',
              Synchronization: 'GNSS',
            }],
          },
        })}
      >
        <ParameterConfigFields deviceType="GSM" productClass="Example BSC Product" scope="device" readOnly={false} />
      </Form>
    </IntlProvider>
  );
}

describe('device parameter config detail fields', () => {
  it('shows imported workbook values in matching page fields in view mode', () => {
    render(<Harness readOnly />);

    expect(screen.queryByText('导入工作表参数')).not.toBeInTheDocument();
    [
      '192.0.2.1',
      '192.0.2.2',
      '192.0.2.3',
      '192.0.2.4',
      '192.0.2.5',
      'http://acs.example.test',
      '2026-08-20T16:00:00Z',
      '300',
      'UTC+08:00',
      'GPS_PPS',
    ].forEach((value) => {
      const field = screen.getByDisplayValue(value);
      expect(field, value).not.toBeDisabled();
      expect(field, value).toHaveAttribute('readonly');
    });
  });

  it('shows imported workbook values in matching page fields in edit mode', () => {
    const { container } = render(<Harness readOnly={false} />);

    expect(screen.queryByText('导入工作表参数')).not.toBeInTheDocument();
    [
      '192.0.2.1',
      '192.0.2.2',
      '192.0.2.3',
      '192.0.2.4',
      '192.0.2.5',
      'http://acs.example.test',
      '2026-08-20T16:00:00Z',
      '300',
    ].forEach((value) => {
      const field = screen.getByDisplayValue(value);
      expect(field, value).not.toBeDisabled();
      expect(field, value).not.toHaveAttribute('readonly');
    });
    expect(container).toHaveTextContent('UTC+08:00');
    expect(container).toHaveTextContent('GPS_PPS');
  });

  it('shows imported workbook values for every mapped multi-cell row', () => {
    render(<MultiCellHarness />);

    fireEvent.click(screen.getByText('小区 1').closest('.ant-collapse-header')!);
    fireEvent.click(screen.getByText('小区 2').closest('.ant-collapse-header')!);

    ['4', '66', '6', '8', '166'].forEach((value) => {
      const field = screen.getByDisplayValue(value);
      expect(field, value).not.toBeDisabled();
      expect(field, value).toHaveAttribute('readonly');
    });
    expect(screen.queryByText('编辑')).not.toBeInTheDocument();
    expect(screen.queryByText('新增小区')).not.toBeInTheDocument();
  });

  it('shows carrier bandwidth options before subcarrier spacing is selected', async () => {
    render(<BlankCellHarness />);

    fireEvent.click(screen.getByText('小区 1').closest('.ant-collapse-header')!);
    const bandwidthItem = screen.getByText('下行载波带宽').closest('.ant-form-item')!;
    fireEvent.mouseDown(within(bandwidthItem).getByRole('combobox'));

    expect(await screen.findByText('10MHz(24RB)')).toBeInTheDocument();
    expect(screen.queryByText('暂无数据')).not.toBeInTheDocument();
  });

  it('uses PCI allocation only in common gNB config and hides the per-cell PCI field', () => {
    render(<CommonGnbCellHarness />);

    expect(screen.getByText('PCI')).toBeInTheDocument();
    fireEvent.click(screen.getByText('小区 1').closest('.ant-collapse-header')!);

    expect(screen.queryByText(/^PCI \[/)).not.toBeInTheDocument();
    expect(screen.queryByDisplayValue('10')).not.toBeInTheDocument();
  });

  it('renders eNB cell parameters with quick-setting labels and dropdowns', async () => {
    render(<EnbCellHarness />);

    fireEvent.click(screen.getByText('小区 1').closest('.ant-collapse-header')!);
    expect(screen.getByText('小区参数')).toBeInTheDocument();
    expect(screen.queryByText('BANDWIDTH_DL')).not.toBeInTheDocument();
    expect(screen.queryByText('最大发射功率')).not.toBeInTheDocument();
    expect(screen.queryByText('频段支持')).not.toBeInTheDocument();
    expect(screen.getByText('发射功率')).toBeInTheDocument();
    expect(screen.getByDisplayValue('18')).not.toBeDisabled();

    const bandwidthItem = screen.getByText('带宽').closest('.ant-form-item')!;
    fireEvent.mouseDown(within(bandwidthItem).getByRole('combobox'));
    expect(await screen.findByText('5MHz')).toBeInTheDocument();
  });

  it('renders GSM radio instance parameters by template module', () => {
    render(<GsmCellHarness />);

    fireEvent.click(screen.getByText('BTS 1').closest('.ant-collapse-header')!);
    expect(screen.getByText('ABIS 参数')).toBeInTheDocument();
    expect(screen.getByText('其他模板参数')).toBeInTheDocument();
    expect(screen.getByDisplayValue('6969')).not.toBeDisabled();
    expect(screen.getByDisplayValue('192.0.2.10')).not.toBeDisabled();
  });
});
