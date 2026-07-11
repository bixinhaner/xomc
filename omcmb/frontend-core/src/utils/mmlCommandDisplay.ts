export interface MmlCommandDisplayParam {
  key: string;
  value: string;
}

export interface MmlCommandDisplay {
  raw: string;
  operation: string;
  target: string;
  commandHead: string;
  params: MmlCommandDisplayParam[];
  parameterCount: number;
}

function splitOnce(value: string, delimiter: string): [string, string] {
  const index = value.indexOf(delimiter);
  if (index === -1) return [value, ''];
  return [value.slice(0, index), value.slice(index + delimiter.length)];
}

export function parseMmlCommandDisplay(command: string): MmlCommandDisplay {
  const raw = command.trim();
  const [headPart, paramsPart] = splitOnce(raw, ':');
  const commandHead = headPart.trim();
  const firstSpace = commandHead.search(/\s/);
  const operation = firstSpace === -1 ? commandHead : commandHead.slice(0, firstSpace).trim();
  const target = firstSpace === -1 ? '' : commandHead.slice(firstSpace).trim();

  const params = paramsPart
    ? paramsPart
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean)
        .map((item) => {
          const [key, value] = splitOnce(item, '=');
          return {
            key: key.trim(),
            value: value.trim(),
          };
        })
    : [];

  return {
    raw,
    operation,
    target,
    commandHead,
    params,
    parameterCount: params.length,
  };
}
