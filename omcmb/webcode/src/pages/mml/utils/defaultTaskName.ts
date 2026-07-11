import dayjs from 'dayjs';
import type { User } from '@core/store/userStore';

export function buildMmlScriptDefaultTaskName(prefix: string, currentUser?: User | null): string {
  const taskOwner = currentUser?.username || currentUser?.displayName || 'user';
  return `${prefix}_${taskOwner}_${dayjs().format('YYYY-MM-DD HH:mm:ss')}`;
}
