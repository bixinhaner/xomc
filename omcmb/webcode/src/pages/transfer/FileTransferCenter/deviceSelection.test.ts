import { describe, it, expect } from 'vitest';
import type { UnifiedFileTransferDeviceItem } from '@core/types/unifiedFileTransfer';
import {
  mergeCandidatesIntoMap,
  removeFromMap,
  pruneSelectionForProductClass,
  type SelectedDeviceMap,
} from './deviceSelection';

function dev(id: string, productType = 'CLASS_A'): UnifiedFileTransferDeviceItem {
  return {
    id,
    taskId: '',
    taskName: '',
    category: 'FIRMWARE_UPGRADE' as UnifiedFileTransferDeviceItem['category'],
    categoryLabel: '',
    typeCode: '',
    typeDisplayName: '',
    deviceName: `dev-${id}`,
    deviceSn: `SN-${id}`,
    productType,
    currentVersion: '',
    targetVersion: '',
    status: 'PENDING' as UnifiedFileTransferDeviceItem['status'],
  };
}

describe('mergeCandidatesIntoMap', () => {
  it('adds new candidates into the map (success path)', () => {
    const prev: SelectedDeviceMap = {};
    const next = mergeCandidatesIntoMap(prev, [dev('1'), dev('2')]);
    expect(Object.keys(next).sort()).toEqual(['1', '2']);
    expect(next).not.toBe(prev);
  });

  it('returns the SAME reference when nothing changes (no needless rerender)', () => {
    const a = dev('1');
    const prev: SelectedDeviceMap = { '1': a };
    const next = mergeCandidatesIntoMap(prev, [a]);
    expect(next).toBe(prev);
  });

  it('returns prev on empty candidate page (pagination edge)', () => {
    const prev: SelectedDeviceMap = { '1': dev('1') };
    expect(mergeCandidatesIntoMap(prev, [])).toBe(prev);
  });

  it('overwrites with the newer object reference for the same id', () => {
    const oldRef = dev('1', 'CLASS_A');
    const newRef = dev('1', 'CLASS_A');
    const next = mergeCandidatesIntoMap({ '1': oldRef }, [newRef]);
    expect(next['1']).toBe(newRef);
  });
});

describe('removeFromMap', () => {
  it('removes an existing id and returns a new map', () => {
    const prev: SelectedDeviceMap = { '1': dev('1'), '2': dev('2') };
    const next = removeFromMap(prev, '1');
    expect(Object.keys(next)).toEqual(['2']);
    expect(next).not.toBe(prev);
  });

  it('returns the same reference when id is absent (no-op)', () => {
    const prev: SelectedDeviceMap = { '1': dev('1') };
    expect(removeFromMap(prev, 'nope')).toBe(prev);
  });
});

describe('pruneSelectionForProductClass (#215 firmware-class regression guard)', () => {
  const map: SelectedDeviceMap = {
    a: dev('a', 'CLASS_A'),
    b: dev('b', 'CLASS_B'),
    c: dev('c', 'CLASS_A'),
  };

  it('drops selections whose productType differs from the chosen class', () => {
    const { kept, droppedCount } = pruneSelectionForProductClass(['a', 'b', 'c'], map, 'CLASS_A');
    expect(kept).toEqual(['a', 'c']);
    expect(droppedCount).toBe(1);
  });

  it('keeps everything and reports 0 dropped when class is empty/undefined', () => {
    expect(pruneSelectionForProductClass(['a', 'b'], map, undefined)).toEqual({
      kept: ['a', 'b'],
      droppedCount: 0,
    });
    expect(pruneSelectionForProductClass(['a', 'b'], map, '')).toEqual({
      kept: ['a', 'b'],
      droppedCount: 0,
    });
  });

  it('conservatively keeps ids unknown to the map or with empty productType', () => {
    const sparse: SelectedDeviceMap = { a: dev('a', 'CLASS_A'), x: dev('x', '') };
    const { kept, droppedCount } = pruneSelectionForProductClass(['a', 'x', 'missing'], sparse, 'CLASS_A');
    expect(kept).toEqual(['a', 'x', 'missing']);
    expect(droppedCount).toBe(0);
  });

  it('is pagination-safe: a matching id absent from the current page is still kept', () => {
    // 'c' belongs to CLASS_A but is not on the current candidate page; map still has it.
    const { kept } = pruneSelectionForProductClass(['c'], map, 'CLASS_A');
    expect(kept).toEqual(['c']);
  });
});
