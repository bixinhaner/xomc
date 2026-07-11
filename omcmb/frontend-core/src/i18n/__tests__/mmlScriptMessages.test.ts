import { describe, expect, it } from 'vitest';

import { messages, SUPPORTED_LOCALES } from '../index';

const REQUIRED_KEYS = [
  'mml.script.action.execute',
  'mml.script.action.more',
  'mml.script.action.viewDetail',
  'mml.script.action.reimport',
  'mml.script.action.downloadTxt',
  'mml.script.action.importTxt',
  'mml.script.editBasicInfo',
  'mml.script.originalFile',
  'mml.script.validationVersion',
  'mml.script.readOnlyTxtContent',
  'mml.script.noContent',
  'mml.scriptImport.replaceTitle',
  'mml.scriptImport.chooseTxt',
  'mml.scriptImport.saveConfirm',
  'mml.scriptImport.warningTitle',
  'mml.scriptImport.warningSaveContent',
  'mml.scriptImport.continueSave',
  'mml.scriptImport.fileLabel',
  'mml.scriptImport.readOnlyPreview',
  'mml.scriptImport.all',
  'mml.scriptImport.onlyErrors',
  'mml.scriptImport.onlyWarnings',
  'mml.scriptImport.downloadErrorReport',
  'mml.scriptImport.lineNo',
  'mml.scriptImport.deviceSn',
  'mml.scriptImport.order',
  'mml.scriptImport.command',
] as const;

describe('MML script management i18n messages', () => {
  it.each(SUPPORTED_LOCALES)('defines script actions and TXT import messages for %s', (locale) => {
    for (const key of REQUIRED_KEYS) {
      expect(messages[locale][key], `${locale} missing ${key}`).toEqual(expect.any(String));
      expect(messages[locale][key].trim(), `${locale} empty ${key}`).not.toBe('');
    }
  });
});
