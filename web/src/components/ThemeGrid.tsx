import { RadioGroup } from '@zovaris/sephiro/radio-group';
import { SegmentedControl } from '@zovaris/sephiro/segmented-control';
import { Tooltip } from '@zovaris/sephiro/tooltip';
import {
  FAMILIES,
  currentScheme,
  htmlAttributes,
  setThemeFamily,
  setThemeMode,
  themeFamily,
  themeMode,
  type Family,
  type Scheme,
} from '@/lib/stores/theme';
import { t } from '@/lib/stores/i18n';

/**
 * A family's look in one variant. The swatch carries the family's own theme
 * attributes, so it reads the real tokens from the theme sheets instead of a
 * copied palette.
 */
function Swatch({ family, scheme }: { family: Family; scheme: Scheme }) {
  const { sephiroTheme, theme } = htmlAttributes(family, scheme);
  return (
    <span
      data-sephiro-theme={sephiroTheme}
      data-theme={theme}
      aria-hidden="true"
      class="ftm-swatch"
    />
  );
}

/**
 * Mode first (Auto follows the OS, Dark and Light force it), then the family as
 * a grid of named options, each with a swatch of its page colour and accent.
 * Swatches show the variant that is on screen, so dark mode shows dark swatches
 * and light mode light ones. The tooltip credits where the palette comes from.
 */
export function ThemeGrid() {
  const scheme = currentScheme.value;
  const modes = [
    { value: 'system', label: t('theme_auto') },
    { value: 'dark', label: t('theme_dark') },
    { value: 'light', label: t('theme_light') },
  ];
  const families = FAMILIES.map((family) => ({
    value: family.id,
    label: (
      <Tooltip content={family.source}>
        <span class="ftm-theme-option">
          <Swatch family={family.id} scheme={scheme} />
          {family.name}
        </span>
      </Tooltip>
    ),
  }));

  return (
    <div class="flex w-full flex-col items-end gap-3">
      <SegmentedControl
        ariaLabel={t('theme_mode')}
        size="sm"
        options={modes}
        value={themeMode.value}
        onValueChange={setThemeMode}
      />
      <RadioGroup
        name="theme-family"
        label={t('theme_family')}
        size="sm"
        orientation="horizontal"
        className="ftm-theme-families"
        options={families}
        value={themeFamily.value}
        onValueChange={setThemeFamily}
      />
    </div>
  );
}
