"""Run with:  python -m unittest discover tests"""
import json
import unittest

from theoryloop import Proposition, make_world, run, WORLD_NAMES
from theoryloop.loop import score
from theoryloop.theory import test_proposition
from theoryloop.agents import LLMProposer, LLMFalsifier, parse_json
from theoryloop.ledger import Ledger
from theoryloop.worlds import OUTCOME


class TestWorlds(unittest.TestCase):
    def test_reproducible(self):
        a, b = make_world("scope", 7), make_world("scope", 7)
        self.assertTrue(a.data.equals(b.data))

    def test_planted_effects_are_present(self):
        for name in WORLD_NAMES:
            w = make_world(name, 0, n=20000)
            for slot in w.truth:
                for p in slot:
                    self.assertTrue(test_proposition(p, w.data, OUTCOME)["holds"], (name, p))

    def test_decoy_is_a_decoy(self):
        w = make_world("confound", 0, n=20000)
        raw = test_proposition(Proposition("verified", +1), w.data, OUTCOME)
        ctl = test_proposition(Proposition("verified", +1), w.data, OUTCOME, controls=["community"])
        self.assertTrue(raw["holds"]); self.assertFalse(ctl["holds"])


class TestScoring(unittest.TestCase):
    def test_equivalent_framings_both_count(self):
        w = make_world("scope", 0)
        a = [Proposition("verified", 1, ("bear_launch", 1))]
        b = [Proposition("bear_launch", 1, ("verified", 1))]
        self.assertEqual(score(a, w.truth)["perfect"], 1)
        self.assertEqual(score(b, w.truth)["perfect"], 1)

    def test_missing_scope_is_wrong_scope_not_false(self):
        s = score([Proposition("verified", 1)], make_world("scope", 0).truth)
        self.assertEqual((s["wrong_scope"], s["false_props"], s["perfect"]), (1, 0, 0))

    def test_null_world(self):
        t = make_world("null", 0).truth
        self.assertEqual(score([], t)["perfect"], 1)
        self.assertEqual(score([Proposition("royalty", 1)], t)["any_false"], 1)


class TestLoop(unittest.TestCase):
    def test_loop_catches_confound(self):
        r = run(make_world("confound", 0), "loop", 0)
        self.assertNotIn(Proposition("verified", 1), r["theory"])
        self.assertIn(Proposition("community", 1), r["theory"])

    def test_ledger_counts_every_test(self):
        r = run(make_world("main", 0), "loop_holdout", 0)
        self.assertGreater(r["n_tests"], 10)
        self.assertTrue(any(e.get("set") == "holdout" for e in r["ledger"].entries))


class FakeClaude:
    """Stands in for the API so the LLM plumbing is tested without a key."""
    def __init__(self, replies): self.replies, self.calls = list(replies), []
    def __call__(self, system, user, model, max_tokens=1500):
        self.calls.append(user); return self.replies.pop(0)


class TestLLMBackend(unittest.TestCase):
    def test_parse_json_with_prose(self):
        self.assertEqual(parse_json('Here you go:\n[{"a": 1}]'), [{"a": 1}])

    def test_anonymized_round_trip(self):
        w = make_world("confound", 0)
        fake = FakeClaude([json.dumps([{"predictor": "x1", "sign": 1}]),   # x1 = verified
                           json.dumps([{"index": 0, "kind": "confound", "var": "x3", "sign": 1}])])
        led = Ledger()
        theory = LLMProposer(anonymize=True, caller=fake).propose(w.data, led)
        self.assertEqual(theory, [Proposition("verified", 1)])
        self.assertNotIn("verified", fake.calls[0])            # names really hidden
        obj = LLMFalsifier(anonymize=True, caller=fake).critique(theory, w.data, led)
        self.assertEqual(obj[0]["var"], "community")           # x3 mapped back
        self.assertNotIn("community", fake.calls[1])


if __name__ == "__main__":
    unittest.main()
