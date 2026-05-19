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
