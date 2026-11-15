import { useEffect, useState } from 'preact/hooks';
import { SpringToaster } from '@zovaris/sephiro/toast';
import { Shell } from '@/components/Shell';
import { Dashboard } from '@/components/Dashboard';
import { SettingsPage } from '@/components/SettingsPage';
import { MiniPanel } from '@/components/MiniPanel';
import { initI18n, i18nReady } from '@/lib/stores/i18n';
import { connectTunnels, disconnectTunnels } from '@/lib/stores/tunnels';
import { fetchProviders } from '@/lib/stores/providers';
import { loadSettings } from '@/lib/stores/settings';
import { checkForUpdate } from '@/lib/stores/update';
import { route } from '@/router';

export function App() {
  const [ready, setReady] = useState(i18nReady.value);

  useEffect(() => {
    void initI18n().then(() => setReady(true));
    void fetchProviders();
    void loadSettings().catch(() => undefined);
    void checkForUpdate();
    connectTunnels();

    return () => disconnectTunnels();
  }, []);

  if (route.value === '/mini') {
    return ready ? <MiniPanel /> : null;
  }

  return (
    <>
      {ready ? (
        <Shell>
          {route.value === '/settings' ? <SettingsPage /> : <Dashboard />}
        </Shell>
      ) : (
        <BootScreen />
      )}
      <SpringToaster position="bottom-right" />
    </>
  );
}

function BootScreen() {
  return (
    <div class="flex h-dvh items-center justify-center bg-bg" aria-busy="true" aria-live="polite">
      <span class="sph-spinner sph-spinner--md" />
    </div>
  );
}
