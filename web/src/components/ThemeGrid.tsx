import { currentTheme, isAutoTheme, setTheme, setThemeFamily, systemScheme } from '@/lib/stores/theme';
import { themeFamilies } from '@/lib/data/themes';
import { t } from '@/lib/stores/i18n';

export function ThemeGrid() {
  return (
    <div class="grid gap-1 px-5 pb-4">
      {themeFamilies.map((family) => (
        <div
          key={family.id}
          class="flex items-center justify-between gap-3 rounded-[var(--sph-radius-sm)] px-1 py-1.5"
        >
          <button
            type="button"
            class="cursor-pointer bg-transparent text-start text-xs text-text"
            onClick={() => setThemeFamily(family.id)}
            aria-pressed={isAutoTheme.value && systemScheme.value === 'dark'}
          >
            {family.name}
          </button>

          <div class="flex items-center gap-1.5">
            {([family.dark, family.light] as const).map((variant) => {
              const selected = currentTheme.value === variant.id;
              return (
                <button
                  key={variant.id}
                  type="button"
                  class="size-5 cursor-pointer rounded-full border transition-transform hover:scale-110"
                  style={{
                    backgroundColor: variant.color,
                    borderColor: selected ? 'var(--color-text)' : 'var(--color-border)',
                    boxShadow: selected ? '0 0 0 2px var(--sph-accent-soft)' : 'none',
                  }}
                  onClick={() => setTheme(variant.id)}
                  aria-label={variant.id}
                  aria-pressed={selected}
                  title={variant.id}
                />
              );
            })}
          </div>
        </div>
      ))}

      <p class="m-0 mt-1 px-1 text-2xs text-text-muted">{t('match_system')}</p>
    </div>
  );
}
