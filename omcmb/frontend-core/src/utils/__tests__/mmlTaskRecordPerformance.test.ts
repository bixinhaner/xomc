import fs from 'node:fs';
import path from 'node:path';
import { describe, expect, it } from 'vitest';

const omcmbRoot = path.resolve(__dirname, '../../../..');

const taskRecordSources = [
  'webcode/src/pages/mml/TaskRecord/index.tsx',
  'webcode-v2/src/pages/mml/TaskRecord.tsx',
  'webcode-v3/src/pages/mml/task-records/index.tsx',
];

describe('MML task record result rendering contract', () => {
  it('does not load 200 device result rows when opening a task detail', () => {
    for (const relativePath of taskRecordSources) {
      const source = fs.readFileSync(path.join(omcmbRoot, relativePath), 'utf8');
      expect(source, relativePath).not.toMatch(/useMMLTaskResults\([^)]*,\s*1,\s*200\)/);
      expect(source, relativePath).toContain('TASK_RESULT_PAGE_SIZE');
    }
  });

  it('keeps the v1 task detail view off the console dynamic result table', () => {
    const source = fs.readFileSync(
      path.join(omcmbRoot, 'webcode/src/pages/mml/TaskRecord/index.tsx'),
      'utf8'
    );

    expect(source).not.toContain("from '../Console/components/ResultTable'");
    expect(source).not.toContain("from '../Console/adapters'");
    expect(source).not.toContain('useMMLTaskById');
  });

  it('keeps the v1 task detail view as a result table instead of an inline raw-output list', () => {
    const source = fs.readFileSync(
      path.join(omcmbRoot, 'webcode/src/pages/mml/TaskRecord/index.tsx'),
      'utf8'
    );

    expect(source).toContain('Table<DeviceTaskResultItem>');
    expect(source).toContain('resultColumns');
    expect(source).toContain('mml.resultCommand');
    expect(source).not.toContain('List<DeviceTaskResultItem>');
    expect(source).not.toContain('previewRawOutput');
  });
});
