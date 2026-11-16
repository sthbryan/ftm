import { Dialog } from '@zovaris/sephiro/dialog';
import { Button } from '@zovaris/sephiro/button';
import { Copy } from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { copyText } from '@/lib/utils/clipboard';
import { QrImage } from './QrImage';

export function QrDialog({
  open,
  url,
  onClose,
}: {
  open: boolean;
  url: string;
  onClose: () => void;
}) {
  async function copy() {
    if (!(await copyText(url))) {
      toast.error(t('copy_failed'));
      return;
    }
    toast.success(t('qr_copied'));
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={t('qr_for_phones')}
      description={url}
      footer={
        <Button variant="secondary" onClick={copy}>
          <Copy size={14} />
          {t('qr_copy')}
        </Button>
      }
    >
      <div class="flex justify-center">
        <QrImage value={url} size={220} class="rounded-control" />
      </div>
    </Dialog>
  );
}
