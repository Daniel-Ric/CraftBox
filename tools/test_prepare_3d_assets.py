import importlib.util
import pathlib
import tempfile
import unittest
import json

spec = importlib.util.spec_from_file_location('prepare', pathlib.Path(__file__).with_name('prepare-3d-assets.py'))
prepare = importlib.util.module_from_spec(spec)
spec.loader.exec_module(prepare)

class ImportTests(unittest.TestCase):
    def test_jsonc_preserves_strings_and_removes_comments_and_trailing_commas(self):
        value = prepare.load_jsonc('{ // comment\n "url": "https://example.test/a,}", "items": [1, 2,], /* note */ }')
        self.assertEqual(value, {'url': 'https://example.test/a,}', 'items': [1, 2]})
    def test_bds_version_overlays_and_sample_textures_survive(self):
        with tempfile.TemporaryDirectory() as tmp:
            root = pathlib.Path(tmp)
            sample, bds, output = root/'sample', root/'bds', root/'output'
            (sample/'textures').mkdir(parents=True)
            (sample/'blocks.json').write_text(json.dumps({'stone': {'textures': 'old'}}))
            (sample/'textures'/'stone.png').write_bytes(b'fixture pixels')
            for version, texture in [('vanilla', 'base'), ('vanilla_1.26.9', 'middle'), ('vanilla_1.26.52', 'new')]:
                pack = bds/'resource_packs'/version
                pack.mkdir(parents=True)
                (pack/'blocks.json').write_text(json.dumps({'stone': {'textures': texture}}))
            prepare.import_assets(bds, sample, output, {'source': 'test'})
            self.assertEqual(json.loads((output/'blocks.json').read_text())['stone']['textures'], 'new')
            self.assertEqual((output/'textures'/'stone.png').read_bytes(), b'fixture pixels')
            self.assertEqual(json.loads((output/'provenance.json').read_text())['source'], 'test')

if __name__ == '__main__':
    unittest.main()
