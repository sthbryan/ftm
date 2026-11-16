import { RadioGroup } from '@zovaris/sephiro/radio-group';
import { SegmentedControl } from '@zovaris/sephiro/segmented-control';
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
import { rememberRevealOrigin, revealTheme } from '@/lib/themeReveal';

/**
 * A family drawn as a tiny ftm window: title bar dots, the ticket with its
 * world name, a button in the accent and the stub with its code. The thumb
 * carries the family's own theme attributes, so every colour in it is read
 * from that family's theme sheet, never copied here.
 */
function Thumb({ family, scheme }: { family: Family; scheme: Scheme }) {
  const { sephiroTheme, theme } = htmlAttributes(family, scheme);
  return (
    <span data-sephiro-theme={sephiroTheme} data-theme={theme} aria-hidden="true" class="ftm-thumb">
      <span class="ftm-thumb__bar">
        <i />
        <i />
        <i />
      </span>
      <span class="ftm-thumb__ticket">
        <span class="ftm-thumb__body">
          <b />
          <u />
          <s />
        </span>
        <span class="ftm-thumb__stub">
          <em />
        </span>
      </span>
    </span>
  );
}

/** Auto follows the OS; Dark and Light force a variant of every family. */
export function ThemeModeControl() {
  return (
    <div onPointerDown={rememberRevealOrigin}>
      <SegmentedControl
        ariaLabel={t('theme_mode')}
        size="sm"
        options={[
          { value: 'system', label: t('theme_auto') },
          { value: 'dark', label: t('theme_dark') },
          { value: 'light', label: t('theme_light') },
        ]}
        value={themeMode.value}
        onValueChange={(mode: string) => revealTheme(() => setThemeMode(mode))}
      />
    </div>
  );
}

/**
 * Every family as a thumbnail of the app in the variant on screen, so dark mode
 * shows dark thumbnails and light mode light ones. Under each: the name and
 * where the palette comes from.
 */
export function ThemeGrid() {
  const scheme = currentScheme.value;
  const families = FAMILIES.map((family) => ({
    value: family.id,
    label: (
      <span class="ftm-theme-option">
        <Thumb family={family.id} scheme={scheme} />
        <span class="ftm-theme-option__name">
          {family.name}
          <small>{family.source}</small>
        </span>
      </span>
    ),
  }));

  return (
    <div onPointerDown={rememberRevealOrigin}>
      <RadioGroup
        name="theme-family"
        label={t('theme_family')}
        size="sm"
        orientation="horizontal"
        className="ftm-theme-families"
        options={families}
        value={themeFamily.value}
        onValueChange={(family: string) => revealTheme(() => setThemeFamily(family))}
      />
    </div>
  );
}
