import http from '../http';
import type { QuickSettingsGroupsResponse, TechCode } from '../../types/quicksettings';

/**
 * 快速设置分组元数据 API（T-0138）。
 *
 * 后端模块：omcgo/internal/quicksettings/
 * 端点：GET /api/v1/quicksettings/groups?tech={lte|nr}
 *
 * 字段中英文 label 来自后端 XML（titleZh / titleEn），不进 i18n 文件。
 * 字段元属性（类型 / 约束 / 枚举）由 useParameterSchema 拉，与本接口分离。
 */
export const quicksettingsApi = {
  async getGroups(tech: TechCode): Promise<QuickSettingsGroupsResponse> {
    const { data } = await http.get<QuickSettingsGroupsResponse>(
      '/quicksettings/groups',
      { params: { tech } },
    );
    return data;
  },
};
