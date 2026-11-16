import type { ComponentChildren } from 'preact';
import { useEffect, useRef, useState } from 'preact/hooks';
import { Button } from '@zovaris/sephiro/button';
import { Kbd } from '@zovaris/sephiro/kbd';
import { Link } from '@zovaris/sephiro/link';
import { IconAction } from '@/components/IconAction';
import { ArrowLeft, GearSix } from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { cycleTheme } from '@/lib/stores/theme';
import { revealTheme } from '@/lib/themeReveal';
import { navigate, route } from '@/router';
import { CommandPalette } from '@/components/CommandPalette';
import { Mark } from '@/components/Mark';
import { FoundryBeacon } from '@/components/FoundryBeacon';
import { dragRegion, shortcutModifier } from '@/lib/desktop';

/**
 * One page, one header. The header stays put and only the page under it
 * scrolls; a hairline appears under the header once the page has moved. In the
 * desktop window the header is also the title bar: empty space drags the
 * window, and on macOS it leaves room for the traffic lights.
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
        revealTheme(cycleTheme);
      }
    };

    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, []);

  const isHome = route.value === '/';
  const scroller = useRef<HTMLElement>(null);
  const [scrolled, setScrolled] = useState(false);

  useEffect(() => {
    scroller.current?.scrollTo({ top: 0 });
    setScrolled(false);
  }, [route.value]);

  return (
    <div class="ftm-shell bg-bg text-text">
      <header class="ftm-header" data-scrolled={scrolled ? '' : undefined} {...dragRegion()}>
        <div class="ftm-header__inner">
          <div class="flex min-w-0 flex-1">
            <Link
              href="/"
              variant="subtle"
              className="ftm-brand"
              onClick={(event) => {
                event.preventDefault();
                navigate('/');
              }}
            >
              <Mark class="ftm-brand__mark shrink-0" />
              <span class="truncate text-sm font-medium">{t('app_name')}</span>
            </Link>
          </div>

          <FoundryBeacon />

          <div class="max-sm:hidden">
            <Button
              variant="quiet"
              size="sm"
              aria-label={t('command_palette_placeholder')}
              onClick={() => setPaletteOpen(true)}
            >
              <Kbd keys={[shortcutModifier, 'K']} variant="ghost" />
            </Button>
          </div>

          {route.value === '/settings' ? (
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
        </div>
      </header>

      <main
        ref={scroller}
        class="ftm-scroll"
        onScroll={(event) => setScrolled((event.currentTarget as HTMLElement).scrollTop > 2)}
      >
        <div class={`ftm-content ${isHome ? '' : 'pt-2'}`}>{children}</div>
      </main>

      <CommandPalette open={paletteOpen} onOpenChange={setPaletteOpen} />
    </div>
  );
}
