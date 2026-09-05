// Browser-test entry only. Vite production build does not include this HTML.
// API fixtures are installed by Playwright, never by the production component.
import React from 'react';
import { createRoot } from 'react-dom/client';
import { App, ConfigProvider, theme } from 'antd';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { BrowserRouter } from 'react-router-dom';
import { IntlProvider } from 'react-intl';
import ActiveIntelligence from '../../src/pages/system/ActiveIntelligence';
import zh from '../../../frontend-core/src/i18n/zh-CN';
import en from '../../../frontend-core/src/i18n/en-US';
import { antdClassicTheme } from '../../src/theme/classicTheme';
import 'antd/dist/reset.css';
const query = new URLSearchParams(location.search);
const dark = query.get('theme') === 'dark'; const english = query.get('locale') === 'en-US';
// The host supplies semantic dark tokens; do not pin the classic light palette.
const darkTokens = { ...antdClassicTheme.token };
delete darkTokens.colorBgLayout; delete darkTokens.colorText; delete darkTokens.colorTextHeading;
const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
function Surface() { const { token } = theme.useToken(); return <div style={{ background: token.colorBgLayout, color: token.colorText, minHeight: '100vh', padding: 'clamp(12px,2.5vw,36px)', fontFamily: token.fontFamily }}><ActiveIntelligence /></div>; }
createRoot(document.getElementById('root')!).render(<React.StrictMode><BrowserRouter><IntlProvider locale={english ? 'en-US' : 'zh-CN'} messages={english ? en : zh}><ConfigProvider theme={{ ...antdClassicTheme, token: dark ? darkTokens : antdClassicTheme.token, cssVar: true, algorithm: dark ? theme.darkAlgorithm : theme.defaultAlgorithm }}><App><QueryClientProvider client={client}><Surface /></QueryClientProvider></App></ConfigProvider></IntlProvider></BrowserRouter></React.StrictMode>);
