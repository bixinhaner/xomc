import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it } from 'vitest';
import ScriptImportPreview from '../ScriptImportPreview';

describe('ScriptImportPreview', () => {
  it('filters the visible issue card with the selected severity', async () => {
    render(<ScriptImportPreview validation={{
      originalFilename: 'issues.txt',
      planItems: [],
      summary: { totalLines: 2, validLines: 1, effectiveLines: 1, deviceCount: 1, errorCount: 1, warningCount: 1 },
      issues: [
        { code: 'MML_LINE_FORMAT_INVALID', severity: 'error', lineNo: 1 },
        { code: 'MML_DEVICE_OFFLINE', severity: 'warning', lineNo: 2 },
      ],
    }} />);
    expect(screen.getByText(/MML_DEVICE_OFFLINE/)).toBeInTheDocument();
    await userEvent.click(screen.getByRole('button', { name: '仅看错误' }));
    expect(screen.getByText(/MML_LINE_FORMAT_INVALID/)).toBeInTheDocument();
    expect(screen.queryByText(/MML_DEVICE_OFFLINE/)).not.toBeInTheDocument();
  });
});
