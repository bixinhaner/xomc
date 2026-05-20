import { describe, it, expect } from 'vitest';
import { statementToStructured } from '../mmlConsole';
import type { Statement, SubFieldDef } from '../mmlConsole';

function sf(opts: { id: string; mmlCode: string; tr069Path: string }): SubFieldDef {
  return {
    id: opts.id,
    commandId: 'cmd-1',
    paramId: 'p-' + opts.id,
    mmlCode: opts.mmlCode,
    label: opts.mmlCode,
    labelI18n: {},
    tr069Path: opts.tr069Path,
    valueType: 'string',
    accessType: 'READ_WRITE',
    isObject: false,
    supportsAdd: false,
    supportsDelete: false,
    changeApplies: 'Immediate',
    constraintText: '',
    constraintTextI18n: {},
    defaultSelected: false,
    isRequired: false,
    sortOrder: 0,
  };
}

function baseStmt(overrides: Partial<Statement>): Statement {
  return {
    uid: 'uid-1',
    commandId: 'cmd-1',
    commandCode: 'MOD_IP',
    logicalCode: 'IP',
    operationType: 'MOD',
    logicalNameI18n: {},
    subFields: [],
    selectedSubFieldIds: [],
    values: {},
    unknownCodes: [],
    ...overrides,
  };
}

describe('statementToStructured (R-9.2 前端切结构化通道)', () => {
  const subFields = [
    sf({ id: 'sf-addr', mmlCode: 'ADDR', tr069Path: 'Device.IP.Address' }),
    sf({ id: 'sf-mask', mmlCode: 'MASK', tr069Path: 'Device.IP.Netmask' }),
    sf({ id: 'sf-enbl', mmlCode: 'ENBL', tr069Path: 'Device.IP.Enable' }),
  ];

  it('LST: selectedSubFieldIds → paths (sf.id → tr069Path)', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
      selectedSubFieldIds: ['sf-addr', 'sf-enbl'],
    });
    const out = statementToStructured(stmt);
    expect(out.commandId).toBe('cmd-1');
    expect(out.operationType).toBe('LST');
    expect(out.paths).toEqual(['Device.IP.Address', 'Device.IP.Enable']);
    expect(out.values).toBeUndefined();
    expect(out.instanceIndices).toBeUndefined();
  });

  it('MOD: values key 由 mml_code 翻译为 standardPath', () => {
    const stmt = baseStmt({
      operationType: 'MOD',
      subFields,
      values: { ADDR: '192.168.1.1', MASK: '255.255.255.0' },
    });
    const out = statementToStructured(stmt);
    expect(out.values).toEqual({
      'Device.IP.Address': '192.168.1.1',
      'Device.IP.Netmask': '255.255.255.0',
    });
  });

  it('RMV: rmvInstanceIndices 直通 instanceIndices', () => {
    const stmt = baseStmt({
      operationType: 'RMV',
      subFields,
      rmvInstanceIndices: [1, 3, 5],
    });
    const out = statementToStructured(stmt);
    expect(out.instanceIndices).toEqual([1, 3, 5]);
  });

  it('RMV: 仅旧 rmvInstanceIndex → instanceIndices=[N]', () => {
    const stmt = baseStmt({
      operationType: 'RMV',
      subFields,
      rmvInstanceIndex: 7,
    });
    const out = statementToStructured(stmt);
    expect(out.instanceIndices).toEqual([7]);
  });

  it('RMV: indices 优先于单 Index（与后端 resolveRMVIndices 一致）', () => {
    const stmt = baseStmt({
      operationType: 'RMV',
      subFields,
      rmvInstanceIndex: 99,
      rmvInstanceIndices: [1, 2],
    });
    const out = statementToStructured(stmt);
    expect(out.instanceIndices).toEqual([1, 2]);
  });

  it('selectedSubFieldId 找不到对应 sf → 静默 skip（不报错）', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
      selectedSubFieldIds: ['sf-addr', 'sf-missing'],
    });
    const out = statementToStructured(stmt);
    expect(out.paths).toEqual(['Device.IP.Address']);
  });

  it('values key 找不到对应 mml_code → 静默 skip', () => {
    const stmt = baseStmt({
      operationType: 'MOD',
      subFields,
      values: { ADDR: '10.0.0.1', BOGUS: 'x' },
    });
    const out = statementToStructured(stmt);
    expect(out.values).toEqual({ 'Device.IP.Address': '10.0.0.1' });
  });

  it('subFields 缺 tr069Path → 静默 skip（数据异常防御）', () => {
    const broken = sf({ id: 'sf-broken', mmlCode: 'BRK', tr069Path: '' });
    const stmt = baseStmt({
      operationType: 'LST',
      subFields: [broken, ...subFields],
      selectedSubFieldIds: ['sf-broken', 'sf-addr'],
    });
    const out = statementToStructured(stmt);
    expect(out.paths).toEqual(['Device.IP.Address']);
  });

  it('空 selectedSubFieldIds + 空 values → paths=[] 但 commandId 保留', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
    });
    const out = statementToStructured(stmt);
    expect(out.paths).toEqual([]);
    expect(out.values).toBeUndefined();
    expect(out.commandId).toBe('cmd-1');
  });

  it('commandCode 透传到结构化入参', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
      commandCode: 'LST_DEVICE_INFO',
    });
    const out = statementToStructured(stmt);
    expect(out.commandCode).toBe('LST_DEVICE_INFO');
  });

  it('commandId 缺失（未挂载 catalog）→ 空字符串（后端 422 拦截）', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
      commandId: undefined,
    });
    const out = statementToStructured(stmt);
    expect(out.commandId).toBe('');
  });

  it('R-4: instanceSelectors 直通到结构化入参', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
      instanceSelectors: { iα: '1', iβ: '2' },
    });
    const out = statementToStructured(stmt);
    expect(out.instanceSelectors).toEqual({ iα: '1', iβ: '2' });
  });

  it('R-4: 空 instanceSelectors 不写入结构化入参（保持 payload 干净）', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
      instanceSelectors: {},
    });
    const out = statementToStructured(stmt);
    expect(out.instanceSelectors).toBeUndefined();
  });

  it('R-4: 无 instanceSelectors 字段 → 输出也不带', () => {
    const stmt = baseStmt({
      operationType: 'LST',
      subFields,
    });
    const out = statementToStructured(stmt);
    expect(out.instanceSelectors).toBeUndefined();
  });
});
