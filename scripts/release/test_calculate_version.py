#!/usr/bin/env python3
import unittest
from calculate_version import (
    parse_semver,
    format_semver,
    determine_bump,
    generate_changelog,
    RE_CONVENTIONAL,
)

class TestCalculateVersion(unittest.TestCase):
    def test_parse_semver(self):
        self.assertEqual(parse_semver("v1.2.3"), (1, 2, 3))
        self.assertEqual(parse_semver("0.1.0"), (0, 1, 0))
        self.assertEqual(parse_semver("v10.20.30"), (10, 20, 30))
        with self.assertRaises(ValueError):
            parse_semver("v1.2")

    def test_format_semver(self):
        self.assertEqual(format_semver(1, 0, 0), "v1.0.0")
        self.assertEqual(format_semver(2, 5, 12), "v2.5.12")

    def test_conventional_regex(self):
        m1 = RE_CONVENTIONAL.match("feat(cli): add mem version subcommand")
        self.assertIsNotNone(m1)
        self.assertEqual(m1.group("type"), "feat")
        self.assertEqual(m1.group("scope"), "cli")
        self.assertIsNone(m1.group("breaking"))

        m2 = RE_CONVENTIONAL.match("fix: resolve nil pointer exception")
        self.assertIsNotNone(m2)
        self.assertEqual(m2.group("type"), "fix")
        self.assertIsNone(m2.group("scope"))

        m3 = RE_CONVENTIONAL.match("feat(api)!: remove deprecated endpoints")
        self.assertIsNotNone(m3)
        self.assertEqual(m3.group("breaking"), "!")

    def test_determine_bump(self):
        # Breaking change
        self.assertEqual(determine_bump([{"type": "fix", "is_breaking": True}]), "major")
        # Minor bump
        self.assertEqual(determine_bump([
            {"type": "fix", "is_breaking": False},
            {"type": "feat", "is_breaking": False},
        ]), "minor")
        # Patch bump
        self.assertEqual(determine_bump([
            {"type": "fix", "is_breaking": False},
            {"type": "perf", "is_breaking": False},
        ]), "patch")
        # None bump (only docs/chore)
        self.assertEqual(determine_bump([
            {"type": "docs", "is_breaking": False},
            {"type": "chore", "is_breaking": False},
        ]), "none")

    def test_generate_changelog(self):
        commits = [
            {
                "sha": "1234567",
                "full_sha": "1234567890abcdef",
                "subject": "add new command",
                "scope": "cli",
                "type": "feat",
                "is_breaking": False,
            },
            {
                "sha": "7654321",
                "full_sha": "7654321098fedcba",
                "subject": "handle connection timeout",
                "scope": "db",
                "type": "fix",
                "is_breaking": False,
            }
        ]
        changelog = generate_changelog(commits, "v1.1.0")
        self.assertIn("## v1.1.0", changelog)
        self.assertIn("### ✨ Features", changelog)
        self.assertIn("### 🐛 Bug Fixes & Improvements", changelog)
        self.assertIn("**cli**: add new command", changelog)
        self.assertIn("**db**: handle connection timeout", changelog)

if __name__ == "__main__":
    unittest.main()
