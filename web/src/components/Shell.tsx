import type { ComponentChildren } from 'preact';
import { useEffect, useState } from 'preact/hooks';
import { Kbd } from '@zovaris/sephiro/kbd';
import { IconAction } from '@/components/IconAction';
import { ArrowLeft, GearSix } from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { cycleTheme } from '@/lib/stores/theme';
import { navigate, route } from '@/router';
import { CommandPalette } from '@/components/CommandPalette';
import { Mark } from '@/components/Mark';

/**
 * One page, one header. ftm has a dashboard and a settings page, so the chrome
 * is the mark, the name, the command palette hint and a way to settings.
 */
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
    <div class="min-h-dvh w-full bg-bg text-text">
      <div class="mx-auto flex w-full max-w-app flex-col px-4 pb-16 sm:px-6 lg:px-8">
        <header class="flex items-center gap-3 py-6">
          <a
            href="/"
            class="flex min-w-0 flex-1 items-center gap-3 text-text no-underline"
            onClick={(event) => {
              event.preventDefault();
              navigate('/');
            }}
          >
            <Mark class="size-8 shrink-0" />
            <span class="truncate text-base font-medium">{t('app_name')}</span>
          </a>

          <button
            type="button"
            class="flex cursor-pointer items-center rounded-control border-0 bg-transparent p-1 max-sm:hidden"
            aria-label={t('command_palette_placeholder')}
            onClick={() => setPaletteOpen(true)}
          >
            <Kbd keys={['⌘', 'K']} variant="ghost" />
          </button>

          {isSettings ? (
            <IconAction
              label={t('go_back')}
              side="bottom"
              icon={<ArrowLeft size={18} />}
              onClick={() => navigate('/')}
            />
          ) : (
            <IconAction
              label={t('settings')}
              side="bottom"
              icon={<GearSix size={18} />}
              onClick={() => navigate('/settings')}
            />
          )}
        </header>

        <main class="min-w-0">{children}</main>
      </div>

      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}
