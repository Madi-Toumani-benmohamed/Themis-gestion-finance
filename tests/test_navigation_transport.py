"""Legacy diagnostic transport tests: local fixtures only, never the configured app."""
import http.cookiejar
import importlib.util
import ssl
import unittest
from pathlib import Path
from unittest.mock import patch

SOURCE = Path(__file__).resolve().parents[1] / 'equipe/essais/test_navigation_top.py'
spec = importlib.util.spec_from_file_location('navigation_diagnostic', SOURCE)
diagnostic = importlib.util.module_from_spec(spec)
spec.loader.exec_module(diagnostic)


class NavigationTransportTest(unittest.TestCase):
    def setUp(self):
        self.original = diagnostic.BASE_URL
        diagnostic.BASE_URL = 'https://themis.example.test'

    def tearDown(self):
        diagnostic.BASE_URL = self.original

    def test_file_and_other_origins_are_rejected_before_network(self):
        for url in ('file:///etc/passwd', 'https://other.example.test/api/summary',
                    'https://user:password@themis.example.test/api/summary'):
            with self.subTest(url=url), patch.object(diagnostic.http.client, 'HTTPSConnection') as connection:
                with self.assertRaises(ValueError):
                    diagnostic.fetch(url, http.cookiejar.CookieJar())
                connection.assert_not_called()

    def test_https_verifies_certificates(self):
        with patch.object(diagnostic.http.client, 'HTTPSConnection') as connection:
            response = connection.return_value.getresponse.return_value
            response.status = 200
            response.info.return_value.get_all.return_value = []
            response.read.return_value = b'{"ok":true}'
            value = diagnostic.fetch('https://themis.example.test/api/health', http.cookiejar.CookieJar())
            self.assertEqual(value, {'ok': True})
            context = connection.call_args.kwargs['context']
            self.assertTrue(context.check_hostname)
            self.assertEqual(context.verify_mode, ssl.CERT_REQUIRED)
            connection.return_value.close.assert_called_once()


if __name__ == '__main__':
    unittest.main()
