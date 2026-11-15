function comparePublishedVersions(a, b) {
  const parse = value => {
    const match = /^[vr]?(\d+)\.(\d+)\.(\d+)(?:(-preview\.(\d+))|(-(\d+)-[a-f0-9]{7,40}))?$/.exec(value);
    if (!match) return null;
    return [BigInt(match[1]), BigInt(match[2]), BigInt(match[3]), match[6] ? 0n : match[4] ? 1n : 2n, BigInt(match[5] || match[7] || 0)];
  };
  const left = parse(a), right = parse(b);
  if (!left || !right) return null;
  for (let i = 0; i < left.length; i++) {
    if (left[i] > right[i]) return 1;
    if (left[i] < right[i]) return -1;
  }
  return 0;
}

async function publicationAllowed({ github, context, core, tag, channel }) {
  const repo = { owner: context.repo.owner, repo: context.repo.repo };
  const { data: current } = await github.rest.repos.getCommit({ ...repo, ref: context.ref });
  if (current.sha !== context.sha) {
    core.notice('Skipping a superseded CraftBox build.');
    return false;
  }
  if (context.ref.startsWith('refs/tags/')) {
    const { data } = await github.rest.repos.compareCommits({ ...repo, base: context.sha, head: 'master' });
    if (!['ahead', 'identical'].includes(data.status)) {
      throw new Error('Stable release tags must point to commits already merged into master.');
    }
  }
  try {
    const { data: existing } = await github.rest.repos.getReleaseByTag({ ...repo, tag });
    const { data: tagged } = await github.rest.repos.getCommit({ ...repo, ref: `refs/tags/${tag}` });
    if (tagged.sha !== context.sha) throw new Error(`Release tag ${tag} already belongs to another commit.`);
    if (!existing.draft) {
      core.notice(`Release ${tag} is already published; keeping its files unchanged.`);
      return false;
    }
  } catch (error) {
    if (error.status !== 404) throw error;
  }
  let latest;
  if (channel === 'stable') {
    try {
      latest = (await github.rest.repos.getLatestRelease(repo)).data;
    } catch (error) {
      if (error.status !== 404) throw error;
    }
  } else {
    for (let page = 1; ; page++) {
      const { data } = await github.rest.repos.listReleases({ ...repo, per_page: 100, page });
      for (const release of data) {
        if (release.draft || !release.prerelease || !release.tag_name.includes('-preview.')) continue;
        const order = latest ? comparePublishedVersions(release.tag_name, latest.tag_name) : 1;
        if (order === null) throw new Error(`Cannot compare published Preview version ${release.tag_name}.`);
        if (order > 0) latest = release;
      }
      if (data.length < 100) break;
    }
  }
  if (latest) {
    const order = comparePublishedVersions(tag, latest.tag_name);
    if (order === null || order <= 0) {
      throw new Error(`Release ${tag} must be newer than ${latest.tag_name}. Update VERSION or choose a newer version.`);
    }
  }
  return true;
}

module.exports = { comparePublishedVersions, publicationAllowed };
