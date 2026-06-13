import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import './styles/global.css';
import './styles/alarm-colors.css';
import './styles/animations.css';
import './styles/3d-effects.css';
import './styles/tech-effects.css';
import './styles/fresh-effects.css';
import './styles/cyberpunk-effects.css';
import './styles/minions-effects.css';
import './styles/tiffany-effects.css';
import './styles/rmb-effects.css';
import './styles/responsive.css';
import App from './App';
import { reloadOnceForStaleChunk } from './utils/staleChunkReload';

// antd 6 原生支持 React 19,v5 时代的 unstableSetRender 渲染器桥接已随升级移除。

// 发布后旧 hash 分包失效自愈：Vite 在动态 import（懒加载路由）失败时派发
// `vite:preloadError`。多见于发布替换了 hash 分包、而当前标签页仍跑旧 index.html。
// 重载一次即可拉取最新 index.html + 匹配分包。详见 utils/staleChunkReload。
window.addEventListener('vite:preloadError', (event) => {
  event.preventDefault();
  reloadOnceForStaleChunk();
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
