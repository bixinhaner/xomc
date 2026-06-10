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

// antd 6 原生支持 React 19,v5 时代的 unstableSetRender 渲染器桥接已随升级移除。

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
