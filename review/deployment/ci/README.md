# OMC GitLab CI 模板说明

本目录保存 OMC 项目的 GitLab CI 片段，用于把代码质量检查、安全扫描、镜像构建和镜像推送纳入统一流水线。这里的文件不是一个完整的 `.gitlab-ci.yml`，而是供仓库根目录的主 CI 文件通过 `include` 引用的模板。

## 为什么需要这套 CI

OMC 同时包含 Go 后端、前端 workspace、Docker 镜像和发布镜像仓库。如果只依赖人工本地验证，容易出现以下问题：

- 后端 `go vet`、测试或构建遗漏，导致运行时才发现编译或静态检查问题。
- 前端类型检查、单元测试或生产构建遗漏，导致页面包无法发布。
- 依赖漏洞没有固定入口检查，安全风险只能靠人工临时发现。
- 开发环境镜像和生产环境镜像的构建、命名、推送方式不一致。
- 发布标签可能被打在错误分支或错误提交上，造成不可追溯的版本。

这套 CI 的目标是把交付前的最低质量门禁固化下来：所有镜像发布都必须先经过后端质量、前端质量、后端漏洞检查、前端依赖审计，再按标签类型推送到对应镜像仓库。

## 文件职责

| 文件 | 职责 |
| --- | --- |
| `basic-ci.yml` | 定义公共变量，以及后端质量、前端质量、后端漏洞检查、前端依赖审计任务。 |
| `dev-tag-ci.yml` | 定义开发标签校验和开发镜像构建推送任务。开发标签不能指向 `master` 可达提交。 |
| `release-tag-ci.yml` | 定义发布标签校验和发布镜像构建推送任务。发布标签必须指向 `master` 可达提交。 |

## 流水线做什么

### 基础质量任务

`basic-ci.yml` 提供四个门禁任务：

- `backend:quality`
  - 进入 `omcgo/`
  - 执行 `go vet ./...`
  - 执行 `go test ./...`
  - 执行 `go build ./...`
- `frontend:quality`
  - 进入 `omcmb/`
  - 执行 `npm ci --include=dev --legacy-peer-deps`
  - 执行 `npm run typecheck`
  - 执行 `npm test --workspace webcode`
  - 执行 `npm run build --workspace webcode`
- `backend:govulncheck`
  - 安装指定版本 `govulncheck`
  - 对 `omcgo/` 执行 `govulncheck ./...`
- `frontend:audit`
  - 进入 `omcmb/`
  - 执行 `npm audit --audit-level=high --omit=dev`

### 开发标签镜像

`dev-tag-ci.yml` 提供两个任务：

- `validate:dev-tag`
  - 拉取 `origin/master`
  - 校验当前标签提交不能被 `origin/master` 包含
  - 用于避免开发标签污染正式主干发布链路
- `image:build:dev`
  - 并行构建 `app`、`acs`、`worker`、`web` 四个镜像
  - 镜像名为 `$OMC_DEV_REGISTRY/omc/$SERVICE:$CI_COMMIT_TAG`
  - 构建成功后推送到开发镜像仓库

### 发布标签镜像

`release-tag-ci.yml` 提供两个任务：

- `validate:release-tag`
  - 拉取 `origin/master`
  - 校验当前标签提交必须被 `origin/master` 包含
  - 用于保证生产发布只来自主干可追溯提交
- `image:build:release`
  - 并行构建 `app`、`acs`、`worker`、`web` 四个镜像
  - 标签版本会去掉前缀 `v`，例如 `v1.2.3` 会得到镜像标签 `1.2.3`
  - 先构建并推送到开发镜像仓库
  - 再把同一镜像重新打标签并推送到生产镜像仓库
  - `web` 镜像会额外传入 `APP_VERSION=$VERSION`

## 如何接入

在仓库根目录创建或维护 `.gitlab-ci.yml`，通过 `include` 引用本目录下的模板。示例：

```yaml
stages:
  - quality
  - security
  - validate
  - package

include:
  - local: review/deployment/ci/basic-ci.yml
  - local: review/deployment/ci/dev-tag-ci.yml
  - local: review/deployment/ci/release-tag-ci.yml

workflow:
  rules:
    - if: '$CI_COMMIT_TAG'
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH'

backend:quality:
  rules:
    - if: '$CI_COMMIT_TAG'
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH'

frontend:quality:
  rules:
    - if: '$CI_COMMIT_TAG'
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH'

backend:govulncheck:
  rules:
    - if: '$CI_COMMIT_TAG'
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH'

frontend:audit:
  rules:
    - if: '$CI_COMMIT_TAG'
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
    - if: '$CI_COMMIT_BRANCH'

validate:dev-tag:
  rules:
    - if: '$CI_COMMIT_TAG =~ /^dev-/'

image:build:dev:
  rules:
    - if: '$CI_COMMIT_TAG =~ /^dev-/'

validate:release-tag:
  rules:
    - if: '$CI_COMMIT_TAG =~ /^v[0-9]+\.[0-9]+\.[0-9]+/'

image:build:release:
  rules:
    - if: '$CI_COMMIT_TAG =~ /^v[0-9]+\.[0-9]+\.[0-9]+/'
```

上面的 `rules` 只是推荐入口，实际项目可以按团队约定调整标签命名。关键约束是：

- 开发镜像任务只应该在开发标签上运行。
- 发布镜像任务只应该在正式发布标签上运行。
- 镜像构建任务必须依赖基础质量和安全任务通过。

## 推荐标签约定

建议使用两类标签：

| 标签类型 | 示例 | 用途 | 分支约束 |
| --- | --- | --- | --- |
| 开发标签 | `dev-feature-a-20260714.1` | 构建开发验证镜像 | 不能指向 `master` 可达提交 |
| 发布标签 | `v1.2.3`、`v1.2.3-rc.1` | 构建可发布镜像 | 必须指向 `master` 可达提交 |

开发标签镜像保留完整 Git 标签作为镜像 tag。发布标签镜像会去掉开头的 `v`，例如 `v1.2.3` 会推送为 `1.2.3`。

## 需要的 CI 变量

以下变量需要在 GitLab 项目或 Group 的 CI/CD Variables 中配置。涉及密码的变量应设置为 masked/protected，并按环境限制可见范围。

| 变量 | 用途 |
| --- | --- |
| `CI_DEV_REGISTRY` | 开发镜像仓库地址，例如 `registry.example.com/dev`。 |
| `CI_DEV_REGISTRY_USER` | 开发镜像仓库用户名。 |
| `CI_DEV_REGISTRY_PASSWORD` | 开发镜像仓库密码或访问令牌。 |
| `CI_PRD_REGISTRY` | 生产镜像仓库地址，例如 `registry.example.com/prod`。 |
| `CI_PRD_REGISTRY_USER` | 生产镜像仓库用户名。 |
| `CI_PRD_REGISTRY_PASSWORD` | 生产镜像仓库密码或访问令牌。 |

模板内部会把这些变量映射为：

- `OMC_DEV_REGISTRY`
- `OMC_DEV_REGISTRY_USER`
- `OMC_DEV_REGISTRY_PASSWORD`
- `OMC_PRD_REGISTRY`
- `OMC_PRD_REGISTRY_USER`
- `OMC_PRD_REGISTRY_PASSWORD`

## 运行依赖

### GitLab Runner

Runner 需要支持 Docker executor，并允许 Docker-in-Docker 服务。镜像构建任务使用：

- `docker:29`
- `docker:29-dind`
- `DOCKER_HOST=tcp://docker:2375`
- `DOCKER_TLS_CERTDIR=""`

如果 Runner 开启了严格隔离，通常需要为 Docker-in-Docker 配置 privileged 模式。否则 `docker build`、`docker login` 或 `docker push` 可能失败。

### 基础镜像

CI 默认使用以下镜像：

| 变量 | 默认值 | 用途 |
| --- | --- | --- |
| `GO_VERSION_IMAGE` | `golang:1.25-alpine` | 后端质量检查和漏洞扫描。 |
| `NODE_VERSION_IMAGE` | `node:22-alpine` | 前端依赖安装、类型检查、测试和构建。 |
| `DOCKER_VERSION_IMAGE` | `docker:29` | Docker 镜像构建与推送。 |

### 网络依赖

Runner 需要能够访问：

- GitLab 仓库本身，包含 `git fetch origin master`。
- Go module 代理，默认 `https://goproxy.cn,https://proxy.golang.org,direct`。
- npm registry，默认 `https://registry.npmmirror.com`。
- `golang.org/x/vuln/cmd/govulncheck` 安装来源。
- 开发镜像仓库和生产镜像仓库。
- Docker 基础镜像来源，例如 `golang`、`node`、`docker`、`alpine`、`nginx` 等。

### 仓库文件依赖

镜像构建任务假设以下 Dockerfile 存在：

- `deployments/docker/Dockerfile.app`
- `deployments/docker/Dockerfile.acs`
- `deployments/docker/Dockerfile.worker`
- `deployments/docker/Dockerfile.web`

构建上下文是仓库根目录，因此 CI 必须在仓库根执行，不能切到 `deployments/docker/` 后再执行 `docker build`。

## 常用操作

### 触发开发镜像

在非 `master` 提交上打开发标签并推送：

```bash
git tag dev-my-feature-20260714.1 <commit-sha>
git push origin dev-my-feature-20260714.1
```

通过后会推送：

```text
$CI_DEV_REGISTRY/omc/app:dev-my-feature-20260714.1
$CI_DEV_REGISTRY/omc/acs:dev-my-feature-20260714.1
$CI_DEV_REGISTRY/omc/worker:dev-my-feature-20260714.1
$CI_DEV_REGISTRY/omc/web:dev-my-feature-20260714.1
```

### 触发发布镜像

在 `master` 可达提交上打发布标签并推送：

```bash
git tag v1.2.3 <commit-sha>
git push origin v1.2.3
```

通过后会推送：

```text
$CI_DEV_REGISTRY/omc/app:1.2.3
$CI_DEV_REGISTRY/omc/acs:1.2.3
$CI_DEV_REGISTRY/omc/worker:1.2.3
$CI_DEV_REGISTRY/omc/web:1.2.3

$CI_PRD_REGISTRY/omc/app:1.2.3
$CI_PRD_REGISTRY/omc/acs:1.2.3
$CI_PRD_REGISTRY/omc/worker:1.2.3
$CI_PRD_REGISTRY/omc/web:1.2.3
```

## 失败排查

| 现象 | 常见原因 | 处理方式 |
| --- | --- | --- |
| `Release tags must point to commits reachable from master` | 发布标签没有打在 `master` 可达提交上。 | 把标签移动到已合入 `master` 的提交，或先完成合并。 |
| `Dev tags must not point to master commits` | 开发标签打在了主干提交上。 | 使用开发分支提交重新打开发标签。 |
| `Cannot connect to the Docker daemon` | Runner 未正确配置 Docker-in-Docker。 | 检查 Runner executor、DinD service、privileged 配置和 `DOCKER_HOST`。 |
| `docker login` 失败 | 镜像仓库地址、用户名、密码或令牌错误。 | 检查 `CI_DEV_REGISTRY*` / `CI_PRD_REGISTRY*` 变量。 |
| `npm audit` 失败 | 生产依赖存在 high 及以上漏洞。 | 升级依赖或在 MR 中说明不可修复项并调整策略。 |
| `govulncheck` 失败 | Go 依赖或标准库调用路径存在已知漏洞。 | 升级相关 Go module 或修复受影响调用路径。 |
| `docker build` 找不到文件 | 构建上下文或 Dockerfile 路径不正确。 | 确认 CI 在仓库根执行，并且 `deployments/docker/Dockerfile.*` 存在。 |

## 维护约定

- 只在 `basic-ci.yml` 中维护公共版本变量，避免多个模板版本漂移。
- 新增服务镜像时，同步调整 `image:build:dev` 和 `image:build:release` 的 `SERVICE` matrix。
- 新增质量门禁时，应让开发标签和发布标签的镜像构建任务通过 `needs` 显式依赖该门禁。
- 修改标签规则时，应同时更新主 `.gitlab-ci.yml` 的 `rules` 和本文档。
- 生产镜像推送变量应优先使用 protected variables，避免普通分支或非受保护标签获得生产仓库写权限。
