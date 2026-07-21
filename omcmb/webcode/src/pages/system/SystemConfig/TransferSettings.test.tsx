import { useEffect } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Form } from 'antd';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n';
import TransferSettings from './TransferSettings';

function TransferSettingsHarness({ values }: { values: Record<string, unknown> }) {
  const [form] = Form.useForm();

  useEffect(() => {
    form.setFieldsValue(values);
  }, [form, values]);

  return (
    <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
      <TransferSettings form={form} />
    </IntlProvider>
  );
}

describe('TransferSettings', () => {
  it('保留已保存的 HTTPS、自定义端口和路径，不回写固定地址', async () => {
    render(
      <TransferSettingsHarness
        values={{
          uploadBaseURL: 'https://edge.example.com:9443/custom/upload',
          downloadBaseURL: 'https://edge.example.com:9443/custom/download',
          uploadPath: '/custom/upload-path',
          downloadPath: '/custom/download-path',
        }}
      />,
    );

    await waitFor(() => {
      expect(screen.getByDisplayValue('https://edge.example.com:9443/custom/upload')).toBeInTheDocument();
      expect(screen.getByDisplayValue('https://edge.example.com:9443/custom/download')).toBeInTheDocument();
      expect(screen.getByDisplayValue('/custom/upload-path')).toBeInTheDocument();
      expect(screen.getByDisplayValue('/custom/download-path')).toBeInTheDocument();
    });
  });

  it('拒绝在 Base URL 中混入查询参数', async () => {
    render(<TransferSettingsHarness values={{ uploadBaseURL: '' }} />);

    const input = screen.getByLabelText('上传服务基础URL');
    fireEvent.change(input, { target: { value: 'https://edge.example.com/omc?token=secret' } });
    fireEvent.blur(input);

    expect(await screen.findByText('基础地址不能包含账号、查询参数或片段')).toBeInTheDocument();
  });

  it('拒绝可能逃逸服务目录的路径', async () => {
    render(<TransferSettingsHarness values={{ uploadPath: '' }} />);

    const input = screen.getByLabelText('上传服务路径');
    fireEvent.change(input, { target: { value: '/smallcell/../admin' } });
    fireEvent.blur(input);

    expect(await screen.findByText('路径必须以 / 开头，且不能包含查询参数、片段或目录跳转')).toBeInTheDocument();
  });
});
