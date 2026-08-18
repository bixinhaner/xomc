import { describe, expect, it } from 'vitest';
import { parseMmlDeviceTaskResult } from '../mmlResultParser';

// 真实 GPV 响应样本（cwmp:GetParameterValuesResponse + ParameterValueStruct[N]）。
// 来自 CWMP 标准 + scripts/cpe_simulator.py 模板，多命名空间前缀刻意混用以验证解析鲁棒性。
const gpvSample = `<?xml version="1.0" encoding="UTF-8"?>
<soap-env:Envelope xmlns:soap-env="http://schemas.xmlsoap.org/soap/envelope/"
                   xmlns:soap-enc="http://schemas.xmlsoap.org/soap/encoding/"
                   xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance"
                   xmlns:xsd="http://www.w3.org/2001/XMLSchema"
                   xmlns:cwmp="urn:dslforum-org:cwmp-1-0">
  <soap-env:Body>
    <cwmp:GetParameterValuesResponse>
      <ParameterList soap-enc:arrayType="cwmp:ParameterValueStruct[3]">
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.UserLabel</Name>
          <Value xsi:type="xsd:string">test</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.Manufacturer</Name>
          <Value xsi:type="xsd:string">BAICELLS</Value>
        </ParameterValueStruct>
        <ParameterValueStruct>
          <Name>Device.DeviceInfo.UpTime</Name>
          <Value xsi:type="xsd:unsignedInt">123456</Value>
        </ParameterValueStruct>
      </ParameterList>
    </cwmp:GetParameterValuesResponse>
  </soap-env:Body>
</soap-env:Envelope>`;

const spvSample = `<?xml version="1.0"?>
<soap:Envelope xmlns:soap="x" xmlns:cwmp="y">
  <soap:Body>
    <cwmp:SetParameterValuesResponse><Status>0</Status></cwmp:SetParameterValuesResponse>
  </soap:Body>
</soap:Envelope>`;

const spvRebootSample = `<soap:Envelope xmlns:soap="x" xmlns:cwmp="y"><soap:Body>
<cwmp:SetParameterValuesResponse><Status>1</Status></cwmp:SetParameterValuesResponse>
</soap:Body></soap:Envelope>`;

const addSample = `<soap:Envelope xmlns:soap="x" xmlns:cwmp="y"><soap:Body>
<cwmp:AddObjectResponse><InstanceNumber>7</InstanceNumber><Status>0</Status></cwmp:AddObjectResponse>
</soap:Body></soap:Envelope>`;

describe('parseMmlDeviceTaskResult', () => {
  it('GPV → 解析出每条 path 的 name/value/type', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: gpvSample,
    });
    expect(got?.kind).toBe('gpv');
    expect(got?.params).toEqual([
      { name: 'Device.DeviceInfo.UserLabel', value: 'test', type: 'xsd:string' },
      { name: 'Device.DeviceInfo.Manufacturer', value: 'BAICELLS', type: 'xsd:string' },
      { name: 'Device.DeviceInfo.UpTime', value: '123456', type: 'xsd:unsignedInt' },
    ]);
  });

  // issue #424：standard_parameter_values 存在时优先用它（标准 path），不再解析 raw_response 的私有 path。
  it('GPV → 优先用 ACS 回译的 standard_parameter_values（标准 path）', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: gpvSample, // 私有 path 原文（保留供 XmlViewer），但应被 standard 覆盖
      standard_parameter_values: [
        { name: 'Device.WiFi.SSID', value: 'home', type: 'xsd:string' },
        { name: 'Device.LAN.IP', value: '192.168.1.1', type: 'xsd:string' },
      ],
    });
    expect(got?.kind).toBe('gpv');
    expect(got?.params).toEqual([
      { name: 'Device.WiFi.SSID', value: 'home', type: 'xsd:string' },
      { name: 'Device.LAN.IP', value: '192.168.1.1', type: 'xsd:string' },
    ]);
  });

  it('GPV → standard_parameter_values 缺失/空时回退解析 raw_response', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: gpvSample,
      standard_parameter_values: [],
    });
    expect(got?.params).toHaveLength(3); // 回退到 raw 解析
  });

  it('GPV → 超大 raw_response 默认完整解析，避免任务详情/CSV 静默截断', () => {
    const structs = Array.from({ length: 85 }, (_v, i) => `
      <ParameterValueStruct>
        <Name>DeviceGSM.Bts.1.Param${i}</Name>
        <Value xsi:type="xsd:string">v&amp;${i}</Value>
      </ParameterValueStruct>
    `).join('');
    const got = parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: `<GetParameterValuesResponse>${structs}</GetParameterValuesResponse>${' '.repeat(501_000)}`,
    });

    expect(got?.kind).toBe('gpv');
    expect(got?.params).toHaveLength(85);
    expect(got?.params?.[0]).toEqual({ name: 'DeviceGSM.Bts.1.Param0', value: 'v&0', type: 'xsd:string' });
    expect(got?.params?.[84]?.name).toBe('DeviceGSM.Bts.1.Param84');
  });

  it('GPV → 调用方显式 maxParams 时才截断大 raw_response', () => {
    const structs = Array.from({ length: 85 }, (_v, i) => `
      <ParameterValueStruct>
        <Name>DeviceGSM.Bts.1.Param${i}</Name>
        <Value xsi:type="xsd:string">v${i}</Value>
      </ParameterValueStruct>
    `).join('');
    const got = parseMmlDeviceTaskResult(
      {
        method: 'GetParameterValuesResponse',
        raw_response: `<GetParameterValuesResponse>${structs}</GetParameterValuesResponse>${' '.repeat(501_000)}`,
      },
      { maxParams: 80 },
    );

    expect(got?.params).toHaveLength(80);
    expect(got?.params?.[79]?.name).toBe('DeviceGSM.Bts.1.Param79');
  });

  it('GPV → standard_parameter_values 也只在调用方显式 maxParams 时截断', () => {
    const values = Array.from({ length: 3 }, (_v, i) => ({ name: `Device.X.${i}`, value: `v${i}` }));
    expect(parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: gpvSample,
      standard_parameter_values: values,
    })?.params).toHaveLength(3);

    expect(parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: gpvSample,
      standard_parameter_values: values,
    }, { maxParams: 2 })?.params).toHaveLength(2);
  });

  it('GPV → 先按调用方条件过滤，再应用 maxParams', () => {
    const values = [
      ...Array.from({ length: 85 }, (_v, i) => ({ name: `Device.Noise.${i}`, value: `n${i}` })),
      { name: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth', value: '100' },
      { name: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ULBandwidth', value: '100' },
    ];
    const got = parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: gpvSample,
      standard_parameter_values: values,
    }, {
      maxParams: 80,
      includeParam: (param) => param.name.endsWith('Bandwidth'),
    });

    expect(got?.params).toEqual([
      { name: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.DLBandwidth', value: '100', type: undefined },
      { name: 'Device.Services.FAPService.1.CellConfig.LTE.RAN.RF.ULBandwidth', value: '100', type: undefined },
    ]);
  });

  it('SPV Status=0 → 立即生效', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'SetParameterValuesResponse',
      raw_response: spvSample,
    });
    expect(got).toEqual({ kind: 'spv', status: 0 });
  });

  it('SPV Status=1 → 需重启', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'SetParameterValuesResponse',
      raw_response: spvRebootSample,
    });
    expect(got).toEqual({ kind: 'spv', status: 1 });
  });

  it('AddObject → InstanceNumber + Status', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'AddObjectResponse',
      raw_response: addSample,
    });
    expect(got).toEqual({ kind: 'add', instanceNumber: 7, status: 0 });
  });

  it('Reboot 空 body → status=0', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'RebootResponse',
      raw_response: '<cwmp:RebootResponse/>',
    });
    expect(got).toEqual({ kind: 'reboot', status: 0 });
  });

  it('null / undefined / 缺 raw_response → 返 null', () => {
    expect(parseMmlDeviceTaskResult(null)).toBeNull();
    expect(parseMmlDeviceTaskResult({})).toBeNull();
    expect(parseMmlDeviceTaskResult({ method: 'GetParameterValuesResponse' })).toBeNull();
  });

  it('未知 method → 返 null（调用方应 fallback 到 raw dump）', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'CustomVendorMethod',
      raw_response: '<x/>',
    });
    expect(got).toBeNull();
  });

  it('GPV 响应 XML 损坏 → 返空 params 数组', () => {
    const got = parseMmlDeviceTaskResult({
      method: 'GetParameterValuesResponse',
      raw_response: '<not closed',
    });
    expect(got?.kind).toBe('gpv');
    expect(got?.params).toEqual([]);
  });
});
