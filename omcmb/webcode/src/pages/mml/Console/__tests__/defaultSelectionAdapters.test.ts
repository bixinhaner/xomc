import { describe, expect, it } from 'vitest';
import type { MMLCustomCommandPathDef } from '@core/types/mml';
import type { SubFieldDef } from '@core/types/mmlConsole';
import {
  customCommandPathDefsToParamPaths,
  subFieldsToParamPaths,
} from '../adapters';

describe('MML default selection adapters', () => {
  it('preserves defaultSelected for a standard command sub-field', () => {
    const subFields = [{
      tr069Path: 'Device.Info.Name',
      label: 'Name',
      accessType: 'READ_WRITE',
      isObject: false,
      defaultSelected: true,
    }] as unknown as SubFieldDef[];

    expect(subFieldsToParamPaths(subFields)).toEqual([
      expect.objectContaining({
        path: 'Device.Info.Name',
        defaultSelected: true,
      }),
    ]);
  });

  it('preserves defaultSelected for an enriched custom command Path', () => {
    const paths: MMLCustomCommandPathDef[] = [{
      id: 'path-1',
      commandId: 'custom-1',
      standardPathId: 'standard-1',
      standardPath: 'Device.Info.Name',
      entryType: 'parameter',
      access: 'READ_WRITE',
      dataType: 'STRING',
      description: 'Name',
      defaultSelected: true,
      sortOrder: 10,
      mutable: true,
    }];

    expect(customCommandPathDefsToParamPaths(paths)).toEqual([
      expect.objectContaining({
        path: 'Device.Info.Name',
        defaultSelected: true,
      }),
    ]);
  });
});
