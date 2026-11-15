const { test } = require('node:test');
const assert = require('node:assert/strict');
const { buildMetadata } = require('./ci-metadata.cjs');
const { comparePublishedVersions, publicationAllowed } = require('./release-policy.cjs');

const base = {
  GITHUB_EVENT_NAME: 'push',
  GITHUB_REF: 'refs/heads/master',
  GITHUB_SHA: 'abcdef0123456789abcdef0123456789abcdef01',
  GITHUB_RUN_NUMBER: '17',
  DEFAULT_BRANCH: 'master',
};

test('default branch publishes a versioned build after a tagless history reset', () => {
  const result = buildMetadata(base, '1.0.0');
  assert.equal(result.version, '1.0.17');
  assert.equal(result.build_tag, '1.0.17');
  assert.equal(result.release_tag, 'v1.0.17');
  assert.equal(result.publish, 'true');
});

test('feature branches and pull requests produce development artifacts only', () => {
  for (const override of [
    { GITHUB_REF: 'refs/heads/feature' },
    { GITHUB_EVENT_NAME: 'pull_request', INPUT_PUBLISH: 'true', REPO_KEY: 'private' },
  ]) {
    const result = buildMetadata({ ...base, ...override }, '1.0.0');
    assert.equal(result.publish, 'false');
    assert.equal(result.build_tag, '1.0.0-dev.17.abcdef0');
    if (override.GITHUB_EVENT_NAME === 'pull_request') {
      assert.equal(result.unlock_packs, 'false');
    }
  }
});

test('a release tag supplies its own version', () => {
  const result = buildMetadata({ ...base, GITHUB_REF: 'refs/tags/r2.3.4' }, '1.0.0');
  assert.equal(result.version, '2.3.4');
  assert.equal(result.release_tag, 'v2.3.4');
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
  assert.equal(result.build_tag, '10.2.3-dev.17.abcdef0');
});

test('standard v tags publish their exact version', () => {
  const result = buildMetadata({ ...base, GITHUB_REF: 'refs/tags/v2.3.4' }, '1.0.0');
  assert.equal(result.version, '2.3.4');
  assert.equal(result.release_tag, 'v2.3.4');
  assert.equal(result.build_tag, '2.3.4');
  assert.equal(result.publish, 'true');
});

test('automatic releases advance the base patch and manual versions stay exact', () => {
  assert.equal(buildMetadata(base, '2.3.4').version, '2.3.21');
  assert.equal(buildMetadata({ ...base, GITHUB_EVENT_NAME: 'workflow_dispatch', INPUT_VERSION: '2.4.0', INPUT_PUBLISH: 'true' }, '1.0.0').build_tag, '2.4.0');
});

test('invalid versions and GitHub output injection are rejected', () => {
  for (const version of ['0.0', '01.0.0', '1.0.0\npublish=true', '1.0.0-beta']) {
    assert.throws(() => buildMetadata({ ...base, INPUT_VERSION: version }, '1.0.0'));
  }
});

test('preview branch publishes a prerelease without replacing stable latest', () => {
  const result = buildMetadata({ ...base, GITHUB_REF: 'refs/heads/preview' }, '1.0.0');
  assert.equal(result.version, '1.0.17');
  assert.equal(result.build_tag, '1.0.17-preview.17');
  assert.equal(result.release_tag, 'v1.0.17-preview.17');
  assert.equal(result.publish, 'true');
  assert.equal(result.prerelease, 'true');
  assert.equal(result.make_latest, 'false');
  assert.equal(result.channel, 'preview');
});

test('master stays stable even when the repository default branch changes', () => {
  const stable = buildMetadata({ ...base, DEFAULT_BRANCH: 'preview' }, '1.0.0');
  assert.equal(stable.channel, 'stable');
  assert.equal(stable.publish, 'true');
  assert.equal(stable.prerelease, 'false');
  assert.equal(stable.make_latest, 'true');
  const other = buildMetadata({ ...base, GITHUB_REF:'refs/heads/develop', DEFAULT_BRANCH:'develop' }, '1.0.0');
  assert.equal(other.publish, 'false');
});

test('manual preview publication uses preview versions and pull requests never publish', () => {
  const manual = buildMetadata({ ...base, GITHUB_REF:'refs/heads/preview', GITHUB_EVENT_NAME:'workflow_dispatch', INPUT_VERSION:'2.0.0', INPUT_PUBLISH:'true' }, '1.0.0');
  assert.equal(manual.build_tag, '2.0.0-preview.17');
  assert.equal(manual.prerelease, 'true');
  for (const branch of ['master','preview']) {
    assert.equal(buildMetadata({ ...base, GITHUB_REF:`refs/heads/${branch}`, GITHUB_EVENT_NAME:'pull_request', INPUT_PUBLISH:'true' }, '1.0.0').publish, 'false');
  }
});

test('release ordering handles preview suffixes and legacy builds numerically', () => {
  for (const [a,b,want] of [
    ['v1.0.20','v1.0.10',1], ['v1.0.20-preview.20','v1.0.20',-1],
    ['v1.0.20-preview.100','v1.0.20-preview.99',1],
    ['r1.0.0-100-abcdef0','r1.0.0-99-abcdef0',1],
    ['v1.0.48','r1.0.0-47-abcdef0',1], ['v1.0.20','v1.0.20',0],
  ]) assert.equal(comparePublishedVersions(a,b),want);
});

function policyFixture({latest='v1.0.20', existing, current='commit', ancestor='identical', pages=[]}={}) {
  return {
    context:{repo:{owner:'owner',repo:'repo'},sha:'commit',ref:'refs/heads/master'},
    core:{notice:()=>{}},tag:'v1.0.21',channel:'stable',
    github:{rest:{repos:{
      getCommit:async()=>({data:{sha:current}}),
      compareCommits:async()=>({data:{status:ancestor}}),
      getReleaseByTag:async()=>{if(existing)return {data:existing};throw Object.assign(Error('missing'),{status:404})},
      getLatestRelease:async()=>{if(latest)return {data:{tag_name:latest}};throw Object.assign(Error('missing'),{status:404})},
      listReleases:async({page})=>({data:pages[page-1]||[]}),
    }}},
  };
}

test('older explicit versions cannot roll Stable Latest back', async () => {
  await assert.rejects(publicationAllowed({...policyFixture(),tag:'v1.0.10'}),/must be newer/);
  assert.equal(await publicationAllowed(policyFixture()),true);
});

test('published reruns keep release files unchanged and drafts can be retried', async () => {
  assert.equal(await publicationAllowed(policyFixture({existing:{draft:false}})),false);
  assert.equal(await publicationAllowed(policyFixture({existing:{draft:true}})),true);
});

test('superseded branches and stable tags outside master cannot publish', async () => {
  assert.equal(await publicationAllowed(policyFixture({current:'newer'})),false);
  const tagged=policyFixture({ancestor:'behind'});
  tagged.context.ref='refs/tags/v1.0.21';
  await assert.rejects(publicationAllowed(tagged),/merged into master/);
});

test('preview ordering searches past a full stable release page', async () => {
  const first=Array.from({length:100},()=>({tag_name:'v9.0.0',prerelease:false}));
  const fixture=policyFixture({pages:[first,[{tag_name:'v1.0.30-preview.30',prerelease:true}]]});
  fixture.channel='preview';fixture.tag='v1.0.29-preview.29';
  await assert.rejects(publicationAllowed(fixture),/must be newer/);
  fixture.tag='v1.0.31-preview.31';
  assert.equal(await publicationAllowed(fixture),true);
});
