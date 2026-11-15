import { SegmentedControl } from '@zovaris/sephiro/segmented-control';
import { setThemeMode, themeMode } from '@/lib/stores/theme';
import { t } from '@/lib/stores/i18n';

/** Tinta is the only theme; the user picks its mode or lets the OS decide. */
export function ThemeGrid() {
  const options = [
    { value: 'system', label: t('theme_auto') },
    { value: 'dark', label: t('theme_dark') },
    { value: 'light', label: t('theme_light') },
  ];

  return (
    <SegmentedControl
      ariaLabel={t('theme')}
      size="sm"
      options={options}
      value={themeMode.value}
      onValueChange={setThemeMode}
    />
  );
}
