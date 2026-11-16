import { Cloud, Globe, Hourglass, Plugs, TerminalWindow } from '@phosphor-icons/react';

/** A glyph per provider family, so a preset reads at a glance. */
export function ProviderIcon({ provider, size = 16 }: { provider: string; size?: number }) {
  const props = { size, className: 'shrink-0', 'aria-hidden': true } as const;
  switch (provider) {
    case 'cloudflared':
      return <Cloud {...props} />;
    case 'pinggy':
      return <Hourglass {...props} />;
    case 'serveo':
    case 'localhostrun':
      return <TerminalWindow {...props} />;
    case 'bore':
      return <Plugs {...props} />;
    default:
      return <Globe {...props} />;
  }
}
