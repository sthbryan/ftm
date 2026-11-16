import QRCode from 'qrcode';
import { Spinner } from '@zovaris/sephiro/spinner';
import { useEffect, useState } from 'preact/hooks';

/**
 * Renders a QR for a URL with a white quiet zone, sized in CSS pixels. With
 * `fluid`, `size` only sets the resolution and CSS decides the box.
 */
export function QrImage({
  value,
  size = 220,
  fluid = false,
  class: className,
  alt,
}: {
  value: string;
  size?: number;
  fluid?: boolean;
  class?: string;
  alt?: string;
}) {
  const [dataUrl, setDataUrl] = useState('');

  useEffect(() => {
    if (!value) return;
    let active = true;

    QRCode.toDataURL(value, {
      width: Math.max(240, size * 3),
      margin: 1,
      color: { dark: '#000000', light: '#ffffff' },
    })
      .then((next) => {
        if (active) setDataUrl(next);
      })
      .catch(() => undefined);

    return () => {
      active = false;
    };
  }, [value, size]);

  if (!dataUrl) {
    return <Spinner size="md" label={alt ?? value} />;
  }

  return (
    <img
      src={dataUrl}
      alt={alt ?? value}
      width={size}
      height={size}
      class={className}
      style={fluid ? undefined : { width: `${size}px`, height: `${size}px` }}
    />
  );
}
