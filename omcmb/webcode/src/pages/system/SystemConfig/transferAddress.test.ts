import { describe, expect, it } from 'vitest';
import {
  buildStandardBaseURL,
  isValidTransferHost,
  parseTransferAddress,
} from './transferAddress';

describe('parseTransferAddress', () => {
  it('识别空值和规范的 HTTP 8080 地址', () => {
    expect(parseTransferAddress('')).toEqual({
      kind: 'empty',
      mode: 'standard',
      raw: '',
      host: '',
    });
    expect(parseTransferAddress('http://172.24.224.251:8080')).toEqual({
      kind: 'standard',
      mode: 'standard',
      raw: 'http://172.24.224.251:8080',
      host: '172.24.224.251',
    });
    expect(parseTransferAddress('http://[fd00::10]:8080')).toEqual({
      kind: 'standard',
      mode: 'standard',
      raw: 'http://[fd00::10]:8080',
      host: 'fd00::10',
    });
  });

  it('按指定协议和端口识别规范的 HTTPS 8443 地址', () => {
    expect(parseTransferAddress('https://172.24.224.251:8443', {
      protocol: 'https',
      port: '8443',
    })).toEqual({
      kind: 'standard',
      mode: 'standard',
      raw: 'https://172.24.224.251:8443',
      host: '172.24.224.251',
    });
  });

  it('底层解析保留完整 URL 兼容，页面校验决定是否允许提交', () => {
    for (const raw of [
      'https://edge.example.com:9443/omc',
      'http://edge.example.com:9090',
      'http://edge.example.com:8080/',
    ]) {
      expect(parseTransferAddress(raw)).toEqual({
        kind: 'custom',
        mode: 'full',
        raw,
      });
    }
  });

  it.each([
    ' http://172.24.224.251:8080 ',
    'http:172.24.224.251:8080',
    'http:/172.24.224.251:8080',
    'http:////172.24.224.251:8080',
    String.raw`http:\172.24.224.251:8080`,
    'http://172.24.224.251:8080/.',
    'http://172.24.224.251:8080/a/..',
    'http://172.24.224.251:8080/%2e%2e',
    'http://172.24.224.251:8080/%5cadmin',
    'http://172.24.224.251:8080?',
    'http://172.24.224.251:8080#',
    'http://@172.24.224.251:8080',
    'http://172.24.224.251:99999',
  ])('拒绝浏览器可能自动修复或隐藏的原始 URL：%s', (raw) => {
    expect(parseTransferAddress(raw)).toMatchObject({
      kind: 'invalid',
      mode: 'full',
      raw,
    });
  });

  it.each([
    'http://127.1:8080',
    'http://2130706433:8080',
    'http://01.2.3.4:8080',
  ])('拒绝后端生产校验不接受的非规范数字主机：%s', (raw) => {
    expect(parseTransferAddress(raw)).toMatchObject({
      kind: 'invalid',
      raw,
    });
  });

  it('保留标准输入框中的主机草稿，但标记为非法', () => {
    expect(parseTransferAddress('http://172.:8080')).toEqual({
      kind: 'invalid',
      mode: 'standard',
      raw: 'http://172.:8080',
      host: '172.',
    });
  });
});

describe('standard transfer host', () => {
  it.each([
    ['172.24.224.251', true],
    ['edge.example.com', true],
    ['fd00::10', true],
    ['999.24.224.251', false],
    ['127.1', false],
    ['2130706433', false],
    ['edge.example.com/path', false],
  ])('校验主机 %s => %s', (host, expected) => {
    expect(isValidTransferHost(host)).toBe(expected);
  });

  it('构建 IPv4、域名和 IPv6 的标准地址', () => {
    expect(buildStandardBaseURL('172.24.224.251')).toBe('http://172.24.224.251:8080');
    expect(buildStandardBaseURL('edge.example.com')).toBe('http://edge.example.com:8080');
    expect(buildStandardBaseURL('fd00::10')).toBe('http://[fd00::10]:8080');
    expect(buildStandardBaseURL('172.24.224.251', {
      protocol: 'https',
      port: '8443',
    })).toBe('https://172.24.224.251:8443');
  });
});
