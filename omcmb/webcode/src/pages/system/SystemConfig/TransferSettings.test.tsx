import { useEffect } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Form } from 'antd';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n';
import TransferSettings from './TransferSettings';

function TransferSettingsHarness({
  values,
  onFormReady,
}: {
  values: Record<string, unknown>;
  onFormReady?: (form: ReturnType<typeof Form.useForm>[0]) => void;
}) {
  const [form] = Form.useForm();

  useEffect(() => {
    form.setFields(
      Object.entries(values).map(([name, value]) => ({ name, value, touched: false })),
    );
    onFormReady?.(form);
  }, [form, onFormReady, values]);

  return (
    <IntlProvider locale="zh-CN" messages={getMessages('zh-CN')}>
      <TransferSettings form={form} />
    </IntlProvider>
  );
}

describe('TransferSettings', () => {
  it('为标准 ACS 网关显示两个独立 IP 输入框和固定的 HTTP 8080', () => {
    render(<TransferSettingsHarness values={{}} />);

    expect(screen.getByLabelText('上传服务IP')).toBeInTheDocument();
    expect(screen.getByLabelText('下载服务IP')).toBeInTheDocument();
    expect(screen.getAllByText('http://')).toHaveLength(2);
    expect(screen.getAllByText(':8080')).toHaveLength(2);
    expect(screen.getByText(/生产环境允许运营商网络中基站可达的私网地址/)).toBeInTheDocument();
  });

  it('提供正确且可编辑的上传和下载服务路径默认值', () => {
    render(<TransferSettingsHarness values={{}} />);

    expect(screen.getByDisplayValue('/smallcell/FileUploadService')).toBeEnabled();
    expect(screen.getByDisplayValue('/smallcell/FileDownloadService')).toBeEnabled();
    expect(screen.getByText(/上传默认路径：\/smallcell\/FileUploadService/)).toBeInTheDocument();
    expect(screen.getByText(/下载默认路径：\/smallcell\/FileDownloadService/)).toBeInTheDocument();
  });

  it('用户分别输入两个 IP 后保存为完整的 HTTP 8080 Base URL', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{}}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    fireEvent.change(screen.getByLabelText('上传服务IP'), {
      target: { value: '172.24.224.251' },
    });
    fireEvent.change(screen.getByLabelText('下载服务IP'), {
      target: { value: '172.24.224.252' },
    });

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe('http://172.24.224.251:8080');
      expect(form?.getFieldValue('downloadBaseURL')).toBe('http://172.24.224.252:8080');
      expect(form?.isFieldTouched('uploadBaseURL')).toBe(true);
      expect(form?.isFieldTouched('downloadBaseURL')).toBe(true);
    });
  });

  it('逐字输入 IPv4 时不被 URL 解析器改写', async () => {
    const user = userEvent.setup();
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{}}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    const input = screen.getByLabelText('上传服务IP');
    await user.type(input, '172.24.224.251');

    expect(input).toHaveValue('172.24.224.251');
    expect(form?.getFieldValue('uploadBaseURL')).toBe('http://172.24.224.251:8080');
  });

  it('加载存量自定义地址和路径时不自动回写配置', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    const values = {
      uploadBaseURL: 'https://edge.example.com:9443/custom/upload',
      downloadBaseURL: 'https://edge.example.com:9443/custom/download',
      uploadPath: '/custom/upload-path',
      downloadPath: '/custom/download-path',
    };
    render(
      <TransferSettingsHarness
        values={values}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe(values.uploadBaseURL);
      expect(form?.getFieldValue('downloadBaseURL')).toBe(values.downloadBaseURL);
      expect(form?.getFieldValue('uploadPath')).toBe(values.uploadPath);
      expect(form?.getFieldValue('downloadPath')).toBe(values.downloadPath);
      expect(form?.isFieldsTouched()).toBe(false);
      expect(screen.getByDisplayValue(values.uploadBaseURL)).toBeInTheDocument();
      expect(screen.getByDisplayValue(values.downloadBaseURL)).toBeInTheDocument();
      expect(screen.getByDisplayValue('/custom/upload-path')).toBeInTheDocument();
      expect(screen.getByDisplayValue('/custom/download-path')).toBeInTheDocument();
    });
  });

  it('允许高级部署直接保存 HTTPS、自定义端口和反向代理前缀', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{}}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    const customURL = 'https://edge.example.com:9443/omc';
    fireEvent.change(screen.getByLabelText('上传服务IP'), {
      target: { value: customURL },
    });

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe(customURL);
      expect(screen.getByDisplayValue(customURL)).toBeInTheDocument();
    });
  });

  it('允许把存量自定义 URL 直接替换为标准部署 IP', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{ uploadBaseURL: 'https://edge.example.com:9443/omc' }}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    fireEvent.change(screen.getByLabelText('上传服务IP'), {
      target: { value: '172.24.224.251' },
    });

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe('http://172.24.224.251:8080');
      expect(screen.getByLabelText('上传服务IP')).toHaveValue('172.24.224.251');
      expect(screen.getAllByText('http://')).toHaveLength(2);
      expect(screen.getAllByText(':8080')).toHaveLength(2);
    });
  });

  it('拒绝在 IP 输入框中混入协议、端口或路径', async () => {
    render(<TransferSettingsHarness values={{}} />);

    const input = screen.getByLabelText('上传服务IP');
    fireEvent.change(input, { target: { value: '172.24.224.251:9090' } });
    fireEvent.blur(input);

    expect(await screen.findByText('请输入有效的 IP 地址或主机名，不要包含协议、端口或路径')).toBeInTheDocument();
  });

  it('拒绝可能逃逸服务目录的路径', async () => {
    render(<TransferSettingsHarness values={{ uploadPath: '' }} />);

    const input = screen.getByLabelText('上传服务路径');
    fireEvent.change(input, { target: { value: '/smallcell/../admin' } });
    fireEvent.blur(input);

    expect(await screen.findByText('路径必须以 / 开头，且不能包含查询参数、片段或目录跳转')).toBeInTheDocument();
  });
});
