import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const basePath = path.join(__dirname, '..', 'src-tauri', 'target', 'release', 'bundle');

const copies = [
  {
    src: path.join(__dirname, '..', 'src-tauri', 'bin', 'ftm'),
    dest: path.join(basePath, 'macos', 'Foundry Tunnel Manager.app', 'Contents', 'MacOS', 'ftm'),
  },
  {
    src: path.join(__dirname, '..', 'src-tauri', 'bin', 'ftm'),
    dest: path.join(basePath, 'linux', 'ftm'),
  },
  {
    src: path.join(__dirname, '..', 'src-tauri', 'bin', 'ftm.exe'),
    dest: path.join(basePath, 'windows', 'ftm.exe'),
  },
];

copies.forEach(({ src, dest }) => {
  if (fs.existsSync(src)) {
    const dir = path.dirname(dest);
    fs.mkdirSync(dir, { recursive: true });
    fs.copyFileSync(src, dest);
    console.log(`✓ Copied ${path.basename(src)} to ${dest}`);
  }
});
