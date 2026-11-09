import argparse
import hashlib
import importlib.util
import json
import pathlib
import shutil
import tempfile
import time
import urllib.request
import zipfile


def digest(file):
    with file.open('rb') as source:
        return hashlib.file_digest(source, 'sha256').hexdigest()


def download(asset, destination):
    if not destination.exists():
        destination.parent.mkdir(parents=True, exist_ok=True)
        temporary = destination.with_name(destination.name + '.download')
        for attempt in range(3):
            try:
                request = urllib.request.Request(asset['url'], headers={'User-Agent': 'CraftBox-CI-assets'})
                with urllib.request.urlopen(request, timeout=60) as response, temporary.open('wb') as output:
                    received = 0
                    while chunk := response.read(1 << 20):
                        received += len(chunk)
                        if received > 512 << 20:
                            raise ValueError('Asset download exceeds 512 MiB')
                        output.write(chunk)
                if digest(temporary) != asset['sha256'].lower():
                    raise ValueError(f'SHA256 mismatch for {destination.name}')
                temporary.replace(destination)
                break
            except (OSError, ValueError):
                temporary.unlink(missing_ok=True)
                if attempt == 2:
                    raise
                time.sleep(attempt + 1)
    if digest(destination) != asset['sha256'].lower():
        raise ValueError(f'SHA256 mismatch for cached {destination.name}')
    return destination


def extract(archive, destination, marker=None):
    with zipfile.ZipFile(archive) as source:
        total = 0
        entries = source.infolist()
        if len(entries) > 50000:
            raise ValueError('Asset archive exceeds 50000 entries')
        for entry in entries:
            name = pathlib.PurePosixPath(entry.orig_filename)
            parts = name.parts
            if marker:
                if marker not in parts:
                    continue
                parts = parts[parts.index(marker) + 1:]
            if not parts or entry.is_dir():
                continue
            if name.is_absolute() or any(part in ('.', '..') or ':' in part or chr(92) in part for part in parts):
                raise ValueError(f'Invalid asset archive path: {entry.filename}')
            if (entry.external_attr >> 16) & 0o170000 == 0o120000:
                raise ValueError(f'Asset archive contains a symbolic link: {entry.filename}')
            total += entry.file_size
            if total > 2 << 30 or entry.file_size > 256 << 20:
                raise ValueError('Asset archive exceeds its extraction budget')
            target = destination.joinpath(*parts)
            target.parent.mkdir(parents=True, exist_ok=True)
            with source.open(entry) as input_file, target.open('wb') as output:
                while chunk := input_file.read(1 << 20):
                    output.write(chunk)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--manifest', type=pathlib.Path, default=pathlib.Path(__file__).with_name('ci-assets.json'))
    parser.add_argument('--cache', type=pathlib.Path, default=pathlib.Path('tmp/ci-assets-download'))
    parser.add_argument('--output', type=pathlib.Path, default=pathlib.Path('assets/vanilla-3d'))
    args = parser.parse_args()
    manifest = json.loads(args.manifest.read_text(encoding='utf8'))
    files = {key: download(manifest[key], args.cache / manifest[key]['file']) for key in ('bds', 'sample', 'font', 'tenFont')}
    spec = importlib.util.spec_from_file_location('prepare_assets', pathlib.Path(__file__).with_name('prepare-3d-assets.py'))
    prepare = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(prepare)
    with tempfile.TemporaryDirectory(prefix='craftbox-ci-assets-') as temporary:
        root = pathlib.Path(temporary)
        bds, sample = root / 'bds', root / 'sample'
        extract(files['bds'], bds)
        extract(files['sample'], sample, 'resource_pack')
        if not (bds / 'resource_packs/vanilla/blocks.json').is_file() or not (sample / 'textures/terrain_texture.json').is_file():
            raise ValueError('Official archives are missing required vanilla resources')
        provenance = {'source': 'Mojang BDS block definitions + Mojang bedrock-samples textures',
                      'bdsExecutableSHA256': digest(bds / 'bedrock_server.exe'),
                      'bdsVersion': manifest['version'], 'sampleCommit': manifest['sampleCommit'],
                      'sampleArchiveSHA256': manifest['sample']['sha256'],
                      'bdsArchiveSHA256': manifest['bds']['sha256'],
                      'minecraftTenSHA256': manifest['tenFont']['sha256']}
        prepare.import_assets(bds, sample, args.output, provenance, files['font'])
        shutil.copy2(files['tenFont'], args.output / 'font/minecraft-ten.ttf')
    print(f'Prepared verified Minecraft {manifest["version"]} assets in {args.output}')


if __name__ == '__main__':
    main()
