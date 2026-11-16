import { Dialog } from '@zovaris/sephiro/dialog';
import { Button } from '@zovaris/sephiro/button';
import { Copy } from '@phosphor-icons/react';
import { t } from '@/lib/stores/i18n';
import { toast } from '@/lib/stores/toast';
import { copyText } from '@/lib/utils/clipboard';
import { QrImage } from './QrImage';

/**
 * The invite for a player across the table: a large code on its own white
 * plate, the world's name and the link under it, and one action. The long URL
 * no longer sits in the dialog's description, where it wrapped into a block.
 */
export function QrDialog({
  open,
  name,
  url,
  onClose,
}: {
  open: boolean;
  name: string;
  url: string;
  onClose: () => void;
}) {
  async function copy() {
    if (!(await copyText(url))) {
      toast.error(t('copy_failed'));
      return;
    }
    toast.success(t('link_copied'));
  }

  return (
    <Dialog
      open={open}
      onClose={onClose}
      title={name}
      description={t('qr_scan_hint')}
      className="ftm-qr-dialog"
      footer={
        <Button variant="primary" onClick={copy}>
          <Copy size={16} />
          {t('copy_link')}
        </Button>
      }
    >
      <div class="ftm-qr-dialog__body">
        <span class="ftm-qr ftm-qr-dialog__plate">
          <QrImage value={url} size={232} alt={t('qr_for_phones')} />
        </span>
        <p class="ftm-qr-dialog__url" title={url}>
          {url.replace(/^https?:\/\//, '')}
        </p>
      </div>
    </Dialog>
  );
}
