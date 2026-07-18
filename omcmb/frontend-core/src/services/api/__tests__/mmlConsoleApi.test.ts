import { beforeEach, describe, expect, it, vi } from 'vitest';

const { getMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
}));

vi.mock('../../http', () => ({
  default: { get: getMock },
}));

import { mmlApi } from '../mmlApi';

beforeEach(() => {
  getMock.mockReset();
});

describe('mmlApi console product filtering', () => {
  it('passes the selected device to the command tree request', async () => {
    getMock.mockResolvedValue({ data: { tree: [] } });

    await mmlApi.buildGroupTree(undefined, 'zh-CN', undefined, 'SN-MLQ-1');

    expect(getMock).toHaveBeenCalledWith('/mml/group-tree', {
      params: { lang: 'zh-CN', device_sn: 'SN-MLQ-1' },
    });
  });

  it('passes the selected device to the command sub-fields request', async () => {
    getMock.mockResolvedValue({ data: { sub_fields: [] } });

    await mmlApi.getCommandSubFields('command-1', 'zh-CN', 'SN-MLQ-1');

    expect(getMock).toHaveBeenCalledWith('/mml/commands/command-1/sub-fields', {
      params: { lang: 'zh-CN', device_sn: 'SN-MLQ-1' },
    });
  });

  it('maps standard min and max values from command sub-fields', async () => {
    getMock.mockResolvedValue({ data: { sub_fields: [{
      id: 'sf-1',
      command_id: 'command-1',
      param_id: 'param-1',
      mml_code: 'CHANNEL',
      label: 'Channel',
      label_i18n: {},
      tr069_path: 'Device.Radio.Channel',
      value_type: 'unsignedInt',
      access_type: 'READ_WRITE',
      is_object: false,
      supports_add: false,
      supports_delete: false,
      change_applies: 'Immediate',
      constraint_text: '[1, 13]',
      constraint_text_i18n: {},
      min_value: 1,
      max_value: 13,
      default_selected: false,
      is_required: false,
      sort_order: 1,
    }] } });

    const result = await mmlApi.getCommandSubFields('command-1');

    expect(result[0]).toMatchObject({
      valueType: 'unsignedInt',
      minValue: 1,
      maxValue: 13,
    });
  });

  it('loads and maps enriched custom command paths', async () => {
    getMock.mockResolvedValue({ data: { items: [{
      id: 'path-1',
      command_id: 'custom-1',
      standard_path_id: 'standard-1',
      standard_path: 'Device.Info.Name',
      entry_type: 'parameter',
      access: 'readWrite',
      data_type: 'string',
      description: 'Name',
      min_value: 2,
      max_value: 32,
      default_selected: false,
      sort_order: 1,
      mutable: false,
    }] } });

    const result = await mmlApi.getTemplatePaths('custom-1');

    expect(getMock).toHaveBeenCalledWith('/mml/templates/custom-1/paths');
    expect(result[0]).toMatchObject({
      standardPath: 'Device.Info.Name',
      dataType: 'string',
      minValue: 2,
      maxValue: 32,
      mutable: false,
    });
  });
});
