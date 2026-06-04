import type { RawPathRow } from './types';

// 裸路径行工厂：单调递增 id 作为 React key（与列表增删稳定对应）。
let rowIdSeq = 1;

export const newRawPathRow = (path = '', value = ''): RawPathRow => ({
  id: rowIdSeq++,
  path,
  value,
});
