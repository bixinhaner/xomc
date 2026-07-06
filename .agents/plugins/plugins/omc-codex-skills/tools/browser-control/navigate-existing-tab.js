#!/usr/bin/env node
'use strict';

const http = require('http');

const port = Number(process.env.BROWSER_CONTROL_PORT || 9222);
const targetUrl = process.argv[2] || 'http://127.0.0.1:8081';
const preferredHost = process.env.BROWSER_CONTROL_HOST_MATCH || '127.0.0.1:8081';

function getJson(path) {
  return new Promise((resolve, reject) => {
    const req = http.get({ host: '127.0.0.1', port, path, timeout: 5000 }, (res) => {
      let body = '';
      res.setEncoding('utf8');
      res.on('data', (chunk) => {
        body += chunk;
      });
      res.on('end', () => {
        if (res.statusCode < 200 || res.statusCode >= 300) {
          reject(new Error(`GET ${path} returned HTTP ${res.statusCode}: ${body}`));
          return;
        }
        try {
          resolve(JSON.parse(body));
        } catch (err) {
          reject(new Error(`GET ${path} returned invalid JSON: ${err.message}`));
        }
      });
    });
    req.on('timeout', () => {
      req.destroy(new Error(`GET ${path} timed out`));
    });
    req.on('error', reject);
  });
}

function requestJson(method, path) {
  return new Promise((resolve, reject) => {
    const req = http.request({ host: '127.0.0.1', port, path, method, timeout: 5000 }, (res) => {
      let body = '';
      res.setEncoding('utf8');
      res.on('data', (chunk) => {
        body += chunk;
      });
      res.on('end', () => {
        if (res.statusCode < 200 || res.statusCode >= 300) {
          reject(new Error(`${method} ${path} returned HTTP ${res.statusCode}: ${body}`));
          return;
        }
        try {
          resolve(JSON.parse(body));
        } catch (err) {
          reject(new Error(`${method} ${path} returned invalid JSON: ${err.message}`));
        }
      });
    });
    req.on('timeout', () => {
      req.destroy(new Error(`${method} ${path} timed out`));
    });
    req.on('error', reject);
    req.end();
  });
}

function delay(ms) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function send(ws, id, method, params = {}) {
  ws.send(JSON.stringify({ id, method, params }));
}

async function connectAndNavigate(target) {
  const ws = new WebSocket(target.webSocketDebuggerUrl);
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });

  let id = 1;
  send(ws, id++, 'Page.enable');
  send(ws, id++, 'Page.bringToFront');
  send(ws, id++, 'Page.navigate', { url: targetUrl });
  await delay(2500);
  ws.close();
}

(async () => {
  const beforeTargets = await getJson('/json/list');
  const beforePages = beforeTargets.filter((target) => target.type === 'page');
  let target = beforePages.find((page) => page.url.includes(preferredHost)) || beforePages[0];

  if (!target) {
    target = await requestJson('PUT', `/json/new?${encodeURIComponent(targetUrl)}`);
  }

  await connectAndNavigate(target);

  const afterTargets = await getJson('/json/list');
  const afterPages = afterTargets.filter((item) => item.type === 'page');
  console.log(JSON.stringify({
    port,
    targetUrl,
    beforePageCount: beforePages.length,
    beforePages: beforePages.map(({ id, title, url, type }) => ({ id, title, url, type })),
    afterPageCount: afterPages.length,
    afterPages: afterPages.map(({ id, title, url, type }) => ({ id, title, url, type })),
  }, null, 2));
})().catch((err) => {
  console.error(err.stack || String(err));
  process.exit(1);
});
