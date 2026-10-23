"""Import local BDS definitions and Mojang's official vanilla texture assets.

Python 3.10+, no third-party packages. Nothing is downloaded during capture.
Mojang assets are local runtime data and must not be committed to this repo.
"""
import argparse
import hashlib
import json
import os
import pathlib
import re
import shutil
import subprocess
import time
import urllib.request
import zipfile


def version_key(path):
    return tuple(int(n) for n in re.findall(r'\d+', path.name))


def load_jsonc(content):
    strings = r'"(?:[^"\\]|\\.)*"'
    content = re.sub(f'({strings})|//[^\n]*|/\\*.*?\\*/',
                     lambda match: match.group(1) or ' ', content, flags=re.S)
    content = re.sub(f'({strings})|,(\\s*[}}\\]])',
                     lambda match: match.group(1) or match.group(2), content)
    return json.loads(content)


def import_assets(bds, sample, output, provenance):
    output.mkdir(parents=True, exist_ok=True)
    shutil.copytree(sample, output, dirs_exist_ok=True)
    definitions = {}
    for pack in [sample] + sorted((bds/'resource_packs').glob('vanilla*'), key=version_key):
        file = pack/'blocks.json'
        if file.is_file():
            definitions.update(load_jsonc(file.read_text(encoding='utf-8-sig')))
    (output/'blocks.json').write_text(json.dumps(definitions, ensure_ascii=False), encoding='utf8')
    (output/'provenance.json').write_text(json.dumps(provenance, indent=2), encoding='utf8')


def probe(bds, destination):
    """Run an isolated BDS, generate a loaded fixture, save, and always stop it."""
    if destination.exists():
        raise RuntimeError(f'Probe directory already exists: {destination}; choose a fresh path')
    destination.mkdir(parents=True)
    for name in ['bedrock_server.exe', 'bedrock_server', 'behavior_packs', 'resource_packs', 'definitions', 'data', 'config', 'allowlist.json', 'permissions.json']:
        src, dst = bds/name, destination/name
        if src.is_dir():
            shutil.copytree(src, dst)
        elif src.is_file():
            shutil.copy2(src, dst)
    (destination/'server.properties').write_text(
        'server-name=CraftBox 3D test\nlevel-name=3d-fixture\nonline-mode=false\n'
        'server-port=19942\nserver-portv6=19943\nenable-lan-visibility=false\n'
        'view-distance=4\ntick-distance=4\nallow-cheats=true\n'
        'gamemode=creative\nlevel-type=FLAT\nmax-players=1\ntransport=nethernet\n', encoding='utf8')
    executable = destination/('bedrock_server.exe' if os.name == 'nt' else 'bedrock_server')
    environment = dict(os.environ)
    if os.name != 'nt':
        environment['LD_LIBRARY_PATH'] = str(destination)
    with (destination/'probe.log').open('w', encoding='utf8') as log:
        proc = subprocess.Popen([str(executable.resolve())], cwd=destination, stdin=subprocess.PIPE,
                                stdout=log, stderr=subprocess.STDOUT, env=environment,
                                creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)
        try:
            deadline = time.monotonic()+60
            while time.monotonic() < deadline:
                if proc.poll() is not None:
                    raise RuntimeError('BDS exited before startup; see probe.log')
                if 'Server started.' in (destination/'probe.log').read_text(encoding='utf8', errors='replace'):
                    break
                time.sleep(.5)
            else:
                raise RuntimeError('BDS startup timed out')
            commands = [
                'tickingarea add 0 0 0 31 128 31 scene3d true',
                'fill 0 -60 0 31 -60 15 stone',
                'setblock 3 -59 3 oak_stairs', 'setblock 5 -59 3 glass',
                'setblock 7 -59 3 water', 'setblock 9 -59 3 oak_slab',
                'save hold', 'save query', 'save resume', 'stop']
            for command in commands:
                proc.stdin.write((command+'\n').encode())
                proc.stdin.flush()
                time.sleep(4 if command.startswith('tickingarea') else 1)
            proc.wait(timeout=25)
        finally:
            if proc.poll() is None:
                proc.kill()
                proc.wait()
    result = (destination/'probe.log').read_text(encoding='utf8', errors='replace')
    if proc.returncode != 0 or 'Cannot place' in result or 'Syntax error' in result:
        raise RuntimeError(f'BDS fixture failed; see {destination / "probe.log"}')
    return result


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--bds', type=pathlib.Path, required=True, help='Official extracted BDS directory')
    parser.add_argument('--sample', type=pathlib.Path, help='Existing official resource_pack directory')
    parser.add_argument('--sample-ref', default='main', help='Mojang bedrock-samples commit/tag; resolved to immutable SHA')
    parser.add_argument('--output', type=pathlib.Path, default=pathlib.Path('assets/vanilla-3d'))
    parser.add_argument('--probe', type=pathlib.Path, help='Fresh directory for isolated BDS test server')
    args = parser.parse_args()
    if not (args.bds/'resource_packs'/'vanilla'/'blocks.json').is_file():
        parser.error('--bds must contain official resource_packs/vanilla/blocks.json')
    provenance = {'source': 'Mojang BDS block definitions + Mojang bedrock-samples textures',
                  'bdsExecutableSHA256': hashlib.sha256((args.bds/('bedrock_server.exe' if os.name == 'nt' else 'bedrock_server')).read_bytes()).hexdigest()}
    if args.probe:
        log = probe(args.bds.resolve(), args.probe.resolve())
        provenance['bdsProbeVersion'] = re.search(r'Version: ([\d.]+)', log).group(1)
    if args.sample:
        sample = args.sample
        provenance['sampleSource'] = 'local official resource_pack'
    else:
        cache = pathlib.Path('tmp/scene-assets-download')
        cache.mkdir(parents=True, exist_ok=True)
        request = urllib.request.Request(f'https://api.github.com/repos/Mojang/bedrock-samples/commits/{args.sample_ref}', headers={'User-Agent': 'CraftBox-asset-import'})
        with urllib.request.urlopen(request, timeout=60) as response:
            commit = json.load(response)['sha']
        provenance['sampleCommit'] = commit
        url = f'https://codeload.github.com/Mojang/bedrock-samples/zip/{commit}'
        archive = cache/f'{commit}.zip'
        if not archive.exists():
            urllib.request.urlretrieve(url, archive)
        provenance['sampleArchiveSHA256'] = hashlib.sha256(archive.read_bytes()).hexdigest()
        sample = cache/commit
        with zipfile.ZipFile(archive) as pack:
            for entry in pack.infolist():
                if '/resource_pack/' not in entry.filename or entry.is_dir():
                    continue
                relative = pathlib.PurePosixPath(entry.filename.split('/resource_pack/', 1)[1])
                if relative.is_absolute() or '..' in relative.parts:
                    raise RuntimeError('Invalid archive path')
                dest = sample/pathlib.Path(*relative.parts)
                dest.parent.mkdir(parents=True, exist_ok=True)
                dest.write_bytes(pack.read(entry))
    if not (sample/'textures'/'terrain_texture.json').is_file():
        parser.error('Official sample pack is missing textures/terrain_texture.json')
    import_assets(args.bds, sample, args.output, provenance)
    print(json.dumps(provenance, indent=2))
    print(f'Imported assets: {args.output.resolve()}')


if __name__ == '__main__':
    main()
