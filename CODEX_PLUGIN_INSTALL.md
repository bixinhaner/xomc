# Codex Plugin Install

请在本机安装/刷新 goomc 仓库内置的 Codex plugins。

仓库路径：`~/goomc`
插件 marketplace 根路径：`~/goomc`
插件 marketplace 名称：`omc-codex-skills`
插件 marketplace 文件：`~/goomc/.agents/plugins/marketplace.json`

## 执行步骤

1. 进入仓库：

   ```bash
   cd ~/goomc
   ```

2. 检查 marketplace：

   ```bash
   codex plugin marketplace list
   ```

3. 如果 marketplace 列表中没有名称为 `omc-codex-skills` 的条目，执行：

   ```bash
   codex plugin marketplace add ~/goomc
   ```

4. 刷新安装插件。`remove` 命令如果提示插件未安装或不存在，视为正常，继续执行后续命令：

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

如果命令因为权限、PATH alias 创建失败等非致命警告继续执行成功，不要中断。
如果 `add` 或 `list` 明确失败，记录失败命令和错误信息并停止。

安装完成后，请提示用户重新开启 Codex 会话，让插件生效。
