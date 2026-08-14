/**
 * deviceApi 契约测试（#22 关键 API）：
 *   - getList：前端 filter → 后端 query 映射（CSV 多选、is_online 字符串/布尔双源、
 *     legacy networkType eNB/gNB → lte/nr），BackendDevice → Device 映射 + stats 优先后端。
 *   - getById：成功映射；错误（404/500）吞错返 null（详情页不崩，符合现行 catch 兜底）。
 *   - getStats / getProductClasses：端点透传。
 *   - mapListResponse 在 stats 缺省时用当前页 items 估算（向下兼容兜底）。
 *
 * 用 vi.mock 替换 http 客户端（参照 alarmApi.test.ts 既有模式）。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { UNASSIGNED_GROUP_ID } from '../../../utils/deviceGroupTargets';

const { getMock, postMock, putMock } = vi.hoisted(() => ({
  getMock: vi.fn(),
  postMock: vi.fn(),
  putMock: vi.fn(),
}));
vi.mock('../../http', () => ({
  default: { get: getMock, post: postMock, put: putMock, patch: vi.fn(), delete: vi.fn() },
}));

import { deviceApi } from '../deviceApi';

function backendDevice(overrides: Record<string, unknown> = {}) {
  return {
    id: 'd1',
    serial_number: 'SN001',
    oui: 'AABBCC',
    product_class: 'PC100',
    manufacturer: 'Acme',
    model_name: 'M1',
    carrier: 'cmcc',
    technology: 'nr',
    status: 'active',
    is_online: true,
    firmware_version: 'v1.0',
    ip_address: '10.0.0.1',
    group_id: 'group-1',
    group_name: '默认设备组',
    source_type: 'manual',
    connection_request_url: '',
    inform_interval: 300,
    device_name: '基站A',
    site_id: 'S1',
    latitude: 30,
    longitude: 120,
    created_at: '2026-01-01T00:00:00Z',
    updated_at: '2026-01-02T00:00:00Z',
    ...overrides,
  };
}

beforeEach(() => {
  getMock.mockReset();
  postMock.mockReset();
  putMock.mockReset();
  getMock.mockResolvedValue({ data: { items: [], total: 0, page: 1, page_size: 20, total_pages: 0 } });
});

describe('deviceApi.getList — filter → query 映射', () => {
	it('映射 OMC 管控摘要并提交来源/阶段筛选', async () => {
		getMock.mockResolvedValue({
			data: {
				items: [backendDevice({
					control_summary: {
						source_type: 'geofence',
						source_id: 'fence-1',
						source_name: '园区围栏',
						reason_code: 'confirmed_exit',
						phase: 'deactivated',
						action_id: 'action-1',
						triggered_at: '2026-08-13T08:00:00+08:00',
					},
				})],
				total: 1, page: 1, page_size: 20, total_pages: 1,
			},
		});

		const out = await deviceApi.getList({
			page: 1,
			pageSize: 20,
			controlSource: 'geofence',
			controlPhase: ['deactivated', 'recovery_failed'],
		});

		expect(getMock.mock.calls[0][1].params).toMatchObject({
			control_source: 'geofence',
			control_phase: 'deactivated,recovery_failed',
		});
		expect(out.items[0].controlSummary).toMatchObject({
			sourceType: 'geofence', sourceName: '园区围栏', phase: 'deactivated',
		});
	});
	it('映射 location_sync 对账状态及设备上报坐标', async () => {
		getMock.mockResolvedValue({
			data: {
				items: [backendDevice({
					location_sync: {
						status: 'pending',
						accepted: { latitude: 30, longitude: 120 },
						reported: {
							latitude: 30.001,
							longitude: 120.001,
							observed_at: '2026-07-17T02:00:00Z',
							version: 8,
							source_path: 'Device.DeviceInfo.SAS.FAP.GPS',
						},
						distance_meters: 146,
					},
				})],
				total: 1,
				page: 1,
				page_size: 20,
				total_pages: 1,
			},
		});

		const out = await deviceApi.getList({ page: 1, pageSize: 20 });
		expect(out.items[0].locationSourceMode).toBe('tr069');
		expect(out.items[0].locationSync.status).toBe('pending');
		expect(out.items[0].locationSync.reported?.version).toBe(8);
		expect(out.items[0].locationSync.distanceMeters).toBe(146);
	});

	it('映射并提交设备位置来源模式', async () => {
		getMock.mockResolvedValueOnce({ data: backendDevice({ location_source_mode: 'external' }) });
		const device = await deviceApi.getById('d1');
		expect(device?.locationSourceMode).toBe('external');

		putMock.mockResolvedValueOnce({ data: backendDevice({ location_source_mode: 'external' }) });
		getMock.mockResolvedValueOnce({ data: backendDevice({ location_source_mode: 'external' }) });
		await deviceApi.update('d1', { locationSourceMode: 'external' });
		expect(putMock).toHaveBeenCalledWith('/devices/d1', {
			location_source_mode: 'external',
		});
	});

	it('确认 GPS 同步提交 reported_version', async () => {
		postMock.mockResolvedValue({ data: { status: 'in_sync' } });
		const result = await deviceApi.acceptLocationSync('d1', 8);
		expect(result.status).toBe('in_sync');
		expect(postMock).toHaveBeenCalledWith('/devices/d1/location-sync/accept', { reported_version: 8 });
	});

  it('分页和排序字段使用后端 snake_case 契约', async () => {
    await deviceApi.getList({
      page: 2,
      pageSize: 50,
      sortField: 'serial_number',
      sortOrder: 'ascend',
    });
    const [, opts] = getMock.mock.calls[0];
    expect(opts.params).toMatchObject({
      page: 2,
      page_size: 50,
      sort_by: 'serial_number',
      sort_dir: 'asc',
    });
    expect(opts.params.pageSize).toBeUndefined();
    expect(opts.params.sortField).toBeUndefined();
    expect(opts.params.sortOrder).toBeUndefined();
  });

  it('多选字段序列化为 CSV（避免 axios key[] 形态被 gin 静默丢弃）', async () => {
    await deviceApi.getList({
      page: 1,
      pageSize: 20,
      snList: ['SN1', 'SN2'],
      lifecycleState: ['commissioned', 'maintenance'],
      modelName: 'M1',
    });
    const [url, opts] = getMock.mock.calls[0];
    expect(url).toBe('/devices');
    expect(opts.params.sn_list).toBe('SN1,SN2');
    expect(opts.params.lifecycle_state).toBe('commissioned,maintenance');
    expect(opts.params.model_name).toBe('M1');
  });

  it('legacy networkType eNB/gNB → technology lte/nr 翻译', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, networkType: 'gNB' });
    expect(getMock.mock.calls[0][1].params.technology).toBe('nr');
  });

  it('networkType GSM/gsm 都归一为 technology gsm', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, networkType: 'GSM' });
    expect(getMock.mock.calls[0][1].params.technology).toBe('gsm');

    await deviceApi.getList({ page: 1, pageSize: 20, networkType: 'gsm' });
    expect(getMock.mock.calls[1][1].params.technology).toBe('gsm');
  });

  it('isOnline 字符串 "true"（来自字典/表单）也映射为 is_online=true', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, isOnline: 'true' as unknown as boolean });
    expect(getMock.mock.calls[0][1].params.is_online).toBe('true');
  });

  it('isOnline 布尔 false（程序化调用）映射为 is_online=false', async () => {
    await deviceApi.getList({ page: 1, pageSize: 20, isOnline: false });
    expect(getMock.mock.calls[0][1].params.is_online).toBe('false');
  });

  it('BackendDevice → Device 映射 + 后端 stats 优先', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice()],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
        stats: {
          total: 1,
          online_count: 1,
          offline_count: 0,
          current_ue_count: 4,
          alarmed: 0,
        },
      },
    });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items).toHaveLength(1);
    const d = out.items[0];
    expect(d.sn).toBe('SN001');
    expect(d.name).toBe('基站A');
    expect(d.networkType).toBe('gNB'); // nr → gNB
    expect(d.connStatus).toBe('online'); // is_online → online
    expect(d.isOnline).toBe(true);
    expect(d.groupId).toBe('group-1');
    expect(d.groupName).toBe('默认设备组');
    expect(d.sourceType).toBe('manual');
    // 后端 stats 直读（不靠 items.filter 估算）
    expect(out.stats.online_count).toBe(1);
    expect(out.stats.current_ue_count).toBe(4);
  });

  it('按制式映射核心网状态，非本制式字段保持为空', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          backendDevice({ id: 'lte', technology: 'lte', mme_status: 'connected', amf_status: 'wrong', bsc_link_status: 'wrong' }),
          backendDevice({ id: 'nr', technology: 'nr', mme_status: 'connected' }),
          backendDevice({ id: 'gsm', technology: 'gsm', mme_status: 'disconnected', bsc_link_status: 'connected' }),
        ],
        total: 3,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    const [lte, nr, gsm] = out.items;
    expect(lte.mmeStatus).toBe('connected');
    expect(lte.amfStatus).toBe('');
    expect(nr.mmeStatus).toBe('');
    expect(nr.amfStatus).toBe('connected');
    expect(gsm.mmeStatus).toBe('');
    expect(gsm.amfStatus).toBe('');
    expect(gsm.bscLinkStatus).toBe('connected');
  });

  it('BackendDevice source_type=auto 映射为 Device.sourceType', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ group_id: undefined, group_name: undefined, source_type: 'auto' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items[0].groupId).toBeUndefined();
    expect(out.items[0].groupName).toBe('');
    expect(out.items[0].sourceType).toBe('auto');
  });

  it('映射最近一次参数同步完成时间，供设备列表展示周期同步结果', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ last_param_sync_at: '2026-07-24T07:30:00Z' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items[0].lastParamSyncAt).toBe('2026-07-24T07:30:00Z');
  });

  it('列表接口返回 device_address 时也能映射成 installAddress', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ install_address: undefined, device_address: '南京市雨花台区软件大道 1 号' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items[0].installAddress).toBe('南京市雨花台区软件大道 1 号');
  });

  it('映射后的 Device 不含 platformType（#177：该字段及关联逻辑已移除）', async () => {
    getMock.mockResolvedValue({
      data: {
        // 即便后端仍回 platform_type，映射也不再透传到前端 Device
        items: [backendDevice({ platform_type: 'CPE_436Q' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    const d = out.items[0];
    expect('platformType' in d).toBe(false);
    // 其它正常字段仍在，证明不是整体映射坏掉
    expect(d.sn).toBe('SN001');
    expect(d.productClass).toBe('PC100');
  });

  it('transmit_power=-1 视为占位值，不映射成列表 Tx Power', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ transmit_power: -1, tx_power: '' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items[0].txPower).toBe('');
  });

  it('保留 LTE ReferenceSignalPower 的负 dBm 实际值', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ transmit_power: -21, tx_power: '' })],
        total: 1,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });

    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items[0].txPower).toBe('-21');
  });

  it('stats 缺省时用当前页 items 估算（向下兼容兜底）', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [backendDevice({ is_online: true }), backendDevice({ id: 'd2', is_online: false })],
        total: 2,
        page: 1,
        page_size: 20,
        total_pages: 1,
      },
    });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.stats.online_count).toBe(1);
    expect(out.stats.offline_count).toBe(1);
  });

  it('items 缺失（null）不崩，返空列表', async () => {
    getMock.mockResolvedValue({ data: { total: 0, page: 1, page_size: 20, total_pages: 0 } });
    const out = await deviceApi.getList({ page: 1, pageSize: 20 });
    expect(out.items).toEqual([]);
  });
});

describe('deviceApi.getControlActions', () => {
	it('映射动作、参数状态和围栏评估证据', async () => {
		getMock.mockResolvedValue({
			data: {
				items: [{
					id: 'action-1', source_type: 'geofence', source_name: '园区围栏',
					reason_code: 'confirmed_exit', effective_state_version: 3,
					action_type: 'deactivate', status: 'verified',
					before_state: [{ path: 'Device.Cell.AdminState', value: '1' }],
					requested_state: [{ path: 'Device.Cell.AdminState', value: '0' }],
					verified_state: [{ path: 'Device.Cell.AdminState', value: '0' }],
					evaluation: {
						id: 'eval-1', observation_version: 9, latitude: 30, longitude: 120,
						observed_at: '2026-08-13T08:00:00+08:00', rule_type: 'polygon_allow_zone',
						signed_distance_meters: 12.4, confirmed_state: 'outside', reason_code: 'confirmed_exit',
					},
					created_at: '2026-08-13T08:00:01+08:00', updated_at: '2026-08-13T08:00:02+08:00',
				}],
				total: 1, page: 1, page_size: 20,
			},
		});

		const out = await deviceApi.getControlActions('device-1');

		expect(getMock).toHaveBeenCalledWith('/devices/device-1/control-actions', {
			params: { page: 1, page_size: 20 },
		});
		expect(out.items[0].beforeState[0]).toEqual({ path: 'Device.Cell.AdminState', value: '1' });
		expect(out.items[0].evaluation).toMatchObject({ observationVersion: 9, confirmedState: 'outside' });
	});
});

describe('deviceApi.getById', () => {
  it('成功：映射设备名空时回退 SN（friendlyName）', async () => {
    getMock.mockResolvedValue({ data: backendDevice({ device_name: '' }) });
    const d = await deviceApi.getById('d1');
    expect(d?.name).toBe('SN001'); // device_name 空 → SN
  });

  it('lifecycle_state 缺省时从 legacy status 派生（discovered 直传，active→commissioned）', async () => {
    getMock.mockResolvedValue({ data: backendDevice({ lifecycle_state: undefined, status: 'discovered', is_online: undefined }) });
    const d = await deviceApi.getById('d1');
    expect(d?.lifecycleState).toBe('discovered');
  });

  it('404/500 错误吞错返 null（详情页不崩，现行 catch 兜底）', async () => {
    getMock.mockRejectedValue({ response: { status: 404 } });
    const d = await deviceApi.getById('missing');
    expect(d).toBeNull();
  });
});

describe('deviceApi.getBySn', () => {
  it('详情接口字段为空时保留列表接口解析出的设备分组和小区字段', async () => {
    getMock
      .mockResolvedValueOnce({
        data: {
          items: [backendDevice({
            group_id: 'group-1',
            group_name: '默认设备组',
            pci: '425',
            freq_point: '1498',
            bandwidth: 20,
            band: '3',
            admin_state: 'true',
          })],
          total: 1,
          page: 1,
          page_size: 1,
          total_pages: 1,
        },
      })
      .mockResolvedValueOnce({
        data: backendDevice({
          group_id: undefined,
          group_name: '',
          pci: '',
          freq_point: '',
          bandwidth: '',
          band: '',
          admin_state: '',
        }),
      });

    const d = await deviceApi.getBySn('SN001');

    expect(getMock).toHaveBeenNthCalledWith(1, '/devices', {
      params: expect.objectContaining({ sn: 'SN001', page: 1, page_size: 1 }),
    });
    expect(getMock).toHaveBeenNthCalledWith(2, '/devices/d1');
    expect(d?.groupId).toBe('group-1');
    expect(d?.groupName).toBe('默认设备组');
    expect(d?.pci).toBe('425');
    expect(d?.dlEarfcn).toBe('1498');
    expect(d?.bandwidth).toBe(20);
    expect(d?.band).toBe('3');
    expect(d?.adminState).toBe('true');
  });
});

describe('deviceApi.update', () => {
  it('安装详细地址与备注走 /devices/:id/info，并在更新后回读设备', async () => {
    putMock.mockResolvedValue({ data: null });
    getMock.mockResolvedValue({
      data: backendDevice({
        install_address: '南京市雨花台区软件大道 1 号',
        remark: '现场已核对',
      }),
    });

    const out = await deviceApi.update('d1', {
      installAddress: '南京市雨花台区软件大道 1 号',
      remark: '现场已核对',
    });

    expect(putMock).toHaveBeenCalledTimes(1);
    expect(putMock).toHaveBeenCalledWith('/devices/d1/info', {
      address: '南京市雨花台区软件大道 1 号',
      remark: '现场已核对',
    });
    expect(getMock).toHaveBeenCalledWith('/devices/d1');
    expect(out.installAddress).toBe('南京市雨花台区软件大道 1 号');
    expect(out.remark).toBe('现场已核对');
  });

  it('写成功但回读失败时，使用 fallbackDevice 与当前变更降级返回，不误报失败', async () => {
    putMock.mockResolvedValue({ data: null });
    getMock.mockRejectedValueOnce(new Error('temporary readback failure'));

    const out = await deviceApi.update(
      'd1',
      { installAddress: '南京市雨花台区软件大道 2 号' },
      { id: 'd1', sn: 'SN001', installAddress: '', remark: '旧备注' }
    );

    expect(putMock).toHaveBeenCalledWith('/devices/d1/info', {
      address: '南京市雨花台区软件大道 2 号',
    });
    expect(out.id).toBe('d1');
    expect(out.sn).toBe('SN001');
    expect(out.installAddress).toBe('南京市雨花台区软件大道 2 号');
    expect(out.remark).toBe('旧备注');
  });
});

describe('deviceApi.renameDevice', () => {
  it('maps backend task_id to optional taskId for device-side rename progress', async () => {
    postMock.mockResolvedValue({ data: { task_id: 'task-rename-1' } });

    const out = await deviceApi.renameDevice('d1', '新基站名');

    expect(postMock).toHaveBeenCalledWith('/devices/d1/rename', { name: '新基站名' });
    expect(out).toEqual({ taskId: 'task-rename-1' });
  });

  it('keeps successful rename without task_id as success without fake progress', async () => {
    postMock.mockResolvedValue({ data: { message: 'renamed' } });

    const out = await deviceApi.renameDevice('d1', '仅网管侧改名');

    expect(out).toEqual({});
  });
});

describe('deviceApi.getGroups', () => {
  it('保持后端返回的分组名称与 i18n 原样透传', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'root-group',
            name: '默认设备组',
            parent_id: null,
            device_count: 0,
            description: '',
            remark: '',
            is_default: true,
            level: 1,
            children: [
              {
                id: UNASSIGNED_GROUP_ID,
                name: '默认设备组',
                parent_id: 'root-group',
                device_count: 6,
                description: '',
                remark: '',
                is_default: true,
                level: 2,
                name_i18n: { 'zh-CN': '默认设备组', 'en-US': 'Default Group' },
                source_group_id: 'source-l2',
                matching_mode: 'serialNumber',
                serial_number_list: ['SN-001', 'SN-002'],
              },
            ],
          },
        ],
        stats: { grouped_devices: 0, ungrouped_devices: 6 },
      },
    });

    const out = await deviceApi.getGroups();
    const group = out.groups.find((item) => item.id === UNASSIGNED_GROUP_ID);

    expect(group?.name).toBe('默认设备组');
    expect(group?.nameI18n).toEqual({ 'zh-CN': '默认设备组', 'en-US': 'Default Group' });
    expect(group?.sourceGroupId).toBe('source-l2');
    expect(group?.matchingMode).toBe('serialNumber');
    expect(group?.serialNumberList).toEqual(['SN-001', 'SN-002']);
    expect(out.stats.totalDevices).toBe(6);
  });

  it('将省略 parent_id 的根分组归一为 null，避免被当成二级源分组', async () => {
    getMock.mockResolvedValue({
      data: {
        items: [
          {
            id: 'root-group',
            name: '默认设备组',
            device_count: 0,
            description: '',
            remark: '',
            is_default: true,
            level: 1,
            children: [
              {
                id: UNASSIGNED_GROUP_ID,
                name: '默认设备组',
                parent_id: 'root-group',
                device_count: 6,
                description: '',
                remark: '',
                is_default: true,
                level: 2,
              },
            ],
          },
        ],
      },
    });

    const out = await deviceApi.getGroups();

    expect(out.groups.find((item) => item.id === 'root-group')?.parentId).toBeNull();
    expect(out.groups.find((item) => item.id === UNASSIGNED_GROUP_ID)?.parentId).toBe('root-group');
  });
});

describe('deviceApi.getStats / getProductClasses — 端点透传', () => {
  it('getStats 打 /devices/stats', async () => {
    getMock.mockResolvedValue({ data: { counts: { total: 5, online: 3, offline: 2, alarm: 0 } } });
    const s = await deviceApi.getStats();
    expect(getMock.mock.calls[0][0]).toBe('/devices/stats');
    expect(s.counts.total).toBe(5);
  });

  it('getProductClasses 打 /devices/product-classes', async () => {
    getMock.mockResolvedValue({ data: ['PC100', 'PC200'] });
    const list = await deviceApi.getProductClasses();
    expect(getMock.mock.calls[0][0]).toBe('/devices/product-classes');
    expect(list).toEqual(['PC100', 'PC200']);
  });
});

describe('deviceApi.updateAntennaSectorPlan', () => {
  it('保存五项本地规划参数并映射响应', async () => {
    putMock.mockResolvedValue({
      data: {
        number: 1,
        azimuth: 0,
        antenna_height: 18,
        mechanical_downtilt: 6,
        horizontal_beamwidth: 65,
        vertical_beamwidth: 8,
        field_sources: { azimuth: 'planned' },
        direction_available: true,
        coverage_available: true,
        coverage_status: 'available',
        missing_fields: [],
      },
    });

    const result = await deviceApi.updateAntennaSectorPlan('d1', 1, {
      azimuth: 0,
      antennaHeight: 18,
      mechanicalDowntilt: 6,
      horizontalBeamwidth: 65,
      verticalBeamwidth: 8,
    });

    expect(putMock).toHaveBeenCalledWith('/devices/d1/antenna-sectors/1', {
      azimuth: 0,
      antenna_height: 18,
      mechanical_downtilt: 6,
      horizontal_beamwidth: 65,
      vertical_beamwidth: 8,
    });
    expect(result.azimuth).toBe(0);
    expect(result.coverageAvailable).toBe(true);
    expect(result.coverageStatus).toBe('available');
  });
});
