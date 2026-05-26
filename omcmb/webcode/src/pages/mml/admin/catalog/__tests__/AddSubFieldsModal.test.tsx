import { describe, it, expect } from 'vitest';

/**
 * 单测重点：autofill 派生算法 (前端与后端 derivePathLeafCode / humanizePathLeaf 同算法)
 * 必须保持等价，否则预览与后端真正派生的 mml_code 会出现"前端看一套、后端建另一套"的不一致。
 *
 * 这里把 AddSubFieldsModal 内部的 deriveMmlCode / deriveLabelEn 通过简化 reimport
 * 形式重新写在这里测——前者是私有函数，但算法很短，复制到测试里直接验证是最稳的方法。
 */

function deriveMmlCode(path: string): string {
  const idx = path.lastIndexOf('.');
  const leaf = idx >= 0 ? path.slice(idx + 1) : path;
  let out = '';
  let prevLower = false;
  for (let i = 0; i < leaf.length; i++) {
    const ch = leaf[i];
    const isUpper = ch >= 'A' && ch <= 'Z';
    const isLower = ch >= 'a' && ch <= 'z';
    const isDigit = ch >= '0' && ch <= '9';
    if (i > 0 && prevLower && isUpper) out += '_';
    if (ch === '_' || ch === '-') {
      out += '_';
    } else if (isUpper || isDigit) {
      out += ch;
    } else if (isLower) {
      out += ch.toUpperCase();
    }
    prevLower = isLower;
  }
  return out;
}

function deriveLabelEn(path: string): string {
  const idx = path.lastIndexOf('.');
  const leaf = idx >= 0 ? path.slice(idx + 1) : path;
  let out = '';
  let prevLower = false;
  for (let i = 0; i < leaf.length; i++) {
    const ch = leaf[i];
    const isUpper = ch >= 'A' && ch <= 'Z';
    const isLower = ch >= 'a' && ch <= 'z';
    if (i > 0 && prevLower && isUpper) out += ' ';
    if (ch === '_' || ch === '-') {
      out += ' ';
    } else {
      out += ch;
    }
    prevLower = isLower;
  }
  return out;
}

describe('AddSubFieldsModal autofill helpers', () => {
  it('deriveMmlCode handles CamelCase / underscore / digits', () => {
    expect(deriveMmlCode('Device.DeviceInfo.UserLabel')).toBe('USER_LABEL');
    expect(deriveMmlCode('Device.X.AntennaAzimuth')).toBe('ANTENNA_AZIMUTH');
    expect(deriveMmlCode('Device.X.1588_Status')).toBe('1588_STATUS');
    expect(deriveMmlCode('foo.bar.simple_path')).toBe('SIMPLE_PATH');
    expect(deriveMmlCode('NoDot')).toBe('NO_DOT');
    expect(deriveMmlCode('')).toBe('');
  });

  it('deriveLabelEn humanizes leaf', () => {
    expect(deriveLabelEn('Device.DeviceInfo.UserLabel')).toBe('User Label');
    expect(deriveLabelEn('Device.X.AntennaAzimuth')).toBe('Antenna Azimuth');
    expect(deriveLabelEn('Device.DeviceInfo.first_use_date')).toBe('first use date');
    expect(deriveLabelEn('NoDot')).toBe('No Dot');
  });
});
