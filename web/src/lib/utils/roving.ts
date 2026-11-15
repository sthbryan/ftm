const NAVIGATION_KEYS = ['ArrowRight', 'ArrowDown', 'ArrowLeft', 'ArrowUp', 'Home', 'End'];

export function rovingRadioKeydown(event: KeyboardEvent) {
  if (!NAVIGATION_KEYS.includes(event.key)) return;

  const group = event.currentTarget as HTMLElement;
  const radios = [
    ...group.querySelectorAll<HTMLElement>('[role="radio"]:not([disabled])'),
  ];
  const current = radios.indexOf(document.activeElement as HTMLElement);
  if (current === -1) return;

  event.preventDefault();

  const last = radios.length - 1;
  const next =
    event.key === 'Home'
      ? 0
      : event.key === 'End'
        ? last
        : event.key === 'ArrowRight' || event.key === 'ArrowDown'
          ? current === last
            ? 0
            : current + 1
          : current === 0
            ? last
            : current - 1;

  radios[next].focus();
  radios[next].click();
}
