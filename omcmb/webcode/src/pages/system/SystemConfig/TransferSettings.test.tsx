import { useEffect } from 'react';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Form } from 'antd';
import { IntlProvider } from 'react-intl';
import { getMessages } from '@core/i18n';
import TransferSettings from './TransferSettings';
import { buildBatchItems } from './sysConfigSerialize';

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
  it('默认强制使用 HTTP 并展示四个协议方向地址', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{}}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    await waitFor(() => {
      expect(form?.getFieldValue('protocolPolicy')).toBe('force_http');
    });
    expect(screen.getByText('基站文件传输协议')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTP 上传地址')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTPS 上传地址')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTP 下载地址')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTPS 下载地址')).toBeInTheDocument();
  });

  it('切换为 HTTPS 优先后要求同时填写两个 HTTPS 地址', async () => {
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

    await user.click(screen.getByText('优先使用 HTTPS 协议'));

    await waitFor(() => {
      expect(form?.getFieldValue('protocolPolicy')).toBe('prefer_https');
    });
    await expect(
      form!.validateFields(['httpsUploadBaseURL', 'httpsDownloadBaseURL']),
    ).rejects.toBeDefined();
    expect(await screen.findAllByText('HTTPS 优先时必须填写该地址')).toHaveLength(2);
  });

  it('HTTPS 地址字段非空时拒绝 HTTP scheme', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{
          protocolPolicy: 'force_http',
          httpsUploadBaseURL: 'http://acs.example.com:8080',
        }}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    await expect(form!.validateFields(['httpsUploadBaseURL'])).rejects.toBeDefined();
    expect(
      await screen.findByText('请输入以 https:// 开头的合法基础地址，且不能包含账号、查询参数、片段或目录跳转'),
    ).toBeInTheDocument();
  });

  it('HTTP 和 HTTPS 地址都拒绝端口 0', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{
          uploadBaseURL: 'http://acs.example.com:0',
          httpsUploadBaseURL: 'https://acs.example.com:0',
        }}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    await expect(
      form!.validateFields(['uploadBaseURL', 'httpsUploadBaseURL']),
    ).rejects.toBeDefined();
    expect(form!.getFieldError('uploadBaseURL')).not.toHaveLength(0);
    expect(form!.getFieldError('httpsUploadBaseURL')).not.toHaveLength(0);
  });

  it('HTTP 地址字段拒绝 HTTPS scheme', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{ uploadBaseURL: 'https://acs.example.com:8443/upload' }}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    await expect(form!.validateFields(['uploadBaseURL'])).rejects.toBeDefined();
    expect(
      await screen.findByText('检测到旧 HTTP 字段中保存了 HTTPS 地址。请将该地址迁移到对应的 HTTPS 地址字段，并为此字段填写 http:// 地址'),
    ).toBeInTheDocument();
  });

  it('回显并保存 HTTPS 优先策略与两个 HTTPS 地址', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    const values = {
      protocolPolicy: 'prefer_https',
      httpsUploadBaseURL: 'https://acs.example.com:8443/upload',
      httpsDownloadBaseURL: 'https://acs.example.com:8443/download',
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
      expect(form?.getFieldValue('protocolPolicy')).toBe('prefer_https');
      expect(screen.getByDisplayValue(values.httpsUploadBaseURL)).toBeInTheDocument();
      expect(screen.getByDisplayValue(values.httpsDownloadBaseURL)).toBeInTheDocument();
    });

    const submitted = await form!.validateFields([
      'protocolPolicy',
      'httpsUploadBaseURL',
      'httpsDownloadBaseURL',
    ]);
    expect(buildBatchItems(submitted, [])).toEqual([
      { key: 'protocolPolicy', value: 'prefer_https', value_type: 'string' },
      { key: 'httpsUploadBaseURL', value: values.httpsUploadBaseURL, value_type: 'string' },
      { key: 'httpsDownloadBaseURL', value: values.httpsDownloadBaseURL, value_type: 'string' },
    ]);
  });

  it('为标准 ACS 网关显示两个独立 IP 输入框和固定的 HTTP 8080', () => {
    render(<TransferSettingsHarness values={{}} />);

    expect(screen.getByLabelText('HTTP 上传地址')).toBeInTheDocument();
    expect(screen.getByLabelText('HTTP 下载地址')).toBeInTheDocument();
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

    fireEvent.change(screen.getByLabelText('HTTP 上传地址'), {
      target: { value: '172.24.224.251' },
    });
    fireEvent.change(screen.getByLabelText('HTTP 下载地址'), {
      target: { value: '172.24.224.252' },
    });

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe('http://172.24.224.251:8080');
      expect(form?.getFieldValue('downloadBaseURL')).toBe('http://172.24.224.252:8080');
      expect(form?.isFieldTouched('uploadBaseURL')).toBe(true);
      expect(form?.isFieldTouched('downloadBaseURL')).toBe(true);
    });

    const values = await form!.validateFields(['uploadBaseURL', 'downloadBaseURL']);
    expect(buildBatchItems(values, [])).toEqual([
      {
        key: 'uploadBaseURL',
        value: 'http://172.24.224.251:8080',
        value_type: 'string',
      },
      {
        key: 'downloadBaseURL',
        value: 'http://172.24.224.252:8080',
        value_type: 'string',
      },
    ]);
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

    const input = screen.getByLabelText('HTTP 上传地址');
    await user.type(input, '172.24.224.251');

    expect(input).toHaveValue('172.24.224.251');
    expect(form?.getFieldValue('uploadBaseURL')).toBe('http://172.24.224.251:8080');
  });

  it('异常 URL 必须原样显示且不能通过表单校验', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    const malformedURL = 'http:////172.24.224.251:8080';
    render(
      <TransferSettingsHarness
        values={{ uploadBaseURL: malformedURL }}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    expect(screen.getByLabelText('HTTP 上传地址')).toHaveValue(malformedURL);
    expect(form).toBeDefined();
    await expect(form!.validateFields(['uploadBaseURL'])).rejects.toBeDefined();
  });

  it('粘贴带双斜杠的主机内容时不生成隐藏的畸形 URL', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{}}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    const malformedHost = '//172.24.224.251';
    fireEvent.change(screen.getByLabelText('HTTP 上传地址'), {
      target: { value: malformedHost },
    });

    await waitFor(() => {
      expect(screen.getByLabelText('HTTP 上传地址')).toHaveValue(malformedHost);
      expect(form?.getFieldValue('uploadBaseURL')).toBe(malformedHost);
    });
    await expect(form!.validateFields(['uploadBaseURL'])).rejects.toBeDefined();
  });

  it('标准地址控件限制最大宽度以让固定端口紧邻 IP', () => {
    render(<TransferSettingsHarness values={{}} />);

    const addressGroup = screen.getByLabelText('HTTP 上传地址').closest('.ant-space-compact');
    expect(addressGroup).toHaveStyle({
      width: '100%',
      maxWidth: '520px',
    });
  });

  it('加载存量自定义地址和路径时不自动回写配置', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    const values = {
      uploadBaseURL: 'http://edge.example.com:9080/custom/upload',
      downloadBaseURL: 'http://edge.example.com:9080/custom/download',
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

  it('允许 HTTP 地址使用自定义端口和反向代理前缀', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{}}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    const customURL = 'http://edge.example.com:9080/omc';
    fireEvent.change(screen.getByLabelText('HTTP 上传地址'), {
      target: { value: customURL },
    });

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe(customURL);
      expect(screen.getByDisplayValue(customURL)).toBeInTheDocument();
    });
    await expect(form!.validateFields(['uploadBaseURL'])).resolves.toEqual({ uploadBaseURL: customURL });
  });

  it('允许把存量自定义 URL 直接替换为标准部署 IP', async () => {
    let form: ReturnType<typeof Form.useForm>[0] | undefined;
    render(
      <TransferSettingsHarness
        values={{ uploadBaseURL: 'http://edge.example.com:9080/omc' }}
        onFormReady={(instance) => {
          form = instance;
        }}
      />,
    );

    fireEvent.change(screen.getByLabelText('HTTP 上传地址'), {
      target: { value: '172.24.224.251' },
    });

    await waitFor(() => {
      expect(form?.getFieldValue('uploadBaseURL')).toBe('http://172.24.224.251:8080');
      expect(screen.getByLabelText('HTTP 上传地址')).toHaveValue('172.24.224.251');
      expect(screen.getAllByText('http://')).toHaveLength(2);
      expect(screen.getAllByText(':8080')).toHaveLength(2);
    });
  });

  it('拒绝在 IP 输入框中混入协议、端口或路径', async () => {
    render(<TransferSettingsHarness values={{}} />);

    const input = screen.getByLabelText('HTTP 上传地址');
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
