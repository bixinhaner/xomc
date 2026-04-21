# MML控制台功能完善
1、MML控制台页面，点击”保存脚本“，弹出”保存脚本“页面，点击”确认“按钮，没有成功或者失败的提示
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
2、mml_script表里有数据，但脚本任务列表没有数据，请排查接口问题还是前端页面问题
