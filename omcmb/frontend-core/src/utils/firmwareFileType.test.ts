import { describe, expect, it } from 'vitest';
import {
  firmwareLibraryFileTypeToFileManagerTab,
  parseFirmwareFileManagerTabParam,
} from './firmwareFileType';

describe('firmware file type helpers', () => {
  it('maps AP firmware library files to the AP upload tab', () => {
    expect(firmwareLibraryFileTypeToFileManagerTab(5)).toBe('ap');
  });

  it('maps the other firmware library file types to file manager tabs', () => {
    expect(firmwareLibraryFileTypeToFileManagerTab(0)).toBe('upgrade');
    expect(firmwareLibraryFileTypeToFileManagerTab(1)).toBe('patch');
    expect(firmwareLibraryFileTypeToFileManagerTab(6)).toBe('fpga');
    expect(firmwareLibraryFileTypeToFileManagerTab(undefined)).toBe('upgrade');
  });

  it('parses file manager URL aliases', () => {
    expect(parseFirmwareFileManagerTabParam('ups')).toBe('upgrade');
    expect(parseFirmwareFileManagerTabParam('IMG')).toBe('upgrade');
    expect(parseFirmwareFileManagerTabParam('image')).toBe('upgrade');
    expect(parseFirmwareFileManagerTabParam('fpga')).toBe('fpga');
  });
});
