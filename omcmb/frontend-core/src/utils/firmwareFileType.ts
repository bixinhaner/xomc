import type { FirmwareLibraryFileType } from '../types/unifiedFileTransfer';

export type FirmwareFileManagerTab = 'upgrade' | 'patch' | 'ap' | 'fpga';

export function firmwareLibraryFileTypeToFileManagerTab(
  fileType: FirmwareLibraryFileType | undefined,
): FirmwareFileManagerTab {
  switch (fileType) {
    case 1:
      return 'patch';
    case 5:
      return 'ap';
    case 6:
      return 'fpga';
    case 0:
    default:
      return 'upgrade';
  }
}

export function parseFirmwareFileManagerTabParam(
  value: string | null | undefined,
): FirmwareFileManagerTab {
  switch (value?.trim().toLowerCase()) {
    case 'patch':
      return 'patch';
    case 'ap':
      return 'ap';
    case 'fpga':
      return 'fpga';
    case 'ups':
    case 'img':
    case 'image':
    case 'upgrade':
    default:
      return 'upgrade';
  }
}
