# MML控制台

# 系统管理

# 关于 MML 任务的处理流程，涉及 app、acs、worker 三个服务之间的交互流程
## 场景描述
添加一个脚本任务，可以选择立即执行、定时执行、周期执行，需要有一个方案将脚本任务转化为 mml_task 里的任务，同时将 mml_task 里的任务分解为 devices_tasks任务，同时需要将devices_tasks任务投递到ACS里的 RPC 任务，基站链接 ACS 后通过 RPC 任务的方式下放给基站，基站再通过 RPC response 的方式响应给 ACS，ACS 将根据 eventCode 投递不同队列，app 或者 worker 订阅队列，消费数据并完整相关的devices_tasks 任务处理，如果是 mml 的任务，同时需要通知 mml 的消息订阅，mml 通过订阅的方式指导任务处理结果，并在 mml_tasks 和 mml_scripts记录执行状态和记录，展示在 mml 相关页面上
## 确保关于 devices_tasks 的处理流程统一，切入口一致
1、mml  任务  > devices 任务 > acs 服务下行给基站 > 基站响应给 acs > acs部分数据处理 >投递对应队列 >消费队列数据 更新 devices 任务数据 > 投递任务来源队列（source=mml,投递mml 队列）> mml 队列消费>更新 mml 的相关数据（如果存在script_id，增根据 ID 去更新 mml_scripts 表数据）
2、mml 是一个任务来源的上游，系统涉及允许多个任务来源上游，上游处理自由业务，acs 和 devices_tasks 需要在项目层级统一且标准