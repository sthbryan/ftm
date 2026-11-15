import type { ComponentChildren } from 'preact';
import { useEffect, useState } from 'preact/hooks';
import { Sidebar } from '@zovaris/sephiro/sidebar';
import { Kbd } from '@zovaris/sephiro/kbd';
import { IconButton } from '@zovaris/sephiro/icon-button';
import {
  Gear,
  MagnifyingGlass,
  Palette,
  Plugs,
} from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { tunnels } from '@/lib/stores/tunnels';
import { cycleTheme } from '@/lib/stores/theme';
import { navigate, route } from '@/router';
import { CommandPalette } from '@/components/CommandPalette';
import { Mark } from '@/components/Mark';

export function Shell({ children }: { children: ComponentChildren }) {
  const [paletteOpen, setPaletteOpen] = useState(false);

  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault();
        setPaletteOpen((open) => !open);
      }
      if ((event.metaKey || event.ctrlKey) && event.shiftKey && event.key.toLowerCase() === 't') {
        event.preventDefault();
        cycleTheme();
      }
    };

    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const isSettings = route.value === '/settings';

  return (
    <div class="flex h-dvh w-full overflow-hidden bg-bg text-text">
      <Sidebar
        ariaLabel={t('app_name')}
        value={isSettings ? 'settings' : 'tunnels'}
        onSelect={(value) => navigate(value === 'settings' ? '/settings' : '/')}
        sections={[
          {
            items: [
              {
                value: 'tunnels',
                label: t('connections'),
                icon: <Plugs size={18} />,
                badge: tunnels.value.length > 0 ? tunnels.value.length : undefined,
              },
              {
                value: 'settings',
                label: t('settings'),
                icon: <Gear size={18} />,
              },
            ],
          },
        ]}
        header={
          <div class="flex items-center gap-2.5 px-1">
            <Mark class="size-8 shrink-0" />
            <div class="min-w-0">
              <p class="m-0 truncate text-sm leading-tight font-semibold text-text">
                {t('app_name')}
              </p>
              <p class="m-0 truncate text-2xs text-text-muted">{t('app_tagline')}</p>
            </div>
          </div>
        }
        footer={
          <div class="flex items-center justify-between gap-2 px-1">
            <IconButton
              label={t('appearance')}
              icon={<Palette size={16} />}
              onClick={() => cycleTheme()}
            />
            <Kbd keys={['⇧', 'T']} variant="ghost" />
          </div>
        }
        collapsible
        defaultCollapsed={false}
        collapsedMode="rail"
      />

      <div class="flex min-w-0 flex-1 flex-col">
        <header class="flex shrink-0 items-center justify-between gap-3 border-b border-border px-4 py-2">
          <div class="flex min-w-0 items-center gap-2 text-2xs text-text-muted">
            <span class="truncate">{t('dashboard_hint')}</span>
          </div>

          <button
            type="button"
            class="sph-button"
            data-variant="secondary"
            data-size="sm"
            onClick={() => setPaletteOpen(true)}
          >
            <MagnifyingGlass size={14} />
            <span class="text-2xs">{t('command_palette_placeholder')}</span>
            <Kbd keys={['⌘', 'K']} variant="ghost" />
          </button>
        </header>

        <main class="min-h-0 flex-1 overflow-hidden p-3">{children}</main>
      </div>

      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}
