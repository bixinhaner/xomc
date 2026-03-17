---
mode: agent
---
# 目标
添加单元测试。类名和方法：TemplateServiceImpl.queryENBList()。

# 要求
- 不要生成的markdown格式的说明文件。
- 不要依赖springcloud，mysql，mongo等，必要时采用mockito模拟这些操作
- 单元测试类中，使用TestLoggerWatcher，用来在用例执行前后打印信息。
- 所有测试例，都应该调用所测试的方法，而不是将方法逻辑片段拿出来测试。
- 每次只写一个测试例，等我确认后，再继续。

# 重点关注

模板关联的指标级别对查询结果的影响。