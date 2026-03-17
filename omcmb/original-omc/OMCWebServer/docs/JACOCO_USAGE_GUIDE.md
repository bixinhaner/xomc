# JaCoCo 代码覆盖率使用指南

## 📊 简介

JaCoCo（Java Code Coverage）是一个开源的 Java 代码覆盖率工具，已集成到本项目中。

## 🚀 快速开始

### 1. 运行测试并生成覆盖率报告

```bash
mvn clean test jacoco:report
```

或者使用简化命令：

```bash
mvn clean test
```

运行特定测试类

```bash
mvn clean test -Dtest=TemplateServiceImplDealMLQDataInfoTest
```

### 2. 查看覆盖率报告

生成的报告位置：
```
target/coverage-reports/jacoco-ut/index.html
```

在浏览器中打开此 HTML 文件即可查看详细的覆盖率分析。

## 📈 覆盖率指标说明

### 覆盖率类型

| 指标 | 说明 | 优先级 |
|------|------|--------|
| **行覆盖率** (Line) | 源代码行是否被执行 | ⭐⭐⭐ 最常用 |
| **分支覆盖率** (Branch) | if/else、switch 等分支是否完全覆盖 | ⭐⭐⭐ 重要 |
| **方法覆盖率** (Method) | 是否调用了所有方法 | ⭐⭐ 参考 |
| **指令覆盖率** (Instruction) | 字节码指令的覆盖率 | ⭐ 技术细节 |
| **圆环复杂度** (Cyclomatic) | 代码复杂度 | ⭐⭐ 参考 |

### 颜色标记

- 🟩 **绿色** - 覆盖率 100%，代码被执行
- 🟨 **黄色** - 部分覆盖（50-99%），存在未执行分支
- 🟥 **红色** - 覆盖率 0%，代码未执行

## 🎯 当前项目覆盖率要求

未强制要求

## 💡 如何解读报告

### 1. 打开报告首页
```
target/coverage-reports/jacoco-ut/index.html
```

### 2. 查看项目级覆盖率
- 显示整个项目的覆盖率统计
- 包含行覆盖率、分支覆盖率等

### 3. 查看包级覆盖率
- 点击包名进入包详情
- 查看该包下各类的覆盖率

### 4. 查看类级覆盖率
- 点击类名进入类详情
- 查看每个方法的覆盖率
- **红色行** = 未执行
- **绿色行** = 已执行
- **黄色行** = 部分执行（分支未全覆盖）

### 5. 查看方法级覆盖率
- 点击方法名查看详细信息
- 显示该方法的覆盖率

## 🔍 分析覆盖率报告

### 识别未覆盖代码
1. 找出报告中的红色行
2. 这些是测试中未被执行的代码
3. 根据需要补充测试用例

### 识别部分覆盖的分支
1. 寻找黄色行
2. 这表示存在未被执行的代码分支
3. 需要补充测试用例覆盖所有分支

### 优化测试策略
1. 关注圆环复杂度高的方法
2. 对复杂方法补充更多测试用例
3. 确保所有业务逻辑分支都被测试

## 📊 生成多个覆盖率报告

### 只生成报告（不检查阈值）
```bash
mvn test jacoco:report
```

### 运行覆盖率检查（包含阈值验证）
```bash
mvn test jacoco:check
```

如果覆盖率未达到阈值，构建将失败。

### 跳过测试
```bash
mvn jacoco:report -DskipTests
```

## 🛠 自定义覆盖率规则

编辑 `pom.xml` 中 `<jacoco-check>` 部分的 `<rules>` 标签：

```xml
<rule>
    <element>BUNDLE</element>
    <excludes>
        <exclude>*Test</exclude>
    </excludes>
    <limits>
        <limit>
            <counter>LINE</counter>
            <value>COVEREDRATIO</value>
            <minimum>0.70</minimum>  <!-- 修改为 70% -->
        </limit>
    </limits>
</rule>
```

## 📖 更多信息

- [JaCoCo 官方网站](https://www.jacoco.org/)
- [JaCoCo Maven Plugin](https://www.eclemma.org/jacoco/trunk/doc/maven.html)
- [JaCoCo 规则配置](https://www.jacoco.org/jacoco/trunk/doc/maven.html#rule-configuration)
