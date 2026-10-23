const { test } = require('node:test');
const assert = require('node:assert/strict');
const { buildMetadata } = require('./ci-metadata.cjs');

const base = {
  GITHUB_EVENT_NAME: 'push',
  GITHUB_REF: 'refs/heads/master',
  GITHUB_SHA: 'abcdef0123456789abcdef0123456789abcdef01',
  GITHUB_RUN_NUMBER: '17',
  DEFAULT_BRANCH: 'master',
};

test('default branch publishes a versioned build after a tagless history reset', () => {
  const result = buildMetadata(base, '1.0.0');
  assert.equal(result.version, '1.0.0');
  assert.equal(result.build_tag, '1.0.0-17-abcdef0');
  assert.equal(result.release_tag, 'r1.0.0-17-abcdef0');
  assert.equal(result.release_tag, `r${result.build_tag}`);
  assert.equal(result.publish, 'true');
});

test('feature branches and pull requests produce development artifacts only', () => {
  for (const override of [
    { GITHUB_REF: 'refs/heads/feature' },
    { GITHUB_EVENT_NAME: 'pull_request', INPUT_PUBLISH: 'true', REPO_KEY: 'private' },
  ]) {
    const result = buildMetadata({ ...base, ...override }, '1.0.0');
    assert.equal(result.publish, 'false');
    assert.ok(result.build_tag.endsWith('-dev'));
    if (override.GITHUB_EVENT_NAME === 'pull_request') {
      assert.equal(result.unlock_packs, 'false');
    }
  }
});

test('a release tag supplies its own version', () => {
  const result = buildMetadata({ ...base, GITHUB_REF: 'refs/tags/r2.3.4' }, '1.0.0');
  assert.equal(result.version, '2.3.4');
  assert.equal(result.release_tag, 'r2.3.4');
  assert.equal(result.build_tag, '2.3.4');
  assert.equal(result.publish, 'true');
});

test('manual builds can choose a version and disable publication', () => {
  const result = buildMetadata({
    ...base,
    GITHUB_EVENT_NAME: 'workflow_dispatch',
    INPUT_VERSION: '10.2.3',
    INPUT_PUBLISH: 'false',
  }, '1.0.0');
  assert.equal(result.version, '10.2.3');
  assert.equal(result.publish, 'false');
  assert.ok(result.build_tag.endsWith('-dev'));
});

test('invalid versions and GitHub output injection are rejected', () => {
  for (const version of ['0.0', '01.0.0', '1.0.0\npublish=true', '1.0.0-beta']) {
    assert.throws(() => buildMetadata({ ...base, INPUT_VERSION: version }, '1.0.0'));
  }
});
