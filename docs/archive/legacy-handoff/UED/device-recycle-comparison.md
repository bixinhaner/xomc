# 设备回收站页面分析 (deviceRecycleBin.jsp)

## 1. 页面概述

设备回收站页面用于管理已删除/移入回收站的设备，支持恢复设备或彻底删除。

## 2. 设备类型

| 类型 | 说明 | 权限码 |
|------|------|--------|
| eNB | 4G 基站 | CODE_ENB_DEVICE_REGISTER |
| gNB | 5G 基站 | CODE_GNB_DEVICE_REGISTER |
| CPE | 客户端设备 | CODE_CPE_DEVICE |

## 3. 功能列表

| 功能 | 说明 | API |
|------|------|-----|
| 设备列表 | 分页查询回收站设备 | 见下方 API 章节 |
| 搜索 | 模糊搜索设备编码/序列号/MAC | - |
| 设备组筛选 | 按设备组过滤 | - |
| 批量选择 | 多选设备进行批量操作 | - |
| 移出回收站 | 恢复设备到正常列表 | 见下方 API 章节 |
| 批量删除 | 彻底删除设备 | 见下方 API 章节 |
| 关闭 | 关闭回收站弹窗 | - |

## 4. API 接口

### 4.1 获取设备列表

| 设备类型 | API |
|----------|-----|
| eNB | `GET /recycle/getRecycleDeviceListByPage.action?isGnb=0` |
| gNB | `GET /recycle/getRecycleDeviceListByPage.action?isGnb=1` |
| CPE | `GET /recycle/getCpeRecycleDeviceListByPage.action` |

**请求参数：**
```typescript
interface RecycleListParams {
  searchText?: string;      // 搜索文本
  like_fields?: string;     // 模糊搜索字段 (eNB/gNB: serial_number, CPE: serial_number,macaddress)
  group_id?: string;        // 设备组ID
  timeZone?: string;        // 时区
}
```

### 4.2 移出回收站（恢复）

| 设备类型 | API | 参数名 |
|----------|-----|--------|
| eNB | `POST /recycle/moveDeviceToSmallCellInfos.action` | smallCellCodeStr |
| gNB | `POST /recycle/moveDeviceToSmallCellInfos.action?isGnb=1` | smallCellCodeStr |
| CPE | `POST /recycle/moveDeviceToCpeInfos.action` | cpeCodeStr |

**请求参数：**
```typescript
// eNB/gNB
interface RestoreEnbParams {
  smallCellCodeStr: string;  // 设备编码，逗号分隔
}

// CPE
interface RestoreCpeParams {
  cpeCodeStr: string;  // CPE编码，逗号分隔
}
```

### 4.3 批量删除

| 设备类型 | API | 参数名 |
|----------|-----|--------|
| eNB | `POST /system/deviceGroup/delCellinfo.action` | ids |
| gNB | `POST /system/deviceGroup/delCellinfo.action?isGnb=1` | ids |
| CPE | `POST /cell/CPE/delCpeinfo.action` | cpeCodes |

**请求参数：**
```typescript
// eNB/gNB
interface DeleteEnbParams {
  ids: string;        // 设备编码_产品类型，逗号分隔 (如: SN001_product1,SN002_product2)
  whereFrom: string;  // 固定值 'recycle'
}

// CPE
interface DeleteCpeParams {
  cpeCodes: string;   // CPE编码，逗号分隔
  whereFrom: string;  // 固定值 'recycle'
}
```

### 4.4 获取设备组选项

```
POST /system/deviceGroup/getSimpleDeviceGroupList.action
```

**请求参数：**
```typescript
interface DeviceGroupParams {
  isAll: string;  // '0' 表示不包含全部
}
```

**响应：**
```typescript
interface DeviceGroupOption {
  id: string;
  group_name: string;
}
```

## 5. 表格字段

### 5.1 eNB 专用字段

| 字段名 | 中文 | 说明 |
|--------|------|------|
| serial_number | 小站编码 | 设备序列号 |
| host_name | HostName | 主机名 |
| mac_address | MAC地址 | - |
| manufacturer | 厂商 | 扩展字段 |
| city | 城市 | 扩展字段 |
| sub_branches | 所属支局 | 扩展字段 |
| sub_station_name | 站点名称 | - |
| install_address | 安装详细地址 | 扩展字段(部分运营商) |
| contact_number | 业主联系方式 | 扩展字段(部分运营商) |
| site_id | Site ID | 扩展字段(部分运营商) |
| circuit_ref | Circuit Ref. | 扩展字段(部分运营商) |
| circuit_jo | Circuit J & O | 扩展字段(部分运营商) |
| service_status | Status | 扩展字段(部分运营商) |
| rom | Rom | 扩展字段(部分运营商) |
| longitude | 经度 | - |
| latitude | 纬度 | - |
| height | 高度 | - |

### 5.2 gNB 专用字段

| 字段名 | 中文 | 说明 |
|--------|------|------|
| serial_number | 小站编码 | 设备序列号 |
| mac_address | MAC地址 | - |
| longitude | 经度 | - |
| latitude | 纬度 | - |
| height | 高度 | - |

### 5.3 CPE 专用字段

| 字段名 | 中文 | 说明 |
|--------|------|------|
| serial_number | CPE序列号 | - |
| macaddress | MAC地址 | - |
| imsi | IMSI | - |
| longitude | 经度 | - |
| latitude | 纬度 | - |
| height | 高度 | - |
| distance | 距离 | - |

### 5.4 通用字段（所有设备类型）

| 字段名 | 中文 | 说明 |
|--------|------|------|
| offlineDays | 离线天数 | - |
| group_name | 设备组名称 | - |
| moveType | 回收方式 | 0: 自动, 1: 手动 |
| moveTime | 回收时间 | - |
| move_author | 账户 | 操作人 |
| product | 产品类型 | 用于删除时拼接ID |

## 6. 数据结构

### 6.1 设备数据 (DeviceRecycleItem)

```typescript
interface DeviceRecycleItem {
  // 通用字段
  serial_number: string;      // 序列号
  offlineDays: number;        // 离线天数
  group_name: string;         // 设备组名称
  moveType: '0' | '1';        // 回收方式: 0-自动, 1-手动
  moveTime: string;           // 回收时间
  move_author: string;        // 操作账户
  product?: string;           // 产品类型 (eNB/gNB)

  // eNB 专用
  host_name?: string;
  mac_address?: string;
  manufacturer?: string;
  city?: string;
  sub_branches?: string;
  sub_station_name?: string;
  install_address?: string;
  contact_number?: string;
  site_id?: string;
  circuit_ref?: string;
  circuit_jo?: string;
  service_status?: string;
  rom?: string;
  longitude?: number | string;
  latitude?: number | string;
  height?: number | string;

  // CPE 专用
  macaddress?: string;        // 注意: CPE使用macaddress而非mac_address
  imsi?: string;
  distance?: number | string;
  cpe_code?: string;          // CPE唯一标识
}
```

### 6.2 已选设备弹窗显示

| 设备类型 | 显示格式 | Row Key |
|----------|----------|---------|
| eNB | serial_number | small_cell_code |
| gNB | serial_number | small_cell_code |
| CPE | macaddress(serial_number) | cpe_code |

## 7. 权限控制

| 操作 | 权限检查 |
|------|----------|
| 显示操作按钮 | 根据 CODE_ENB_DEVICE_REGISTER / CODE_GNB_DEVICE_REGISTER / CODE_CPE_DEVICE 权限 |
| 批量操作 | batchOperation 配置项控制是否允许多选 |

## 8. 交互细节

### 8.1 搜索功能

- eNB/gNB: 按 `serial_number` 模糊搜索
- CPE: 按 `serial_number` 或 `macaddress` 模糊搜索
- 支持回车键触发搜索

### 8.2 批量选择

- 点击已选数量按钮展开已选列表弹窗
- 支持清空所有已选项
- 支持单个移除已选项
- 已选列表分页显示

### 8.3 操作确认

- 移出回收站: 弹出确认框 "确认将设备移出回收站？"
- 批量删除: 弹出确认框 "确认删除？"

## 9. 扩展配置

| 配置项 | 说明 |
|--------|------|
| isDeviceMoreParams | 是否显示更多扩展字段 |
| showOrHideCol | 控制显示哪组扩展字段 ('true' 显示运营商特定字段) |
| siteIdLabel | Site ID 字段标签 (国际化) |
| siteNameLabel | 站点名称字段标签 (国际化) |

## 10. 前端实现建议

### 10.1 组件结构

```
DeviceRecycleBin/
├── index.tsx              # 主页面
├── components/
│   ├── RecycleTable.tsx   # 设备表格
│   ├── SelectedDrawer.tsx # 已选设备抽屉
│   └── FilterBar.tsx      # 筛选条件栏
└── types.ts               # 类型定义
```

### 10.2 状态管理

```typescript
interface RecycleBinState {
  deviceType: 'eNB' | 'gNB' | 'CPE';
  searchText: string;
  selectedGroupId: string;
  selectedDevices: DeviceRecycleItem[];
  loading: boolean;
}
```

### 10.3 关键实现点

1. **动态列渲染**: 根据设备类型动态显示不同列
2. **批量选择**: 使用 DataTable 的 selection 功能
3. **已选抽屉**: 展示已选设备列表，支持移除
4. **权限控制**: 根据权限码控制操作按钮显示
5. **确认对话框**: 恢复和删除操作需要二次确认
