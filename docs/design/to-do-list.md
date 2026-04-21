# MML控制台页面调整
1、去掉“目标设备 2 台“ 的提示
2、去掉设备名字上面的整个”MML控制台“ DIV
3、保存脚本功能，弹框让用户输入脚本名称以及脚本其它信息，参考”MML脚本“添加界面


# MML控制台功能完善
1、MML控制台页面，点击”保存脚本“，没有成功提示，脚本任务列表没有新增数据
接口：http://172.19.1.73:8081/mml/script
返回成功：
{
    "id": "c4b917e4-c923-41f1-91c7-a87314a7f894",
    "script_name": "查询传输链路_2026-04-21",
    "description": "Saved from MML console: DSP LINKSTATUS",
    "content": "DSP LINKSTATUS",
    "device_type": "",
    "creator": "admin",
    "tags": [],
    "created_at": "2026-04-21T11:04:41.49985+08:00",
    "updated_at": "2026-04-21T11:04:41.49985+08:00"
}