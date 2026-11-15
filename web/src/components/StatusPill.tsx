import { Badge } from '@zovaris/sephiro/badge';
import { StatusMark, type StatusMarkState } from '@zovaris/sephiro/status-mark';
import { t } from '@/lib/stores/i18n';

const MARK: Record<string, StatusMarkState> = {
  online: 'success',
  starting: 'progress',
  connecting: 'progress',
  installing: 'progress',
  downloading: 'progress',
  stopping: 'progress',
  error: 'error',
  timeout: 'error',
  stopped: 'idle',
  offline: 'idle',
  need_installing: 'idle',
};

const VARIANTS: Record<string, 'neutral' | 'accent' | 'success' | 'warning' | 'danger'> = {
  online: 'success',
  starting: 'accent',
  connecting: 'accent',
  installing: 'warning',
  downloading: 'warning',
  stopping: 'neutral',
  error: 'danger',
  timeout: 'danger',
  need_installing: 'warning',
  stopped: 'neutral',
  offline: 'neutral',
};

export function statusMark(state: string): StatusMarkState {
  return MARK[state] ?? 'idle';
}

export function statusLabel(state: string): string {
  return t(state);
}

export function StatusMarkDot({
  state,
  label,
  size = 'md',
  percent,
}: {
  state: string;
  label: string;
  size?: 'sm' | 'md' | 'lg';
  percent?: number | null;
}) {
  const mark = statusMark(state);
  return (
    <StatusMark
      state={mark}
      label={label}
      size={size}
      value={percent ?? undefined}
    />
  );
}

export function StatusBadge({ state }: { state: string }) {
  const idle = state === 'stopped' || state === 'offline';

  if (idle) {
    return <span class="text-2xs text-text-muted">{t(state)}</span>;
  }

  return (
    <Badge variant={VARIANTS[state] ?? 'neutral'} size="sm">
      {t(state)}
    </Badge>
  );
}
