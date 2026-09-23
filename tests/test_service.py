import os
import unittest

os.environ.setdefault("BUDDY_DATABASE_URL", "postgresql://buddy:unused@localhost:5432/buddy")

from buddy_mcp.validation import EMBEDDING_DIM, checked_embedding, checked_text, normalize_project_url


class ServiceValidationTests(unittest.TestCase):
    def test_normalizes_https_and_ssh_remotes_to_same_project_key(self):
        expected = "https://github.com/mcp-runtime/buddy"
        self.assertEqual(normalize_project_url("git@github.com:MCP-Runtime/Buddy.git"), expected)
        self.assertEqual(normalize_project_url("https://github.com/MCP-Runtime/Buddy/"), expected)

    def test_rejects_urls_that_are_not_github_repository_roots(self):
        bad_urls = (
            "https://github.com/mcp-runtime/buddy/tree/main",
            "https://example.com/mcp-runtime/buddy",
            "git@gitlab.com:team/project.git",
        )
        for value in bad_urls:
            with self.subTest(value=value), self.assertRaises(ValueError):
                normalize_project_url(value)

    def test_rejects_credentials_in_work_notes(self):
        with self.assertRaisesRegex(ValueError, "credential"):
            checked_text("summary", "Found password=supersecretvalue in a log")

    def test_accepts_valid_embedding_and_rejects_invalid_vectors(self):
        vector = checked_embedding([0.0] * EMBEDDING_DIM)
        self.assertTrue(vector.startswith("[0,0,0,"))
        with self.assertRaisesRegex(ValueError, "exactly 1536"):
            checked_embedding([0.0])
        with self.assertRaisesRegex(ValueError, "finite"):
            checked_embedding([float("nan")] + [0.0] * (EMBEDDING_DIM - 1))


if __name__ == "__main__":
    unittest.main()
