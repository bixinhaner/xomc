import http from '../http';
import type { QuickSettingsGroupsResponse } from '../../types/quicksettings';

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
 * 后端字段返回 snake_case `param_model`,axios 拦截器自动转 camelCase `paramModel`。
 */
export const quicksettingsApi = {
  async getGroups(deviceId: string): Promise<QuickSettingsGroupsResponse> {
    const { data } = await http.get<QuickSettingsGroupsResponse>(
      '/quicksettings/groups',
      { params: { device_id: deviceId } },
    );
    return data;
  },
};
