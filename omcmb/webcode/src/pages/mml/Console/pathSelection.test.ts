import { describe, expect, it } from 'vitest';
import type { CommandParamPath } from './types';
import {
  commandUsesPathSelection,
  areSelectedPathValuesComplete,
  getOrderedSelectedCommandPaths,
  getOrderedSelectedPathKeys,
  getDefaultSelectedPathKeys,
  getSelectableCommandPaths,
} from './pathSelection';

const paths: CommandParamPath[] = [
  { path: 'Device.Info.Serial', label: 'Serial', writable: false, isObject: false },
  { path: 'Device.Info.Name', label: 'Name', writable: true, isObject: false },
  { path: 'Device.Info.Mode', label: 'Mode', writable: true, isObject: false },
];

describe('MML command Path selection rules', () => {
  it('uses Path selection only for LST, DSP, and MOD', () => {
    expect(commandUsesPathSelection('LST')).toBe(true);
    expect(commandUsesPathSelection('DSP')).toBe(true);
    expect(commandUsesPathSelection('MOD')).toBe(true);
    expect(commandUsesPathSelection('ADD')).toBe(false);
    expect(commandUsesPathSelection('RMV')).toBe(false);
  });

  it('offers every executable Path for query commands', () => {
    expect(getSelectableCommandPaths('LST', paths)).toEqual(paths);
    expect(getSelectableCommandPaths('DSP', paths)).toEqual(paths);
  });

  it('offers only writable Paths for MOD', () => {
    expect(getSelectableCommandPaths('MOD', paths).map((path) => path.path)).toEqual([
      'Device.Info.Name',
      'Device.Info.Mode',
    ]);
  });

  it('does not introduce Path selection for ADD or RMV', () => {
    expect(getSelectableCommandPaths('ADD', paths)).toEqual([]);
    expect(getSelectableCommandPaths('RMV', paths)).toEqual([]);
  });
});

describe('selected Path projection', () => {
  it('keeps command-definition order and drops stale keys', () => {
    const selected = ['Device.Info.Mode', 'Device.Missing', 'Device.Info.Name'];
    expect(getOrderedSelectedCommandPaths(paths, selected).map((path) => path.path)).toEqual([
      'Device.Info.Name',
      'Device.Info.Mode',
    ]);
  });

  it('derives ordered key list from paths and selected keys', () => {
    const selected = ['Device.Info.Mode', 'Device.Missing', 'Device.Info.Name'];
    expect(getOrderedSelectedPathKeys(paths, selected)).toEqual([
      'Device.Info.Name',
      'Device.Info.Mode',
    ]);
  });

  it('requires a non-whitespace value for every selected MOD Path', () => {
    const selected = paths.slice(1);
    expect(
      areSelectedPathValuesComplete(selected, {
        'Device.Info.Name': 'cell-a',
        'Device.Info.Mode': '1',
      }),
    ).toBe(true);
    expect(
      areSelectedPathValuesComplete(selected, {
        'Device.Info.Name': 'cell-a',
        'Device.Info.Mode': '   ',
      }),
    ).toBe(false);
    expect(
      areSelectedPathValuesComplete(selected, {
        'Device.Info.Name': 'cell-a',
      }),
    ).toBe(false);
  });
});

describe('configured default Path projection', () => {
  it('keeps candidate order and ignores false or missing defaults', () => {
    const candidates: CommandParamPath[] = [
      {
        path: 'Device.Info.Serial',
        label: 'Serial',
        writable: false,
        isObject: false,
        defaultSelected: true,
      },
      {
        path: 'Device.Info.Name',
        label: 'Name',
        writable: true,
        isObject: false,
        defaultSelected: false,
      },
      {
        path: 'Device.Info.Model',
        label: 'Model',
        writable: true,
        isObject: false,
      },
      {
        path: 'Device.Info.Alias',
        label: 'Alias',
        writable: true,
        isObject: false,
        defaultSelected: true,
      },
    ];

    expect(getDefaultSelectedPathKeys(candidates)).toEqual([
      'Device.Info.Serial',
      'Device.Info.Alias',
    ]);
  });
});
