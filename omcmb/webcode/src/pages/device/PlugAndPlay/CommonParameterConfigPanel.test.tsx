import { Form } from 'antd';
import { fireEvent, render, screen } from '@testing-library/react';
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
});
