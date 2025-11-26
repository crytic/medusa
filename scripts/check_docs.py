#!/usr/bin/env python3
"""
Documentation validation script for Medusa.

This script parses Go source code to extract CLI flags, configuration fields,
and cheatcode methods, then validates that they are properly documented.

Validation strategy:
- CLI flags: All flags defined with .Flags() are public-facing
- Cheatcodes: All methods registered with addMethod() are public-facing
  (checks base method name only, e.g., toString covers all toString overloads)
- Config fields: Only fields with `json` tags are validated (user-configurable)

Usage:
    python3 scripts/check_docs.py

Exit codes:
    0 - All documentation checks passed
    1 - One or more documentation checks failed
"""

import os
import re
import sys
from pathlib import Path
from typing import List, Set, Dict


class DocChecker:
    """Main documentation checker class."""

    def __init__(self, repo_root: str):
        self.repo_root = Path(repo_root)
        self.errors: List[str] = []
        self.warnings: List[str] = []

    def check_cli_flags(self) -> None:
        """
        Check that all CLI flags are documented.

        Parses cmd/fuzz_flags.go and cmd/init_flags.go to extract flag names,
        then validates they appear in the corresponding documentation files.
        """
        print("Checking CLI flags documentation...")

        # Check fuzz command flags
        fuzz_flags = self._extract_flags_from_file("cmd/fuzz_flags.go")
        self._validate_flags_documented(fuzz_flags, "fuzz", "docs/src/cli/fuzz.md")

        # Check init command flags
        init_flags = self._extract_flags_from_file("cmd/init_flags.go")
        self._validate_flags_documented(init_flags, "init", "docs/src/cli/init.md")

        print(f"  Found {len(fuzz_flags)} fuzz flags and {len(init_flags)} init flags")

    def _extract_flags_from_file(self, file_path: str) -> Set[str]:
        """Extract flag names from a Go source file."""
        flags = set()
        go_file = self.repo_root / file_path

        if not go_file.exists():
            self.errors.append(f"Source file not found: {file_path}")
            return flags

        content = go_file.read_text()

        # Match patterns like: .Flags().String("flag-name",
        # Handles: String, Int, Bool, Uint64, StringSlice, CountP, etc.
        flag_patterns = [
            r'\.Flags\(\)\.(?:String|Int|Bool|Uint64|StringSlice)\("([^"]+)"',
            r'\.Flags\(\)\.CountP\("([^"]+)",\s*"([^"]+)"',  # CountP has short name
        ]

        for pattern in flag_patterns:
            matches = re.findall(pattern, content)
            for match in matches:
                if isinstance(match, tuple):
                    # CountP returns (long_name, short_name)
                    flags.add(match[0])  # Add long name
                else:
                    flags.add(match)

        return flags

    def _validate_flags_documented(
        self, flags: Set[str], command: str, doc_file: str
    ) -> None:
        """Validate that all flags are documented in the specified doc file."""
        doc_path = self.repo_root / doc_file

        if not doc_path.exists():
            self.errors.append(f"Documentation file not found: {doc_file}")
            return

        doc_content = doc_path.read_text()

        for flag in flags:
            # Check if flag appears in documentation
            # Look for patterns like: --flag-name or `--flag-name`
            if f"--{flag}" not in doc_content and f"`{flag}`" not in doc_content:
                self.errors.append(
                    f"CLI flag '--{flag}' for '{command}' command not documented in {doc_file}"
                )

    def check_cheatcodes(self) -> None:
        """
        Check that all cheatcodes are documented in the interface.

        Parses chain/standard_cheat_code_contract.go to extract cheatcode method names,
        then validates they appear in the interface at docs/src/cheatcodes/cheatcodes_overview.md.

        Note: Uses base method name only (e.g., toString covers all toString overloads).
        The interface is the source of truth - developers may organize documentation however they want.
        """
        print("Checking cheatcodes documentation...")

        cheatcodes = self._extract_cheatcodes()

        # Get unique base method names (e.g., toString from toString(address))
        base_methods = {cheatcode.split("(")[0] for cheatcode in cheatcodes}

        print(f"  Found {len(cheatcodes)} cheatcode methods ({len(base_methods)} unique base methods)")

        # Check cheatcode interface (source of truth)
        self._validate_cheatcodes_in_interface(base_methods)

    def _extract_cheatcodes(self) -> Set[str]:
        """Extract cheatcode method names from standard_cheat_code_contract.go."""
        cheatcodes = set()
        source_file = self.repo_root / "chain/standard_cheat_code_contract.go"

        if not source_file.exists():
            self.errors.append(
                "Cheatcode source file not found: chain/standard_cheat_code_contract.go"
            )
            return cheatcodes

        content = source_file.read_text()

        # Match patterns like: addMethod("methodName", or addMethod(\n    "methodName",
        # \s* matches any whitespace (spaces, tabs, newlines)
        pattern = r'addMethod\(\s*"([^"]+)"'
        matches = re.findall(pattern, content)
        cheatcodes.update(matches)

        return cheatcodes

    def _validate_cheatcodes_in_interface(self, base_methods: Set[str]) -> None:
        """Validate cheatcodes appear in the interface definition."""
        interface_file = self.repo_root / "docs/src/cheatcodes/cheatcodes_overview.md"

        if not interface_file.exists():
            self.errors.append(
                "Cheatcode interface file not found: docs/src/cheatcodes/cheatcodes_overview.md"
            )
            return

        interface_content = interface_file.read_text()

        for method_name in base_methods:
            # Check if method name appears in interface
            # Look for patterns like: function methodName(
            if f"function {method_name}(" not in interface_content:
                self.errors.append(
                    f"Cheatcode '{method_name}' not found in interface at docs/src/cheatcodes/cheatcodes_overview.md"
                )

    def check_config_fields(self) -> None:
        """
        Check that configuration fields are documented.

        Only validates fields with `json` tags - these are user-configurable.
        Fields without json tags are internal-only and automatically excluded.
        """
        print("Checking configuration fields documentation...")

        # Define config files and their corresponding documentation
        config_mappings = [
            (
                "fuzzing/config/config.go",
                ["FuzzingConfig"],
                "docs/src/project_configuration/fuzzing_config.md",
            ),
            (
                "fuzzing/config/config.go",
                ["TestingConfig", "AssertionTestingConfig", "PropertyTestingConfig", "OptimizationTestingConfig"],
                "docs/src/project_configuration/testing_config.md",
            ),
            (
                "chain/config/config.go",
                ["TestChainConfig", "CheatCodeConfig", "ForkConfig"],
                "docs/src/project_configuration/chain_config.md",
            ),
            (
                "compilation/compilation_config.go",
                ["CompilationConfig"],
                "docs/src/project_configuration/compilation_config.md",
            ),
            (
                "fuzzing/config/config.go",
                ["LoggingConfig"],
                "docs/src/project_configuration/logging_config.md",
            ),
        ]

        total_fields = 0
        for source_file, struct_names, doc_file in config_mappings:
            fields = self._extract_config_fields(source_file, struct_names)
            total_fields += len(fields)
            if fields:
                self._validate_config_fields_documented(fields, doc_file)

        print(f"  Found {total_fields} configuration fields across all config structs")

    def _extract_config_fields(
        self, source_file: str, struct_names: List[str]
    ) -> Dict[str, str]:
        """
        Extract configuration fields from Go structs.

        Only extracts fields with `json` tags - these are user-configurable.
        Returns a dict mapping field names (json tags) to struct names.
        """
        fields = {}
        go_file = self.repo_root / source_file

        if not go_file.exists():
            self.warnings.append(f"Config source file not found: {source_file}")
            return fields

        content = go_file.read_text()

        for struct_name in struct_names:
            # Find the struct definition
            struct_pattern = rf"type {struct_name} struct\s*{{([^}}]+)}}"
            struct_match = re.search(struct_pattern, content, re.DOTALL)

            if not struct_match:
                continue

            struct_body = struct_match.group(1)

            # Extract fields with json tags
            # Pattern: FieldName Type `json:"fieldName"`
            # Capture both field name, type, and json tag
            field_pattern = r'(\w+)\s+([^\s`]+)\s*`json:"([^"]+)"'
            field_matches = re.findall(field_pattern, struct_body)

            for go_field_name, field_type, json_field_name in field_matches:
                # Only include exported fields (start with uppercase)
                if not go_field_name[0].isupper():
                    continue

                # Skip fields with omitempty (optional/internal fields not required in user docs)
                if ',omitempty' in json_field_name:
                    continue

                # Strip other JSON tag modifiers like ,omitzero, etc.
                json_field_name = json_field_name.split(',')[0]

                # Skip nested struct fields (fields whose type is another config struct)
                # These are typically documented as sections, not individual fields
                if field_type.endswith('Config') or field_type in ['TestingConfig', 'CheatCodeConfig', 'ForkConfig']:
                    continue

                fields[json_field_name] = struct_name

        return fields

    def _validate_config_fields_documented(
        self, fields: Dict[str, str], doc_file: str
    ) -> None:
        """Validate that config fields are documented."""
        doc_path = self.repo_root / doc_file

        if not doc_path.exists():
            self.errors.append(f"Config documentation file not found: {doc_file}")
            return

        doc_content = doc_path.read_text()

        for field_name, struct_name in fields.items():
            # Check if field appears in documentation
            # Look for the field name in backticks or as a heading
            if f"`{field_name}`" not in doc_content and f"## {field_name}" not in doc_content.lower():
                self.errors.append(
                    f"Config field '{field_name}' from {struct_name} not documented in {doc_file}"
                )

    def report(self) -> None:
        """Print final report and exit with appropriate code."""
        print("\n" + "=" * 70)

        if self.warnings:
            print("Warnings:")
            for warning in self.warnings:
                print(f"  ⚠️  {warning}")
            print()

        if self.errors:
            print("Documentation validation FAILED:")
            print()
            for error in self.errors:
                print(f"  ❌ {error}")
            print()
            print(f"Total errors: {len(self.errors)}")
            print("=" * 70)
            sys.exit(1)
        else:
            print("✅ All documentation checks passed!")
            print("=" * 70)
            sys.exit(0)


def main():
    """Main entry point."""
    # Use current directory as repo root
    repo_root = os.getcwd()

    print("=" * 70)
    print("Medusa Documentation Validation")
    print("=" * 70)
    print()

    checker = DocChecker(repo_root)

    try:
        checker.check_cli_flags()
        checker.check_cheatcodes()
        checker.check_config_fields()
    except Exception as e:
        print(f"\n❌ Unexpected error during validation: {e}")
        import traceback
        traceback.print_exc()
        sys.exit(1)

    checker.report()


if __name__ == "__main__":
    main()
