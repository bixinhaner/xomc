import { StrictMode } from 'react';
import { createRoot } from 'react-dom/client';
import { unstableSetRender } from 'antd';
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

// React 19 + antd v5 静态方法（Modal.confirm / message / notification）需要
// 通过 unstableSetRender 注册基于 createRoot 的渲染器，否则静态方法 silently fail。
// 等价于 @ant-design/v5-patch-for-react-19，内联以避免该包在 npm workspaces 中
// 被 hoist 到 omcmb/node_modules 后找不到同级 antd 的解析问题。
unstableSetRender((node, container) => {
  const c = container as Element & { _reactRoot?: ReturnType<typeof createRoot> };
  c._reactRoot ||= createRoot(c);
  const root = c._reactRoot;
  root.render(node);
  return async () => {
    await new Promise<void>((resolve) => {
      setTimeout(() => {
        root.unmount();
        resolve();
      }, 0);
    });
  };
});

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
