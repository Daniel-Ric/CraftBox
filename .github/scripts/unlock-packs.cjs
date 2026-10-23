const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');

const hex = (process.env.REPO_KEY || '').replace(/\s/g, '');
if (!hex || hex.length % 2 || !/^[0-9a-f]+$/i.test(hex)) {
  throw new Error('REPO_KEY must contain the hexadecimal git-crypt key');
}
const directory = fs.mkdtempSync(path.join(os.tmpdir(), 'CraftBox-key-'));
const keyPath = path.join(directory, 'key');
try {
  fs.writeFileSync(keyPath, Buffer.from(hex, 'hex'), { mode: 0o600 });
  const result = spawnSync('git-crypt', ['unlock', keyPath], { stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`git-crypt unlock failed: ${result.status}`);
} finally {
  if (fs.existsSync(keyPath)) fs.unlinkSync(keyPath);
  fs.rmdirSync(directory);
}
