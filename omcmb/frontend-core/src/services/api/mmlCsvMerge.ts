function parseCsv(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let field = '';
  let quoted = false;
  for (let i = 0; i < text.length; i += 1) {
    const char = text[i];
    if (quoted) {
      if (char === '"' && text[i + 1] === '"') {
        field += '"';
        i += 1;
      } else if (char === '"') quoted = false;
      else field += char;
      continue;
    }
    if (char === '"') quoted = true;
    else if (char === ',') {
      row.push(field);
      field = '';
    } else if (char === '\n') {
      row.push(field);
      rows.push(row);
      row = [];
      field = '';
    } else if (char !== '\r') field += char;
  }
  if (field !== '' || row.length > 0) {
    row.push(field);
    rows.push(row);
  }
  return rows;
}

function encodeCsvField(value: string): string {
  return /[",\r\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value;
}

/** 合并逐任务 CSV，并将每个设备/命令块的序号改为跨任务连续编号。 */
export function mergeMmlTaskCsvTexts(texts: string[]): string {
  let header: string[] | undefined;
  const merged: string[][] = [];
  let sequence = 0;
  for (const source of texts) {
    const records = parseCsv(source.replace(/^\uFEFF/, ''));
    if (!header && records.length > 0) header = records[0];
    for (const record of records.slice(1)) {
      if (record[0]?.trim()) {
        sequence += 1;
        record[0] = String(sequence);
      }
      merged.push(record);
    }
  }
  const records = header ? [header, ...merged] : merged;
  return `\uFEFF${records.map((row) => row.map(encodeCsvField).join(',')).join('\r\n')}\r\n`;
}
