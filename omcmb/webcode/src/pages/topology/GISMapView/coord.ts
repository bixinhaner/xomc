/**
 * GIS 地图坐标有效性判断（独立模块，便于单测）。
 *
 * 无效坐标包含两类（均会导致地图无法定位到真实位置）：
 * 1. null / undefined：后端未回填坐标。
 * 2. (0, 0)：数据质量缺省值，落点在几内亚湾外海空白网格，等同于"定位不到"。
 *
 * 见 issue #192：用户反馈"搜得到却定位不了、停在空白网格"，根因即点击此类
 * 无效坐标设备时地图飞到 (0,0) 海面或被静默拦截，无任何反馈。
 */
export function hasValidCoord(latitude?: number | null, longitude?: number | null): boolean {
  if (latitude == null || longitude == null) return false;
  if (latitude === 0 && longitude === 0) return false;
  return true;
}
