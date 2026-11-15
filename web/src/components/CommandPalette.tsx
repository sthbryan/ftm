import { CommandPalette as Palette, type CommandItem } from '@zovaris/sephiro/command-palette';
import { Gear, Palette as PaletteIcon, Play, Plus, Stop, Plugs } from '@phosphor-icons/react';
import { startTunnel, stopTunnel, tunnels } from '@/lib/stores/tunnels';
import { cycleTheme } from '@/lib/stores/theme';
import { providerLabel } from '@/lib/stores/providers';
import { toast } from '@/lib/stores/toast';
import { t } from '@/lib/stores/i18n';
import { isRunningState } from '@/lib/utils/status';
import { navigate } from '@/router';
import { requestCreate, requestSelect } from '@/lib/stores/ui';

export function CommandPalette({
  open,
  onOpenChange,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}) {
  const list = tunnels.value;

  const items: CommandItem[] = [
    {
      id: 'new',
      label: t('new_connection_action'),
      group: t('connections'),
      icon: <Plus size={15} />,
      shortcut: ['N'],
    },
    ...list.map((tunnel) => {
      const running = isRunningState(tunnel.state);
      return {
        id: running ? `stop-${tunnel.id}` : `start-${tunnel.id}`,
        label: `${running ? t('stop') : t('start')} · ${tunnel.name}`,
        hint: `${providerLabel(tunnel.provider)} · ${tunnel.port}`,
        group: t('connections'),
        icon: running ? <Stop size={15} /> : <Play size={15} />,
        keywords: `${tunnel.name} ${tunnel.provider} ${tunnel.port}`,
      };
    }),
    {
      id: 'settings',
      label: t('settings'),
      group: t('app_name'),
      icon: <Gear size={15} />,
    },
    {
      id: 'theme',
      label: t('theme'),
      group: t('app_name'),
      icon: <PaletteIcon size={15} />,
      shortcut: ['⇧', 'T'],
    },
  ];

  async function select(item: CommandItem) {
    onOpenChange(false);

    if (item.id === 'new') {
      requestCreate();
      navigate('/');
      return;
    }

    if (item.id === 'settings') {
      navigate('/settings');
      return;
    }

    if (item.id === 'theme') {
      cycleTheme();
      return;
    }

    const [action, ...rest] = (item.id ?? '').split('-');
    const id = rest.join('-');
    const tunnel = list.find((candidate) => candidate.id === id);
    if (!tunnel) return;

    try {
      if (action === 'start') await startTunnel(tunnel.id);
      else await stopTunnel(tunnel.id);
    } catch (error) {
      toast.error((error as Error).message);
    }
  }

  return (
    <Palette
      open={open}
      onOpenChange={onOpenChange}
      items={items}
      onSelect={select}
      hotkey={false}
      placeholder={t('search_placeholder')}
      emptyMessage={t('search_empty')}
      ariaLabel={t('connections')}
      filter={(item, query) => {
        const haystack = `${String(item.label ?? '')} ${item.keywords ?? ''}`.toLowerCase();
        return haystack.includes(query.toLowerCase());
      }}
      className="sph-command-palette"
    />
  );
}
