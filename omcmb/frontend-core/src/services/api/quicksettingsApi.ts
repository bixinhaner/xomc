import http from '../http';
import type { QuickSettingsGroupsResponse } from '../../types/quicksettings';

type QuickSettingsGroupsResponseRaw = {
  paramModel?: string;
  param_model?: string;
  groups?: QuickSettingsGroupsResponse['groups'];
};

/**
 * 快速设置分组元数据 API(T-0138)。
 *
 * 后端模块:omcgo/internal/quicksettings/
 * 端点:GET /api/v1/quicksettings/groups?device_id={uuid}
 *
 * 后端走 device→product→paramModel 链路解析,加载 data/quicksettings/<ParamModelName>.xml。
 * groups 为空数组表示该 paramModel 未配置 XML,前端应不显示「快速设置」tab。
 *
 * 字段中英文 label 来自后端 XML(titleZh / titleEn),不进 i18n 文件。
 * 字段元属性(类型 / 约束 / 枚举)由 useParameterSchema 拉,与本接口分离。
 *
 * 后端字段返回 snake_case `param_model`,此处需显式映射到 `paramModel`。
 * TODO: axios 拦截器本应全局 snake→camel转换但此处未生效，需追查为何该端点未被覆盖；还原后可刪除下方手工映射。
 */
export const quicksettingsApi = {
  async getGroups(deviceId: string): Promise<QuickSettingsGroupsResponse> {
    const { data } = await http.get<QuickSettingsGroupsResponseRaw>(
      '/quicksettings/groups',
      { params: { device_id: deviceId } },
    );
    return {
      paramModel: data.paramModel ?? data.param_model ?? '',
      groups: data.groups ?? [],
    };
  },
  async getGroupsByParamModel(paramModel: string): Promise<QuickSettingsGroupsResponse> {
    const { data } = await http.get<QuickSettingsGroupsResponseRaw>(
      '/quicksettings/groups',
      { params: { param_model: paramModel } },
    );
    return {
      paramModel: data.paramModel ?? data.param_model ?? '',
      groups: data.groups ?? [],
    };
  },
};
