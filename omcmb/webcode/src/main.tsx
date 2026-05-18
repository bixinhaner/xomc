// antd v5 在 React 19 下，静态方法（Modal.confirm / message / notification）
// 依赖已被 React 19 移除的 ReactDOM.render 而失效 —— 此补丁恢复其可用性。
// 必须在 antd 使用前最先导入。详见 https://u.ant.design/v5-for-19
import '@ant-design/v5-patch-for-react-19';
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

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
