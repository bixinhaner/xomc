# Codex Plugin Install

请在本机安装或刷新 goomc 仓库内置的 Codex plugins。

仓库根目录：本文中的 `<GOOMC_REPO>` 指当前机器上的 goomc 仓库根目录，例如：

```bash
/Users/shangyingbin/project/goomc
```

插件 marketplace 根路径：`<GOOMC_REPO>`
插件 marketplace 名称：`omc-codex-skills`
插件 marketplace 文件：`<GOOMC_REPO>/.agents/plugins/marketplace.json`

当前 marketplace 包含：

- `omc-codex-skills`：OMC 本地交付、浏览器控制等团队技能集
- `mattpocock-skills`：诊断、TDD、triage 等辅助技能集

## 执行步骤

1. 进入仓库：

   ```bash
   cd <GOOMC_REPO>
   ```

2. 检查 marketplace：

   ```bash
   codex plugin marketplace list
   ```

3. 如果 marketplace 列表中没有名称为 `omc-codex-skills` 的条目，执行：

   ```bash
   codex plugin marketplace add <GOOMC_REPO>
   ```

4. 刷新安装插件。

   常规刷新只需要执行 `add`，Codex 会按 marketplace 中的当前版本安装或更新本地缓存：

   ```bash
   codex plugin add omc-codex-skills@omc-codex-skills
   codex plugin add mattpocock-skills@omc-codex-skills
   ```

   如果怀疑本地缓存状态异常，可以先 remove 再 add。`remove` 命令如果提示插件未安装或不存在，视为正常，继续执行后续命令：

   ```bash
   codex plugin remove omc-codex-skills@omc-codex-skills
   codex plugin add omc-codex-skills@omc-codex-skills

   codex plugin remove mattpocock-skills@omc-codex-skills
   codex plugin add mattpocock-skills@omc-codex-skills
   ```

5. 验证安装状态：

   ```bash
   codex plugin list --marketplace omc-codex-skills
   ```

## 验收条件

- `omc-codex-skills` 显示为 installed 且 enabled。
- `mattpocock-skills` 显示为 installed 且 enabled。
- `omc-codex-skills` 版本应与 `<GOOMC_REPO>/.agents/plugins/plugins/omc-codex-skills/.codex-plugin/plugin.json` 中的 `version` 一致。
- `omc-codex-skills` 的 PATH 应指向 `<GOOMC_REPO>/.agents/plugins/plugins/omc-codex-skills`。
- `mattpocock-skills` 的 PATH 应指向 `<GOOMC_REPO>/.agents/plugins/plugins/mattpocock-skills`。

## 常见情况

- 如果命令输出 `WARNING: proceeding, even though we could not create PATH aliases: Operation not permitted`，但后续显示插件安装或列表读取成功，这是非致命警告，不需要中断。
- 如果 `codex plugin marketplace list` 中没有 `omc-codex-skills`，先执行 `codex plugin marketplace add <GOOMC_REPO>`。
- 如果 `codex plugin add` 报无法写入 `~/.codex/plugins/cache`，需要给 Codex 命令写入本机插件缓存的权限后重试。
- 如果 `codex plugin add` 或 `codex plugin list --marketplace omc-codex-skills` 明确失败，记录失败命令和错误信息并停止。

## 更新插件源后

如果修改了 `<GOOMC_REPO>/.agents/plugins/plugins/omc-codex-skills` 里的 skill、脚本或 manifest：

1. 更新 `.codex-plugin/plugin.json` 中的 cachebuster 版本。
2. 提交并推送 goomc 仓库。
3. 在本机执行：

   ```bash
   codex plugin add omc-codex-skills@omc-codex-skills
   ```

4. 开启新的 Codex 会话，让新版本 plugin 被重新加载。

安装完成后，请重新开启 Codex 会话，让插件生效。
