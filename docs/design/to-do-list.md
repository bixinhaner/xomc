# MML控制台调整
1、~~选择"基本信息" > "设备信息"命令 ，控制面板未根据命令绑定参数进行内容拼接~~ ✅ 已修复
   - 后端 GetCommand/GetCommandByCode 新增 params enrichment（service.go）
   - 新增种子数据 seed/000007_seed_mml_basic_info_params.sql 绑定 LST/MOD DEVICE_INFO 的14个参数
2、~~命令树小的自定义模板改为自定义命令~~ ✅ 已修复
   - i18n key mml.console.customTemplates 从"自定义模板"改为"自定义命令"
3、~~自定义模板下的命令选择时，API 报错~~ ✅ 已修复
   - useCommandSelection.ts selectCommand 跳过 tmpl- 前缀 ID 的 API 调用
   - 根因：模板合成 ID（tmpl-*）无法通过后端 UUID 解析


# 脚本任务调整
1、~~标签页：任务记录、脚本任务，两个标签页相互交互名称；并且调整顺序脚本任务放在前面~~ ✅ 已确认正确（标签和数据源已正确对应）
2、~~脚本任务的数据从 mml_scripts中获取数据，任务记录从 mml_tasks 中获取数据，确保两个页面调用 API 地址是正确的~~ ✅ 已确认正确（API 映射无误）
