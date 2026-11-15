import { Dialog } from '@zovaris/sephiro/dialog';
import { Button } from '@zovaris/sephiro/button';
import { t } from '@/lib/stores/i18n';

export function ConfirmDialog({
  open,
  title,
  body,
  confirmLabel,
  danger = false,
  onConfirm,
  onCancel,
}: {
  open: boolean;
  title: string;
  body: string;
  confirmLabel?: string;
  danger?: boolean;
  onConfirm: () => void;
  onCancel: () => void;
}) {
  return (
    <Dialog
      open={open}
      onClose={onCancel}
      title={title}
      role="alertdialog"
      closeButton={false}
      footer={
        <>
          <Button variant="secondary" onClick={onCancel}>
            {t('cancel')}
          </Button>
          <Button variant="primary" onClick={onConfirm}>
            {confirmLabel ?? t('delete')}
          </Button>
        </>
      }
    >
      <p class={`m-0 text-sm leading-relaxed ${danger ? 'text-text-muted' : 'text-text'}`}>
        {body}
      </p>
    </Dialog>
  );
}
