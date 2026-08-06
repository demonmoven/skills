import os
import stat
import tempfile
import unittest
from pathlib import Path
from unittest import mock

import uploader


class EnsureTitanPassportIdTest(unittest.TestCase):
    def tearDown(self):
        uploader.ensure_titan_passport_id.cache_clear()
        os.environ.pop("DATA_AGENT_TITAN_PASSPORT_ID", None)

    def test_from_plain_cli_output(self):
        os.environ.pop("DATA_AGENT_TITAN_PASSPORT_ID", None)
        uploader.ensure_titan_passport_id.cache_clear()
        with tempfile.TemporaryDirectory() as tmp:
            script = Path(tmp) / "get-titan-id"
            script.write_text("#!/usr/bin/env bash\nprintf 'titan-passport-12345'\n")
            script.chmod(script.stat().st_mode | stat.S_IXUSR)
            with mock.patch.dict(os.environ, {"PATH": f"{tmp}:{os.environ.get('PATH', '')}"}, clear=False):
                self.assertTrue(uploader.ensure_titan_passport_id())
                self.assertEqual(os.environ["DATA_AGENT_TITAN_PASSPORT_ID"], "titan-passport-12345")

    def test_from_json_cli_output(self):
        os.environ.pop("DATA_AGENT_TITAN_PASSPORT_ID", None)
        uploader.ensure_titan_passport_id.cache_clear()
        with tempfile.TemporaryDirectory() as tmp:
            script = Path(tmp) / "get-titan-id"
            script.write_text("#!/usr/bin/env bash\nprintf '{\"data\":{\"passportId\":\"json-passport-67890\"}}'\n")
            script.chmod(script.stat().st_mode | stat.S_IXUSR)
            with mock.patch.dict(os.environ, {"PATH": f"{tmp}:{os.environ.get('PATH', '')}"}, clear=False):
                self.assertTrue(uploader.ensure_titan_passport_id())
                self.assertEqual(os.environ["DATA_AGENT_TITAN_PASSPORT_ID"], "json-passport-67890")

    def test_missing_cli(self):
        os.environ.pop("DATA_AGENT_TITAN_PASSPORT_ID", None)
        uploader.ensure_titan_passport_id.cache_clear()
        with mock.patch.dict(os.environ, {"PATH": ""}, clear=False):
            self.assertFalse(uploader.ensure_titan_passport_id())
            self.assertNotIn("DATA_AGENT_TITAN_PASSPORT_ID", os.environ)


if __name__ == "__main__":
    unittest.main()
