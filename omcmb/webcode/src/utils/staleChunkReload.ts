/**
 * 部署后「旧 hash 分包失效」自愈工具。
 *
 * 背景：前端是 hash 命名的代码分包（如 `CurrentAlarms-<hash>.js`），发布新版本会生成
 * 新 hash 并删除旧分包。若用户的标签页在发布前已打开（仍跑着旧 index.html），之后再
 * 懒加载某个路由时会去取**旧 hash 分包**——服务端已 404 → 浏览器报
 * 「Failed to fetch dynamically imported module」，页面白屏 / 报错。
 *
 * index.html 已配 `no-cache`，所以一次重载即可拿到最新 index.html + 匹配分包。
 * 这里做两件事：识别这类错误、安全地重载一次（带防抖避免重载死循环）。
 */

const CHUNK_RELOAD_KEY = 'omc:chunkReloadAt';
const RELOAD_DEBOUNCE_MS = 10_000;

// 各浏览器/打包器对动态 import 失败的报错措辞不一，全覆盖。
const CHUNK_ERROR_RE =
  /Failed to fetch dynamically imported module|error loading dynamically imported module|Importing a module script failed|Loading chunk [\w-]+ failed|ChunkLoadError/i;

/** 判断一个错误是否为「动态分包加载失败」（典型 = 发布后旧 hash 分包失效）。 */
export function isChunkLoadError(error: unknown): boolean {
  if (!error) return false;
  const msg = error instanceof Error ? error.message : String(error);
  return CHUNK_ERROR_RE.test(msg);
}

/**
 * 重载一次以自愈旧分包失效。
 * 防抖：`RELOAD_DEBOUNCE_MS` 内只重载一次——若分包是真的取不到（离线/服务端真挂），
 * 不至于陷入无限重载。返回是否真的触发了重载。
 */
export function reloadOnceForStaleChunk(): boolean {
  try {
    const now = Date.now();
    const last = Number(window.sessionStorage.getItem(CHUNK_RELOAD_KEY) ?? '0');
    if (now - last < RELOAD_DEBOUNCE_MS) return false;
    window.sessionStorage.setItem(CHUNK_RELOAD_KEY, String(now));
  } catch {
    // sessionStorage 不可用（隐私模式等）——仍允许重载一次
  }
  window.location.reload();
  return true;
}
