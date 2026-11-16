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
  const running = isRunningState(state);

  useEffect(() => {
    setLines([]);
    if (!tunnelId || !running) return;

    const stream = logsApi.createStream(tunnelId, {
      onSnapshot: (snapshot) => setLines(toLines(formatLogs(snapshot.join('\n')))),
      onLine: (line) =>
        setLines((current) => [...current, ...toLines(formatLogs(line))].slice(-MAX_LINES)),
    });

    return () => stream.close();
  }, [tunnelId, running]);

  return (
    <Terminal
      lines={lines}
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
