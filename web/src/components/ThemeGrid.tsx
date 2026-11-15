import { Check } from '@phosphor-icons/react';
import { currentTheme, followSystem, isAutoTheme, setTheme } from '@/lib/stores/theme';
import { themeOptions } from '@/lib/data/themes';
import { t } from '@/lib/stores/i18n';

export function ThemeGrid() {
  return (
    <div class="px-5 pb-4">
      <div class="mb-2.5 flex items-center justify-between gap-3">
        <p class="m-0 text-2xs text-text-muted">{t('match_system')}</p>
        <button
          type="button"
          class="sph-button"
          data-variant={isAutoTheme.value ? 'primary' : 'secondary'}
          data-size="sm"
          aria-pressed={isAutoTheme.value}
          onClick={followSystem}
        >
          {t('theme_match_system')}
        </button>
      </div>

      <div class="grid gap-2 sm:grid-cols-2">
        {themeOptions.map((option) => {
          const selected = currentTheme.value === option.id;

          return (
            <button
              key={option.id}
              type="button"
              class="flex cursor-pointer items-center gap-3 rounded-[var(--sph-radius-sm)] border px-2.5 py-2 text-start transition-colors"
              style={{
                borderColor: selected ? 'var(--sph-accent)' : 'var(--sph-border)',
                backgroundColor: selected ? 'var(--sph-accent-soft)' : 'transparent',
              }}
              aria-pressed={selected}
              onClick={() => setTheme(option.id)}
            >
              <span
                class="flex shrink-0 items-center gap-1 rounded-[var(--sph-radius-xs)] border p-1"
                style={{
                  borderColor: 'var(--sph-border)',
                  backgroundColor: option.bg,
                }}
                aria-hidden="true"
              >
                <span
                  class="size-3 rounded-[2px]"
                  style={{ backgroundColor: option.surface, boxShadow: 'inset 0 0 0 1px rgb(255 255 255 / 0.06)' }}
                />
                <span class="size-3 rounded-full" style={{ backgroundColor: option.accent }} />
              </span>

              <span class="min-w-0 flex-1 truncate text-xs text-text">{option.label}</span>

              {selected && <Check size={13} className="shrink-0 text-primary" />}
            </button>
          );
        })}
      </div>
    </div>
  );
}
