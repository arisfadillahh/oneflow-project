import importlib.util
from pathlib import Path
import unittest


ROOT = Path(__file__).resolve().parents[3]
SCRIPT_PATH = ROOT / "scripts" / "setup_local_oneflow_wa_agent.py"


def load_seed_module():
    spec = importlib.util.spec_from_file_location("setup_local_oneflow_wa_agent", SCRIPT_PATH)
    module = importlib.util.module_from_spec(spec)
    assert spec and spec.loader
    spec.loader.exec_module(module)
    return module


class OneflowWAAgentQualityTests(unittest.TestCase):
    def test_system_prompt_owns_natural_opening_style_without_fixed_template(self):
        prompt = load_seed_module().AGENT_PROMPT

        self.assertIn("Langsung respons maksud customer", prompt)
        self.assertIn("Pilih fungsi pembuka sesuai intent", prompt)
        self.assertIn("memberi arah jawaban, bukan sekadar basa-basi", prompt)
        self.assertIn("Jangan langsung membuka dengan fakta mentah", prompt)
        self.assertIn("setelah pembuka gunakan subjudul singkat", prompt)
        self.assertIn("Jangan memakai stock phrase sebagai awalan universal", prompt)
        self.assertIn("Jangan mengulang konstruksi pembuka atau 2-4 kata awal yang sama", prompt)
        self.assertIn("langsung jawab tanpa pembuka tambahan", prompt)
        self.assertIn("Jangan mengarang nama", prompt)
        for stock_phrase in ("Jadi gini", "Baik, Kak Aris", "Pertanyaan bagus, Kak"):
            self.assertNotIn(stock_phrase, prompt)

    def test_seed_contains_oneflow_knowledge_not_vicecream_knowledge(self):
        module = load_seed_module()
        combined = " ".join(
            [module.AGENT_PROMPT]
            + [question + " " + answer for question, answer, _topic, _intent in module.FAQS]
        )

        self.assertIn("Oneflow.id", combined)
        self.assertNotIn("ViceCream", combined)

    def test_package_recommendation_keeps_ai_agent_and_human_user_as_separate_limits(self):
        module = load_seed_module()
        answer = next(
            answer
            for question, answer, _topic, _intent in module.FAQS
            if question == "Paket apa yang cocok untuk 1 nomor WhatsApp, 2 admin, atau budget kecil?"
        )

        self.assertIn("2 AI agent", answer)
        self.assertIn("3 human user", answer)
        self.assertIn("batas yang terpisah", answer)
        self.assertIn("2 admin manusia", answer)
        self.assertIn("bukan 3 admin", answer)

    def test_comparison_source_stays_within_confirmed_product_scope(self):
        module = load_seed_module()
        answer = next(
            answer
            for question, answer, _topic, _intent in module.FAQS
            if question == "Apa bedanya Oneflow.id dengan chatbot biasa?"
        )

        self.assertIn("Perbedaan utamanya", answer)
        self.assertIn("dalam satu dashboard operasional", answer)
        self.assertNotIn("semua fungsi tersedia", answer.lower())
        self.assertNotIn("satu laporan", answer.lower())

    def test_uncertain_answer_preserves_handoff_as_capability_not_guarantee(self):
        module = load_seed_module()

        self.assertIn(
            "bisa mengarahkan percakapan ke admin",
            module.FAQ_REQUIRED_FACTS[
                "Apa yang terjadi kalau AI kurang yakin atau jawaban kurang data?"
            ],
        )


if __name__ == "__main__":
    unittest.main()
