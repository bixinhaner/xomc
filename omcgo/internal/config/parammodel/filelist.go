package parammodel

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// mergeFileLists 合并 builtin/custom 两组 XML basename 列表为绝对路径切片(T-0178)。
//
// 行为契约:
//   - 同名(完全相等的 basename)冲突时,按 customOverrides 决定胜出:
//   - true  → custom 文件赢(用户上传同名 XML 压制内置)
//   - false → builtin 文件赢(强制使用出厂值)
//   - 输出按 basename 字典序排序(确定性,便于测试 + 日志可读)
//   - 输入切片不被修改(纯函数)
//
// 用途:Loader.run 的 Pass 1 文件枚举核心算法。
func mergeFileLists(
	builtinDir string, builtinFiles []string,
	customDir string, customFiles []string,
	customOverrides bool,
) []string {
	chosen := make(map[string]string, len(builtinFiles)+len(customFiles))
	for _, name := range builtinFiles {
		chosen[name] = filepath.Join(builtinDir, name)
	}
	for _, name := range customFiles {
		_, conflict := chosen[name]
		if !conflict || customOverrides {
			chosen[name] = filepath.Join(customDir, name)
		}
	}
	out := make([]string, 0, len(chosen))
	for _, abs := range chosen {
		out = append(out, abs)
	}
	sort.Slice(out, func(i, j int) bool {
		return filepath.Base(out[i]) < filepath.Base(out[j])
	})
	return out
}

// resolveLoadedFrom 计算写入 param_models.loaded_from 的字段值(T-0178)。
//
// 返回相对 base 的 slash 分隔路径(Windows 反斜杠规范化);
// filepath.Rel 失败时降级为裸 basename(防御性,避免 Loader 崩溃)。
//
// 期望输出形如:
//   - "param-mappings/BTS.xml"(builtin)
//   - "param-mappings-custom/CBQQ.xml"(custom)
//
// 这两个前缀是 source.go 中 ClassifySource / IsDeletable 守门判定的依据。
func resolveLoadedFrom(base, absPath string) string {
	rel, err := filepath.Rel(base, absPath)
	if err != nil {
		return filepath.Base(absPath)
	}
	return filepath.ToSlash(rel)
}

// findShadowedCustom 返回 custom basename 集合中,与 builtin basename 同名的子集
// (按 custom 输入顺序保留;不去重,假设输入已去重)。
//
// 用途(T-0178 R-NEW-T0178-8 缓解):Loader 启动期当 customOverrides=false 时,
// 同名 custom 文件被 builtin 压制 → host 上文件仍在但行为不生效,用户上传后
// 看不到生效。本函数列出受影响清单,供 Loader.run 写 WARN 日志,运维一眼能
// 看到哪些 custom 文件被静默忽略。
//
// 纯函数,无 FS 依赖。两个空切片输入返空切片。
func findShadowedCustom(builtinFiles, customFiles []string) []string {
	if len(builtinFiles) == 0 || len(customFiles) == 0 {
		return nil
	}
	builtinSet := make(map[string]struct{}, len(builtinFiles))
	for _, b := range builtinFiles {
		builtinSet[b] = struct{}{}
	}
	var out []string
	for _, c := range customFiles {
		if _, ok := builtinSet[c]; ok {
			out = append(out, c)
		}
	}
	return out
}

// resolveLoaderFiles 是 Loader.run 文件枚举步骤的纯函数封装(T-0178)。
//
// 两种模式:
//   - 显式白名单(whitelist != nil):仅扫 builtin,custom 不参与
//     (白名单语义强调"严格管控集合",custom 隐式追加会破坏该语义)
//   - 自动扫描(whitelist 空):builtin + custom 双目录扫描 + 同名合并
//
// custom 目录缺失(ENOENT)视为"首次部署未初始化",静默跳过 — builtin 仍正常加载。
// 其他 custom 错误(权限 / IO)→ warnings 返回,不阻塞 builtin。
// builtin 目录错误 → 直接 error,因为 builtin 应该永远可读。
//
// shadowedCustom 始终返回(不论 customOverrides),让调用方按当前配置决定是否
// 写 WARN(T-0178 R-NEW-T0178-8 缓解)。
func resolveLoaderFiles(
	builtinDir, customDir string,
	whitelist, reserved []string,
	customOverrides bool,
) (files, shadowedCustom, warnings []string, err error) {
	if len(whitelist) > 0 {
		files = make([]string, 0, len(whitelist))
		for _, name := range whitelist {
			files = append(files, filepath.Join(builtinDir, name))
		}
		return files, nil, nil, nil
	}

	builtinFiles, err := scanXMLFiles(builtinDir)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("scan builtin %s: %w", builtinDir, err)
	}
	builtinFiles = pruneReserved(builtinFiles, reserved)

	var customFiles []string
	if customDir != "" {
		st, statErr := os.Stat(customDir)
		switch {
		case statErr == nil && st.IsDir():
			cf, scanErr := scanXMLFiles(customDir)
			if scanErr != nil {
				warnings = append(warnings,
					fmt.Sprintf("scan custom dir %s: %v", customDir, scanErr))
			} else {
				customFiles = pruneReserved(cf, reserved)
			}
		case statErr == nil && !st.IsDir():
			warnings = append(warnings,
				fmt.Sprintf("custom path %s exists but is not a directory", customDir))
		case os.IsNotExist(statErr):
			// 首次部署 / custom dir 还没建,静默跳过 — 不写 warning
		default:
			warnings = append(warnings,
				fmt.Sprintf("stat custom dir %s: %v", customDir, statErr))
		}
	}

	shadowedCustom = findShadowedCustom(builtinFiles, customFiles)
	files = mergeFileLists(builtinDir, builtinFiles, customDir, customFiles, customOverrides)
	return files, shadowedCustom, warnings, nil
}
