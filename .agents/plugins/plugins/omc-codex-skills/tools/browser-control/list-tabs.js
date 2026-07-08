#!/usr/bin/env node
'use strict';

const http = require('http');

const port = Number(process.env.BROWSER_CONTROL_PORT || process.argv[2] || 9222);

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

(async () => {
  const targets = await getJson('/json/list');
  const pages = targets.filter((target) => target.type === 'page');
  const browserUi = targets.filter((target) => target.type === 'browser_ui');
  console.log(JSON.stringify({
    port,
    pageCount: pages.length,
    pages: pages.map(({ id, title, url, type }) => ({ id, title, url, type })),
    browserUiCount: browserUi.length,
  }, null, 2));
})().catch((err) => {
  console.error(err.stack || String(err));
  process.exit(1);
});
