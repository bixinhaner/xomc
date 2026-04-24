# MML控制台
1、/mml/console 页面，执行命令 API：/api/v1/mml/execute 执行时，未从页面上携带选择的参数，mml_tasks 表里的 commands字段json 格式中的parameters为空，用户在页面上可以取消个别参数的执行，每次执行命令时不是命令绑定的参数都需要执行
2、需要把执行的参数传递到 device_tasks 表里并投递队列，队列需要把需要执行的参数，解析成满足 TR069协议的 RPC 报文