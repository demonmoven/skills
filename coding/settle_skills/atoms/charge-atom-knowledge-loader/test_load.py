import io
import json
import os
import sys
import tempfile
import unittest
from unittest import mock

sys.path.insert(0, os.path.dirname(__file__))
import load as L

YAML_TEXT = '''version: 1
sources:
  - key: test_doc
    title: Test
    url: https://example.com/x
    cache_path: {cache}
    loader: lark-docs-skill
    refresh_policy: cross_session
    required: true
    scope: [charge]
'''


class T(unittest.TestCase):
    def test_force_fetch_writes_cache(self):
        with tempfile.TemporaryDirectory() as d:
            cfg = os.path.join(d, "k.yaml")
            cache = os.path.join(d, "c.md")
            with open(cfg, "w") as f:
                f.write(YAML_TEXT.format(cache=cache))

            def fake(url, dest):
                with open(dest, "w") as f:
                    f.write("hello")
                return 5

            with mock.patch.dict(L.LOADERS, {"lark-docs-skill": fake}):
                with mock.patch.object(
                    sys, "argv", ["load.py", "--config", cfg, "--scope", "charge", "--refresh", "force"]
                ):
                    buf = io.StringIO()
                    with mock.patch("sys.stdout", buf):
                        with self.assertRaises(SystemExit) as cm:
                            L.main()
            self.assertEqual(cm.exception.code, 0)
            self.assertTrue(os.path.exists(cache))
            out = json.loads(buf.getvalue())
            self.assertEqual(len(out.get("loaded") or []), 1)

    def test_cache_only_miss_aborts(self):
        with tempfile.TemporaryDirectory() as d:
            cfg = os.path.join(d, "k.yaml")
            cache = os.path.join(d, "missing.md")
            with open(cfg, "w") as f:
                f.write(YAML_TEXT.format(cache=cache))
            with mock.patch.object(
                sys, "argv", ["load.py", "--config", cfg, "--scope", "charge", "--refresh", "cache-only"]
            ):
                buf = io.StringIO()
                with mock.patch("sys.stdout", buf):
                    with self.assertRaises(SystemExit) as cm:
                        L.main()
            self.assertEqual(cm.exception.code, 1)


if __name__ == "__main__":
    unittest.main()
