---
applyTo: 'src/main/webapp/**'
---

# 代码修改规范与注意事项
## 修改前必须先阅读
- 修改任何代码前，先使用 read_file 或 grep 了解现有代码结构
- 确认文件路径、行号范围、相关依赖
- 理解现有的命名规范和代码风格

## 增量修改原则
- 优先使用 edit_file 进行精确的增量修改
- 避免大范围替换，减少引入错误的风险
- 每次修改后验证结果，确保修改正确应用

## 代码风格一致性
- 遵循项目现有的命名规范（camelCase/snake_case）
- 保持缩进风格一致（2空格/4空格/Tab）
- 使用项目已有的组件库和工具函数
- 不引入项目未使用的新依赖



# 组件改造规范
## API 兼容性检查
- 列出所有使用旧组件 API 的地方（如 $refs.table.refresh()）
- 逐一替换为新组件对应的 API
- 搜索项目中所有引用点，避免遗漏

## 样式冲突检测
- 检查新旧组件是否有全局样式冲突
- 测试其他页面是否受影响
- 必要时使用 scoped 或命名空间隔离样式



# 功能实现规范
## 数据驱动优先
- 优先使用 computed 属性处理数据转换
- 使用 v-for 动态渲染，避免硬编码重复代码
- 复杂逻辑封装为 methods 函数

## 事件处理规范
- 事件处理函数使用语义化命名：handleXxx, onXxx
- 保存 this 引用：var vm = this;
- 异步操作使用 Promise 或 async/await

## 错误处理
- API 调用添加 try-catch 或 .catch() 处理
- 用户操作添加加载状态和错误提示
- 边界情况处理（空数据、网络错误等）



# 测试用例规范
##  生成测试数据
// 生成模拟数据时包含所有必要字段
generateTestData(count) {
    var data = [];
    for (var i = 0; i < count; i++) {
        data.push({
            id: 'ID_' + i,
            name: '测试项_' + i,
            status: ['active', 'inactive'][i % 2],
            // ... 覆盖所有表格列需要的字段
        });
    }
    return data;
}

## 测试用例覆盖
- 空数据状态
- 单条数据
- 大量数据（1000+条测试虚拟滚动）
- 边界值（最大/最小值）
- 特殊字符（中文、特殊符号）



# 调试与验证规范
## 控制台日志
// 开发阶段添加调试日志
console.log('[功能名称] 操作描述:', 相关数据);

// 发布前移除或使用条件判断
if (process.env.NODE_ENV === 'development') {
    console.log('调试信息');
}

## 验证清单
修改完成后检查：
- [ ] 功能是否正常工作
- [ ] 控制台是否有报错
- [ ] 是否影响其他功能
- [ ] 样式是否正确显示
- [ ] 响应式布局是否正常
- [ ] 浏览器兼容性



# 文档与注释规范
## 代码注释
/**
 * 列配置变更处理
 * @param {Array} columns - 列配置数组
 * @param {Object} options - 配置选项
 * @returns {void}
 */
applyColumnConfig(columns, options) {
    // 1. 更新列显示状态
    // 2. 保存配置到后端
    // 3. 刷新表格
}



# 性能优化建议
## 大数据量处理
- 使用虚拟滚动（vxe-table scroll-y）
- 分页加载数据
- 懒加载/按需加载
- 避免在 template 中进行复杂计算

## 渲染优化
- 使用 v-show 替代频繁切换的 v-if
- 为 v-for 添加唯一 key
- 使用 Object.freeze() 冻结静态数据
- 合理使用 computed 缓存计算结果