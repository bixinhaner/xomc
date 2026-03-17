# 功能说明

使设备可以在商用、BETA两个环境来回迁移。

# 前端接口说明

前端接口类，全部在controller包下。下面是具体的接口说明。

## 获取设备迁移 jsp 页面接口

url：/migrate/toDeviceMigration

## 查询商用 ENB 列表接口

url：/migrate/commercialDevice/queryEnbList

参数：
- product：产品类型，如果有多个值，用逗号分隔
- searchText

返回数据：常规分页表格形式。

## 查询商用 CPE 列表接口

url：/migrate/commercialDevice/queryCpeList

参数：
- product：产品类型，如果有多个值，用逗号分隔
- searchText

返回数据：常规分页表格形式。

## 查询 BETA ENB 列表接口

url：/migrate/betaDevice/queryEnbList

参数：
- product：产品类型，如果有多个值，用逗号分隔
- searchText

返回数据：常规分页表格形式。

## 查询 BETA CPE 列表接口

url：/migrate/betaDevice/queryCpeList

参数：
- product：产品类型，如果有多个值，用逗号分隔
- searchText

返回数据：常规分页表格形式。

## 查询商用 ENB 产品列表接口
url：/migrate/commercialProduct/queryEnbProductList

返回数据：数组格式

## 查询商用 CPE 产品列表接口

url：/migrate/commercialProduct/queryCpeProductList

返回数据：数组格式

## 查询 BETA ENB 产品列表接口

url：/migrate/betaProduct/queryEnbProductList

返回数据：数组格式

## 查询 BETA CPE 产品列表接口

url：/migrate/betaProduct/queryCpeProductList

返回数据：数组格式

## 触发迁移动作

url：/migrate/execute

参数：
- direction：迁移方向。取值为：CommercialToBeta，或者BetaToCommercial
- enbCodeList：ENB small_cell_code列表。多个值用逗号分隔。例如：code-1,code-2
- cpeCodeList：CPE cpe_code列表。多个值用逗号分隔。

返回数据：{success: true, message: "xxx"}

## 查询commercial ENB迁移结果

url：/migrate/commercialResult/queryEnbList

参数：
- searchText

返回数据：
- 常规分页表格形式。
- properties：{successCount: 10, failCount: 10}
- status字段，后端返回的具体数值：success/fail/inProgress。后续各迁移结果接口，都是这三个值。

## 查询commercial CPE迁移结果

url：/migrate/commercialResult/queryCpeList

参数：
- searchText

返回数据：
- 常规分页表格形式。
- properties：{successCount: 10, failCount: 10}

## 查询beta ENB迁移结果

url：/migrate/betaResult/queryEnbList

参数：
- searchText

返回数据：
- 常规分页表格形式。
- properties：{successCount: 10, failCount: 10}

## 查询beta CPE迁移结果

url：/migrate/betaResult/queryCpeList

参数：
- searchText

返回数据：
- 常规分页表格形式。
- properties：{successCount: 10, failCount: 10}

## 导出commercial ENB迁移结果

url：/migrate/commercialResult/exportEnbList

参数：
- searchText

## 导出commercial CPE迁移结果

url：/migrate/commercialResult/exportCpeList

参数：
- searchText

## 导出beta ENB迁移结果

url：/migrate/betaResult/exportEnbList

参数：
- searchText

## 导出beta CPE迁移结果

url：/migrate/betaResult/exportCpeList

参数：
- searchText

## 查询迁移角色

url：/migrate/getMigrateRole

参数：无

结果：{ "migrateRole": "commercial" }

用途：前端根据此接口，获知当前OMC的角色，仅当当前角色为commercial时，才显示迁移按钮。