import { IconButton, type IconButtonProps } from '@zovaris/sephiro/icon-button';
import { Tooltip } from '@zovaris/sephiro/tooltip';

/**
 * An icon-only button that says what it does: the label is the accessible name
 * and shows as a tooltip on hover and keyboard focus. Every icon-only control in
 * ftm goes through here, so none is a mystery glyph.
 */
export function IconAction({
  side = 'top',
  ...props
}: IconButtonProps & { side?: 'top' | 'right' | 'bottom' | 'left' }) {
  return (
    <Tooltip content={props.label} side={side}>
      <IconButton {...props} />
    </Tooltip>
  );
}
