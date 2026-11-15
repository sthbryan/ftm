import { useEffect, useState } from 'preact/hooks';
import { Terminal, type TerminalLine } from '@zovaris/sephiro/terminal';
import { logsApi } from '@/lib/api';
import { t } from '@/lib/stores/i18n';
import { formatLogs } from '@/lib/utils/logs';
import { isRunningState } from '@/lib/utils/status';

const MAX_LINES = 400;

function toLines(text: string): TerminalLine[] {
  return text
    .split('\n')
    .filter((line) => line.length > 0)
    .slice(-MAX_LINES)
    .map((line, index) => ({ id: `${index}-${line.slice(0, 24)}`, text: line }));
}

export function LogsPanel({ tunnelId, state }: { tunnelId: string; state: string }) {
  const [lines, setLines] = useState<TerminalLine[]>([]);
  const [error, setError] = useState<string | null>(null);
  const running = isRunningState(state);

  useEffect(() => {
    let stream: { close: () => void } | null = null;
    let cancelled = false;

    setLines([]);
    setError(null);

    if (!tunnelId || !running) return;

    logsApi
      .get(tunnelId)
      .then((initial) => {
        if (!cancelled) setLines(toLines(formatLogs(initial)));
      })
      .catch(() => {
        if (!cancelled) setError(t('error_loading_logs'));
      });

    stream = logsApi.createStream(tunnelId, {
      onLine: (line) => {
        if (cancelled) return;
        setLines((current) => [...current, ...toLines(formatLogs(line))].slice(-MAX_LINES));
      },
      onClose: () => {
        stream = null;
      },
    });

    return () => {
      cancelled = true;
      stream?.close();
    };
  }, [tunnelId, running]);

  return (
    <Terminal
      lines={error ? [{ text: error, tone: 'error' }] : lines}
      label={t('detail_logs')}
      title={t('detail_logs')}
      running={running}
      empty={running ? t('no_logs') : t('detail_logs_idle')}
      autoScroll
      copyLabel={t('copy_link')}
      copiedLabel={t('copied')}
    />
  );
}
