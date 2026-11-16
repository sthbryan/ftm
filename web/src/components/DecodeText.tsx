import { useEffect, useRef, useState } from 'preact/hooks';

/** Characters a public link is made of, so the scramble looks like one. */
const GLYPHS = 'abcdefghijklmnopqrstuvwxyz0123456789-.';

const reducedMotion = () =>
  typeof window !== 'undefined' && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

function randomGlyph(): string {
  return GLYPHS[Math.floor(Math.random() * GLYPHS.length)];
}

/**
 * Text that arrives like a link being negotiated: every character starts as
 * noise and settles left to right. With `closing`, it goes the other way and
 * the characters collapse into dots from the end, for a link that just closed.
 * Runs once per change of `text` or `closing`; reduced motion shows the result.
 */
export function DecodeText({
  text,
  closing = false,
  duration = 640,
  class: className,
}: {
  text: string;
  closing?: boolean;
  duration?: number;
  class?: string;
}) {
  const [shown, setShown] = useState(closing ? text : '');
  const frame = useRef(0);

  useEffect(() => {
    cancelAnimationFrame(frame.current);

    if (reducedMotion() || !text) {
      setShown(closing ? '·'.repeat(Math.min(text.length, 24)) : text);
      return;
    }

    const started = performance.now();
    const tick = (now: number) => {
      const progress = Math.min(1, (now - started) / duration);
      const settled = Math.floor(progress * text.length);
      let next = '';
      for (let i = 0; i < text.length; i += 1) {
        if (closing) {
          next += i >= text.length - settled ? '·' : text[i];
        } else if (i < settled || text[i] === '/' || text[i] === ':') {
          next += text[i];
        } else {
          next += randomGlyph();
        }
      }
      setShown(next);
      if (progress < 1) frame.current = requestAnimationFrame(tick);
    };
    frame.current = requestAnimationFrame(tick);

    return () => cancelAnimationFrame(frame.current);
  }, [text, closing, duration]);

  return (
    <span class={className} aria-label={closing ? undefined : text}>
      <span aria-hidden="true">{shown || ' '}</span>
    </span>
  );
}
