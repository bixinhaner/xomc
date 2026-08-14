import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { describe, expect, it } from 'vitest';

/**
 * Issue #317 回归:快速设置"参数未修改点保存"不得下发。
 * 根因是 CellParameterForm 表单初值经 normalizeEnumValue 归一化
 * (设备 BOOLEAN "true"/"false" → 选项值 "1"/"0"),
 * 而脏检查 oldVal 若用原始设备值,两者不在同一取值空间 → 永远判脏 → 无效提交。
 * 本测试用源码边界断言防止脏检查退回原始值比较。
 */
describe('quick settings unmodified-save dirty check (issue #317)', () => {
  const source = readFileSync(resolve(
    process.cwd(),
    'src/pages/device/DeviceDetail/QuickSettingsTab/CellParameterForm.tsx',
  ), 'utf8');

  it('compares oldVal through normalizeEnumValue with the same options as form init', () => {
    expect(source).toContain(': normalizeEnumValue(');
    expect(source).toContain('isDeviceTimeGroup && p.name === \'Enable\' ? deviceTimeModeOptions : p.enumOptions');
  });

  it('imports normalizeEnumValue from the shared validators module', () => {
    expect(source).toMatch(/import\s*\{[^}]*normalizeEnumValue[^}]*\}\s*from\s*'\.\/validators'/s);
    // 本地不得再私有定义一份,避免初值/脏检查双源漂移。
    expect(source).not.toContain('function normalizeEnumValue');
  });
});
