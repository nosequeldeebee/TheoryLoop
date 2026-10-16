"""The ledger: every test any agent runs is recorded.

Loops are forking-path machines. A loop that tries 200 specifications and
reports the 3 that worked looks rigorous and is not. The ledger makes the
garden of forking paths visible, so the number of tests behind each claim can
be reported and corrected for.
"""
from __future__ import annotations

import json
from dataclasses import dataclass, field


@dataclass
class Ledger:
    entries: list[dict] = field(default_factory=list)

    def log(self, agent: str, action: str, **info) -> None:
        self.entries.append({"step": len(self.entries), "agent": agent,
                             "action": action, **info})

    @property
    def n_tests(self) -> int:
        return sum(1 for e in self.entries if e["action"] == "test")

    def dump(self, path: str) -> None:
        with open(path, "w") as f:
            for e in self.entries:
                f.write(json.dumps(e, default=str) + "\n")
