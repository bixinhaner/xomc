import { describe, it, expect } from 'vitest';
import { resolveBscBtsInstances, resolveExistsVisibleInstances } from '../useResolvedCellInstances';

// 与 hook 内一致的制式正则（LTE FAPService）。
const LTE_FAP_INUSE_RE = /^Device\.Services\.FAPService\.(\d+)\.FAPControl\.LTE\.InUse$/i;
const FAP_INSTANCE_RE = /^Device\.Services\.FAPService\.(\d+)\./;

type Param = { path: string; currentValue?: string | null };

// 构造一个 FAPService 实例的参数行：始终有一个 PCI 参数（代表「实例存在」），
// InUse 可选（undefined 代表缺参 / 未同步）。
function fapInstance(n: number, inUse?: string | null): Param[] {
  const rows: Param[] = [
    { path: `Device.Services.FAPService.${n}.CellConfig.LTE.RAN.Common.CellIdentity`, currentValue: `${n}` },
  ];
  if (inUse !== undefined) {
    rows.push({ path: `Device.Services.FAPService.${n}.FAPControl.LTE.InUse`, currentValue: inUse });
  }
  return rows;
}

describe('#374 resolveExistsVisibleInstances（存在即可见、仅显式 InUse 假值隐藏）', () => {
  it('3GMS+6LTE：第 6 个实例存在但缺 InUse 参数 → 不被静默隐藏（修第 6 小区丢失根因）', () => {
    const params: Param[] = [
      ...fapInstance(1, '1'),
      ...fapInstance(2, '1'),
      ...fapInstance(3, '1'),
      ...fapInstance(4, '1'),
      ...fapInstance(5, '1'),
      ...fapInstance(6, undefined), // 第 6 个：实例存在但 InUse 缺失
    ];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([1, 2, 3, 4, 5, 6]);
  });

  it('InUse 空串（未同步）也视为可见，不隐藏', () => {
    const params: Param[] = [...fapInstance(1, '1'), ...fapInstance(2, '')];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([1, 2]);
  });

  it('InUse 显式 0/false → 隐藏（设备侧真禁用）', () => {
    const params: Param[] = [
      ...fapInstance(1, '1'),
      ...fapInstance(2, '0'),
      ...fapInstance(3, 'false'),
    ];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([1]);
  });

  it('InUse=1 → 可见', () => {
    const params: Param[] = [...fapInstance(1, '1')];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([1]);
  });

  it('超过 limit 的实例被裁剪（cellModeIdx=1 → lteNum=6，第 7 个不计）', () => {
    const params: Param[] = [...fapInstance(6, '1'), ...fapInstance(7, '1')];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([6]);
  });

  it('objects.currentInstances 也算「存在」（schema 显式列出但参数行缺失时）', () => {
    const got = resolveExistsVisibleInstances([], [1, 2, 3], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([1, 2, 3]);
  });

  it('limit<=0（mode 命中但该制式槽位为 0）→ 空', () => {
    const params: Param[] = [...fapInstance(1, '1')];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 0);
    expect(got).toEqual([]);
  });

  it('混合：1=1(可见) 2=缺参(可见) 3=0(隐藏) 4=空(可见) → [1,2,4]', () => {
    const params: Param[] = [
      ...fapInstance(1, '1'),
      ...fapInstance(2, undefined),
      ...fapInstance(3, '0'),
      ...fapInstance(4, ''),
    ];
    const got = resolveExistsVisibleInstances(params, [], LTE_FAP_INUSE_RE, FAP_INSTANCE_RE, 6);
    expect(got).toEqual([1, 2, 4]);
  });
});

describe('resolveBscBtsInstances', () => {
  it('BSC 快速设置优先按实际参数路径枚举 BTS，不把 currentInstances 兜底 1..256 当真实数量', () => {
    const currentInstances = Array.from({ length: 256 }, (_, idx) => idx + 1);
    const got = resolveBscBtsInstances(
      [
        { path: 'DeviceGSM.Bts.1.IpaUnitId' },
        { path: 'DeviceGSM.Bts.1.CellId' },
        { path: 'DeviceGSM.Bts.0.CellId' },
      ],
      currentInstances,
    );

    expect(got).toEqual([1]);
  });

  it('没有任何 BTS 参数时才回退 currentInstances，并过滤 0 号槽位', () => {
    const got = resolveBscBtsInstances([], [0, 1, 2]);

    expect(got).toEqual([1, 2]);
  });
});
