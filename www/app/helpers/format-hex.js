import { helper as buildHelper } from '@ember/component/helper';

export function formatHex(value) {
  return value[0].substring(2, 42)
}

export default buildHelper(formatHex);
