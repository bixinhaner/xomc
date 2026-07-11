# Remove V2/V3 Frontend Skins Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the V2 and V3 frontend skins completely while keeping V1/frontend-core buildable and redirecting all legacy V2/V3 URLs to V1.

**Architecture:** Collapse the npm workspace and Docker image to `frontend-core + webcode`, remove the parity guard that only existed for three-skin synchronization, and keep compatibility at the Nginx boundary with temporary 302 redirects. Update only active guidance; preserve historical reports with an archive notice on the former multi-skin architecture entry point.

**Tech Stack:** npm workspaces, React/Vite, TypeScript, Docker BuildKit, Nginx.

## Global Constraints

- Delete `omcmb/webcode-v2` and `omcmb/webcode-v3` completely.
- Keep `omcmb/webcode` and `omcmb/frontend-core` intact.
- Legacy `/v2`, `/v2/*`, `/v3`, and `/v3/*` return 302 to `/`.
- Active code/config/guidance must contain no executable V2/V3 or skin-parity references.
- Historical reports remain; `docs/project/frontend-multi-skin-plan-20260422.md` gets an archive notice.
- Never stage or commit `.codex/`.

---

### Task 1: Capture baseline and define active-reference checks

**Files:**
- Inspect: `omcmb/package.json`
- Inspect: `deployments/docker/Dockerfile.web`
- Inspect: `deployments/docker/default.conf`
- Inspect: `deployments/docker/default.local.conf`
- Inspect: `AGENTS.md`, `CLAUDE.md`, `README.md`

**Interfaces:**
- Produces the exact list of active files that must be zero-reference clean after deletion.

- [ ] **Step 1: Run the current V1 build/typecheck baseline**

Run: `cd omcmb && npm run typecheck --workspace webcode && npm run build --workspace webcode`

Expected: both commands pass before deletion.

- [ ] **Step 2: Record active references**

Run `rg` across package scripts, Docker/Nginx, root instructions, current README/system docs, and CI/config files while excluding historical reports and the two soon-to-be-deleted directories.

Expected: references match the design inventory; unrelated protocol `/v2/` paths such as MinIO registry endpoints are excluded by file/path context, not blindly removed.

---

### Task 2: Collapse npm workspace to V1

**Files:**
- Delete: `omcmb/webcode-v2/**`
- Delete: `omcmb/webcode-v3/**`
- Delete: `omcmb/scripts/skin-parity.mjs`
- Modify: `omcmb/package.json`
- Modify: `omcmb/package-lock.json`

**Interfaces:**
- Produces workspaces `frontend-core` and `webcode` only.
- Produces `npm run typecheck` that invokes V1 only.

- [ ] **Step 1: Remove the V2/V3 directories and parity script**

Use tracked deletion for both directories and the parity script. Confirm `git status` shows deletions only under those paths.

- [ ] **Step 2: Update root package metadata and scripts**

Set workspaces to `frontend-core`, `webcode`; lint paths to `webcode/src frontend-core/src`; typecheck to `npm run typecheck --workspace webcode`; remove `skin-parity`.

- [ ] **Step 3: Regenerate lockfile mechanically**

Run: `cd omcmb && npm install --package-lock-only --legacy-peer-deps`

Expected: lockfile contains no `webcode-v2`, `webcode-v3`, or their workspace nodes.

- [ ] **Step 4: Verify workspace**

Run:

```bash
cd omcmb
npm ci --legacy-peer-deps
npm run typecheck
npm run build --workspace webcode
```

Expected: all pass; no V2/V3 workspace lookup occurs.

---

### Task 3: Collapse Docker image and redirect legacy URLs

**Files:**
- Modify: `deployments/docker/Dockerfile.web`
- Modify: `deployments/docker/default.conf`
- Modify: `deployments/docker/default.local.conf`

**Interfaces:**
- Produces a V1-only image rooted at `/usr/share/nginx/html`.
- Produces Nginx 302 redirects for both legacy prefixes.

- [ ] **Step 1: Simplify Dockerfile.web**

Remove V2/V3 package/source copies, build commands, output validation loops, and final image copies. Keep a V1 `index.html` asset-existence validation.

- [ ] **Step 2: Replace both Nginx skin locations**

In both config variants, replace V2/V3 SPA blocks with:

```nginx
location ~ ^/(v2|v3)(/.*)?$ {
    return 302 /;
}
```

- [ ] **Step 3: Validate syntax and image contents**

Build web with compose, run `nginx -t` inside the container, and assert `/usr/share/nginx/html/v2` and `/v3` do not exist.

- [ ] **Step 4: Verify HTTP behavior**

Request `/`, `/v2`, `/v2/performance/query`, `/v3/`, and `/v3/product/kpi-library` against port 8081.

Expected: `/` is 200; every legacy path is 302 with `Location: /`.

---

### Task 4: Update active guidance and archive the old design

**Files:**
- Modify: `AGENTS.md`
- Modify: `CLAUDE.md`
- Modify: `README.md`
- Modify: `docs/系统说明文档.md`
- Modify: `deployments/docker/README.md`
- Modify: `deployments/docker/本地测试环境-访问地址与默认账号.md`
- Modify: `docs/project/frontend-multi-skin-plan-20260422.md`
- Modify other active CI/config files found in Task 1.

**Interfaces:**
- Produces V1-only instructions and commands.

- [ ] **Step 1: Replace root AI instructions**

Remove “three-skin iron law”, V2/V3 parity rules, and three-skin validation commands. State V1 is the only UI shell and shared business logic remains in frontend-core.

- [ ] **Step 2: Update current project/deployment docs**

Remove V2/V3 URLs and architecture claims. Keep V1 URL and `npm run typecheck` as the frontend validation entry point.

- [ ] **Step 3: Add archive notice**

At the top of the historical multi-skin plan, add the removal date, point to the V1-only architecture, and explicitly say the remaining body is historical.

- [ ] **Step 4: Run active-reference audit**

Search the active file set and executable scripts. Expected: no `webcode-v2`, `webcode-v3`, `skin-parity`, “三皮肤”, or executable `/v2/`/`/v3/` deployment reference except intentional Nginx redirect tests/config comments.

---

### Task 5: Final verification, review, GitLab MR, and merge

**Files:**
- Include the design, plan, deleted directories, lockfile, build/deploy config, and active guidance changes only.

**Interfaces:**
- Produces an independently merged GitLab architecture-cleanup MR.

- [ ] **Step 1: Run final verification**

Run fresh commands: `npm ci`, V1 typecheck/build, active-reference audit, `git diff --check`, Docker web build, `nginx -t`, HTTP redirects, and Docker status.

- [ ] **Step 2: Request independent code review**

Reviewer checks accidental V1/frontend-core deletion, lockfile correctness, stale active references, Docker image contents, and redirect semantics. Fix all Critical/Important findings.

- [ ] **Step 3: Commit**

Commit message: `refactor(frontend): 删除 V2 V3 前端皮肤`

- [ ] **Step 4: Push and create GitLab MR**

MR targets main and explicitly states it is separate from #36; include deletion counts and exact verification evidence.

- [ ] **Step 5: Merge and synchronize**

Merge the exact tested SHA, remove the remote branch, fast-forward local main, delete the local branch, and confirm only `.codex/` remains untracked.
