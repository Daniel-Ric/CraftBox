const fs = require('node:fs');

function buildMetadata(env, defaultVersion) {
  const ref = env.GITHUB_REF || '';
  const event = env.GITHUB_EVENT_NAME;
  const tagged = /^refs\/tags\/[vr]/.test(ref);
  let version = env.INPUT_VERSION || (tagged ? ref.slice('refs/tags/v'.length) : defaultVersion.trim());
  if (!/^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$/.test(version)) {
    throw new Error(`Invalid CraftBox version: ${JSON.stringify(version)}`);
  }
  if (!/^[0-9a-f]{40,64}$/.test(env.GITHUB_SHA || '') || !/^[1-9][0-9]*$/.test(env.GITHUB_RUN_NUMBER || '')) {
    throw new Error('Missing or invalid GitHub build identity');
  }
  const primary = ref === 'refs/heads/master';
  const preview = ref === 'refs/heads/preview';
  const publish = (event === 'push' && (primary || preview || tagged)) ||
    (event === 'workflow_dispatch' && (primary || preview) && env.INPUT_PUBLISH === 'true');
  if (publish && !tagged && !env.INPUT_VERSION) {
    const [major, minor, patch] = version.split('.');
    version = `${major}.${minor}.${BigInt(patch) + BigInt(env.GITHUB_RUN_NUMBER)}`;
  }
  const buildTag = publish ? `${version}${preview ? `-preview.${env.GITHUB_RUN_NUMBER}` : ''}` : `${version}-dev.${env.GITHUB_RUN_NUMBER}.${env.GITHUB_SHA.slice(0, 7)}`;
  return {
    version,
    build_tag: buildTag,
    release_tag: `v${buildTag}`,
    publish: String(publish),
    channel: preview ? 'preview' : primary || tagged ? 'stable' : 'development',
    prerelease: String(preview),
    make_latest: String(!preview),
    unlock_packs: String(event !== 'pull_request' && Boolean(env.REPO_KEY)),
  };
}

module.exports = { buildMetadata };

if (require.main === module) {
  const values = buildMetadata(process.env, fs.readFileSync('VERSION', 'utf8'));
  const output = Object.entries(values).map(([key, value]) => `${key}=${value}`).join('\n') + '\n';
  fs.appendFileSync(process.env.GITHUB_OUTPUT, output);
  console.log(`CraftBox ${values.build_tag}; publish=${values.publish}`);
}
