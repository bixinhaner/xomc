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
