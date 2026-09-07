import importlib.util
import sys
import unittest
from pathlib import Path


SCRIPT_PATH = Path(__file__).resolve().parents[3] / "scripts" / "setup_local_wa_agent_dummy.py"
SPEC = importlib.util.spec_from_file_location("oneflow_dummy_vicecream", SCRIPT_PATH)
assert SPEC and SPEC.loader
dummy = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = dummy
SPEC.loader.exec_module(dummy)


class DummyViceCreamQualityTests(unittest.TestCase):
    def test_system_prompt_owns_natural_opening_style_without_fixed_phrase(self) -> None:
        prompt = dummy.AGENT_PROMPT

        self.assertIn("Pembuka harus langsung merespons maksud customer", prompt)
        self.assertIn("Jangan memakai stock phrase sebagai awalan universal", prompt)
        self.assertIn("Jangan mengulang konstruksi pembuka atau 2-4 kata awal yang sama", prompt)
        self.assertIn("Jangan memulai dengan judul knowledge", prompt)
        self.assertNotIn("Baik, ini dia", prompt)
        self.assertNotIn("Pertanyaan bagus", prompt)
        self.assertNotIn("Jadi gini", prompt)

    def test_vanilla_recommendation_has_specific_grounded_answer(self) -> None:
        vanilla_answers = [
            answer
            for question, answer, _topic, intent in dummy.FAQS
            if "vanilla" in question.lower() and intent == "product_recommendation"
        ]

        self.assertEqual(len(vanilla_answers), 1)
        answer = vanilla_answers[0]
        self.assertIn("vanilla", answer.lower())
        self.assertIn("ketersediaan", answer.lower())
        self.assertFalse(answer.startswith("Menu utama ViceCream"))

    def test_restricted_business_fact_answer_is_customer_facing(self) -> None:
        restricted_answers = [
            answer
            for _question, answer, _topic, intent in dummy.FAQS
            if intent == "restricted_business_facts"
        ]

        self.assertEqual(len(restricted_answers), 1)
        answer = restricted_answers[0]
        self.assertNotIn("Jika customer", answer)
        self.assertNotIn("Jangan memakai", answer)
        self.assertNotIn("knowledge resmi lokal", answer)


if __name__ == "__main__":
    unittest.main()
