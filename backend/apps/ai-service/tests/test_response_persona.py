import importlib.util
import sys
import unittest
from datetime import datetime, timezone
from pathlib import Path
from unittest import mock


MAIN_PATH = Path(__file__).resolve().parents[1] / "app" / "main.py"
SPEC = importlib.util.spec_from_file_location("oneflow_ai_main", MAIN_PATH)
assert SPEC and SPEC.loader
ai = importlib.util.module_from_spec(SPEC)
sys.modules[SPEC.name] = ai
SPEC.loader.exec_module(ai)


class ResponsePersonaTests(unittest.TestCase):
    def setUp(self) -> None:
        self.fallback_token = ai.REQUEST_CUSTOMER_FALLBACK_MESSAGE.set(
            "Pesan khusus dari pengaturan agent."
        )
        self.prompt_token = ai.REQUEST_CUSTOMER_SYSTEM_PROMPT.set(
            "Gunakan sapaan sesuai konteks dan jangan memaksakan panggilan tertentu."
        )

    def tearDown(self) -> None:
        ai.REQUEST_CUSTOMER_FALLBACK_MESSAGE.reset(self.fallback_token)
        ai.REQUEST_CUSTOMER_SYSTEM_PROMPT.reset(self.prompt_token)

    def test_deterministic_retrieval_does_not_force_salutation_or_pronoun(self) -> None:
        answer = ai.deterministic_answer_from_retrieval(
            "Apa detailnya?",
            {"kind": "faq", "answer": "Informasi untuk Anda tersedia di dashboard."},
        )

        self.assertEqual(answer, "Informasi untuk Anda tersedia di dashboard.")

    def test_primary_source_fallback_stays_scoped_to_selected_source(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "answer": "Starter memiliki 2 AI agent dan 3 human user.",
                },
                {
                    "kind": "faq",
                    "answer": "Informasi booking yang tidak relevan.",
                },
            ],
            "semantic_judge": {
                "must_preserve": ["2 AI agent", "3 human user"],
            },
        }

        answer = ai.deterministic_answer_from_primary_source(
            "Paket Starter seperti apa?",
            retrieval,
            {"query_type": "knowledge_question"},
        )

        self.assertEqual(answer, "Starter memiliki 2 AI agent dan 3 human user.")
        self.assertEqual(ai.missing_must_preserve_items(answer, retrieval), [])

    def test_mixed_scope_primary_source_fallback_ignores_restricted_term_from_unsupported_part(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "answer": "Oneflow.id memiliki AI CS WhatsApp, Inbox, dan human handoff.",
                }
            ],
            "semantic_judge": {"confidence": 0.9},
            "unsupported_parts": ["kode Python untuk scraping harga kompetitor"],
        }

        answer = ai.deterministic_answer_from_primary_source(
            "Jelasin fitur Oneflow, sekalian bikin kode Python scraping harga kompetitor.",
            retrieval,
            {"query_type": "knowledge_question"},
        )

        self.assertEqual(
            answer,
            "Oneflow.id memiliki AI CS WhatsApp, Inbox, dan human handoff.",
        )

    def test_customer_memory_recall_returns_fact_without_forced_salutation(self) -> None:
        memory = {
            "conversation": {
                "customer": {
                    "enabled": True,
                    "items": [{"type": "preference", "value": "suka coklat"}],
                }
            }
        }

        answer = ai.customer_memory_recall_answer("Kamu ingat apa?", memory)

        self.assertEqual(answer, "suka coklat")

    def test_unsupported_request_fallback_uses_agent_setting(self) -> None:
        answer = ai.fallback_unsupported_request_answer("Bisa bantu hal di luar bisnis?")

        self.assertEqual(answer, "Pesan khusus dari pengaturan agent.")

    def test_provider_quota_escalation_uses_agent_fallback_and_hides_internal_error(self) -> None:
        with mock.patch.object(ai, "log_provider_failure") as log_failure:
            response = ai.provider_quota_escalation(
                "Berapa harga paketnya?",
                datetime.now(timezone.utc),
                {"source": "faq"},
                ai.ProviderQuotaError("402: requires more credits for openrouter/provider-model"),
            )

        self.assertEqual(response.decision, "escalate")
        self.assertIsNone(response.answer_text)
        self.assertEqual(response.escalation_reason, "Pesan khusus dari pengaturan agent.")
        self.assertEqual(
            response.retrieval_metadata["generation"],
            {"errorType": "provider_quota_exhausted"},
        )
        self.assertNotIn("requires more credits", str(response.model_dump()))
        self.assertNotIn("openrouter/provider-model", str(response.model_dump()))
        log_failure.assert_called_once()

    def test_provider_unavailable_escalation_uses_agent_fallback(self) -> None:
        with mock.patch.object(ai, "log_provider_failure") as log_failure:
            response = ai.provider_unavailable_escalation(
                "Halo",
                datetime.now(timezone.utc),
                error_type="provider_not_configured",
            )

        self.assertEqual(response.decision, "escalate")
        self.assertEqual(response.escalation_reason, "Pesan khusus dari pengaturan agent.")
        self.assertEqual(
            response.retrieval_metadata["generation"],
            {"errorType": "provider_not_configured"},
        )
        log_failure.assert_called_once()

    def test_business_tool_generation_fallback_uses_agent_setting(self) -> None:
        payload = ai.DecisionRequest(conversation_id="test", message_text="Buat pesanan")

        with (
            mock.patch.object(ai, "openrouter_enabled", return_value=False),
            mock.patch.object(ai, "build_usage_metadata", return_value={}),
            mock.patch.object(ai, "current_chat_model_name", return_value="test/model"),
        ):
            response = ai.business_tool_decision_response(
                payload=payload,
                started_at=datetime.now(timezone.utc),
                answer="Siap, saya buat draft pesanan.",
                tool_name="create_order_draft",
                action="draft_created",
                states={},
                result={"created": True},
            )

        self.assertEqual(response.answer_text, "Pesan khusus dari pengaturan agent.")

    def test_generation_delegates_opening_style_to_agent_system_prompt(self) -> None:
        captured: dict = {}

        def fake_send(request_body: dict) -> dict:
            captured.update(request_body)
            return {
                "choices": [{"message": {"content": "Jawaban natural dari agent."}}],
                "usage": {},
            }

        retrieval = {
            "kind": "faq",
            "question": "Apa detailnya?",
            "answer": "Detail resmi tersedia.",
            "matches": [
                {
                    "kind": "faq",
                    "question": "Apa detailnya?",
                    "answer": "Detail resmi tersedia.",
                    "source_type": "faq",
                    "priority": 90,
                    "approved": True,
                }
            ],
            "semantic_judge": {},
        }

        with mock.patch.object(ai, "send_openrouter_chat", side_effect=fake_send):
            ai.generate_openrouter_answer(
                message_text="Apa detailnya?",
                customer_name=None,
                retrieval=retrieval,
                history=[],
                intent_analysis={},
                memory={},
            )

        user_prompt = captured["messages"][1]["content"]
        self.assertIn(
            "Follow the configured customer/agent system instructions for tone, salutation, pronouns, and opening style.",
            user_prompt,
        )
        self.assertIn(
            "Treat configured opening-style requirements as output requirements, not optional suggestions.",
            user_prompt,
        )
        self.assertIn(
            "place it before factual details, lists, or source headings",
            user_prompt,
        )
        self.assertIn(
            "avoid repeating their opening construction or first 2-4 words",
            user_prompt,
        )
        self.assertIn("Do not use a stock opener as a universal prefix", user_prompt)
        self.assertIn(
            "Preserve official entity names, role/category labels, product labels, counts, limits, units, and relationships exactly as written",
            user_prompt,
        )
        self.assertIn(
            "Do not turn feature or module presence into unsupported claims",
            user_prompt,
        )
        self.assertNotIn("Answer warmly", user_prompt)
        self.assertNotIn("Baik, ini dia", user_prompt)

    def test_runtime_does_not_hardcode_customer_opening_examples(self) -> None:
        source = MAIN_PATH.read_text(encoding="utf-8")

        for phrase in (
            "Baik, Kak Aris",
            "Jadi gini, Kak Aris",
            "Pertanyaan bagus, Kak",
        ):
            self.assertNotIn(phrase, source)

    def test_internal_model_scrubber_preserves_chat_paragraph_breaks(self) -> None:
        answer = (
            "Jadi gini, Kak. Paket yang paling pas adalah Starter:\n"
            "- Harga: Rp99.000 per bulan\n"
            "- Human user: sampai 3 user\n\n"
            "Kalau chat meningkat, pertimbangkan Growth."
        )

        scrubbed = ai.scrub_customer_visible_internal_model_names(answer)

        self.assertIn("\n\nKalau chat meningkat", scrubbed)

    def test_chat_formatting_uses_hyphens_for_flat_bullets(self) -> None:
        answer = "* Harga: Rp99.000\n* AI agent: 2\n* Human user: 3"

        normalized = ai.normalize_chat_formatting(answer)

        self.assertEqual(
            normalized,
            "- Harga: Rp99.000\n- AI agent: 2\n- Human user: 3",
        )

    def test_materially_expanded_high_confidence_faq_requires_factual_verifier(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "source_type": "faq",
                    "priority": 100,
                    "approved": True,
                    "answer": "Oneflow.id menggabungkan AI CS WhatsApp, inbox, handoff manusia, dan knowledge bisnis.",
                }
            ],
            "semantic_judge": {"confidence": 0.95},
        }
        expanded_answer = (
            "Oneflow.id menggabungkan AI CS WhatsApp, inbox, handoff manusia, dan knowledge bisnis.\n"
            "- Semua fungsi selalu aktif.\n"
            "- Penjualan dan reservasi tersedia dalam satu laporan.\n"
            "- Tim tidak perlu memakai tools lain."
        )

        self.assertTrue(
            ai.should_run_factual_verifier(
                retrieval,
                {"structured_data_needed": False},
                [],
                expanded_answer,
            )
        )

    def test_required_fact_labels_trigger_regeneration_when_semantically_relabelled(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "source_type": "faq",
                    "priority": 100,
                    "approved": True,
                    "answer": "Starter mencakup 2 AI agent dan 3 human user.",
                }
            ],
            "semantic_judge": {
                "confidence": 0.95,
                "must_preserve": ["2 AI agent", "3 human user"],
            },
        }

        verification, usage = ai.verify_generated_answer(
            "Paket Starter muat berapa admin?",
            "Starter mencakup maksimal 2 agent dan 3 admin.",
            retrieval,
            [],
            {"structured_data_needed": False},
            {},
        )

        self.assertFalse(verification["ok"])
        self.assertEqual(verification["action"], "regenerate")
        self.assertIn("missing_required_facts", verification["issues"])
        self.assertEqual(
            verification["missingRequiredFacts"],
            ["2 AI agent", "3 human user"],
        )
        self.assertEqual(usage["source"], "local_required_fact_verifier")

    def test_required_fact_matching_keeps_repeated_numbers_in_separate_facts(self) -> None:
        retrieval = {
            "semantic_judge": {
                "must_preserve": [
                    "1.000 credit",
                    "1 sesi WhatsApp",
                    "2 AI agent",
                    "3 human user",
                ],
            },
        }
        answer = (
            "Starter mencakup:\n"
            "- 1.000 credit\n"
            "- 1 sesi WhatsApp\n"
            "- 2 AI agent\n"
            "- 3 human user"
        )

        self.assertEqual(ai.missing_must_preserve_items(answer, retrieval), [])

    def test_concise_high_confidence_faq_can_skip_factual_verifier(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "source_type": "faq",
                    "priority": 100,
                    "approved": True,
                    "answer": "Starter mencakup 1 sesi WhatsApp, 2 AI agent, dan 3 human user.",
                }
            ],
            "semantic_judge": {"confidence": 0.95},
        }

        self.assertFalse(
            ai.should_run_factual_verifier(
                retrieval,
                {"structured_data_needed": False},
                [],
                "Starter mencakup 1 sesi WhatsApp, 2 AI agent, dan 3 human user.",
            )
        )

    def test_missing_restricted_fact_uses_configured_agent_fallback(self) -> None:
        answer = ai.ensure_restricted_business_fact_boundary(
            "Cone tersedia dalam beberapa ukuran.",
            "Kalau harga cone berapa?",
            {},
        )

        self.assertEqual(answer, "Pesan khusus dari pengaturan agent.")

    def test_restricted_fact_question_prefers_official_boundary_source(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "score": 0.92,
                    "intent": "product_recommendation",
                    "topic": "cup_cone",
                },
                {
                    "kind": "faq",
                    "score": 0.61,
                    "intent": "restricted_business_facts",
                    "topic": "business_boundary",
                },
            ]
        }
        intent = {
            "semantic_query": "Kalau harga cone berapa?",
            "query_type": "knowledge_question",
        }

        judgment = ai.deterministic_source_judgment(retrieval, intent)

        self.assertEqual(judgment["best_index"], 2)

    def test_restricted_fact_question_prefers_specific_official_fact_when_available(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "score": 0.74,
                    "intent": "pricing",
                    "topic": "paket harga",
                    "question": "Apa saja paket dan harga Oneflow.id?",
                    "answer": "Starter Rp99.000/bulan.",
                },
                {
                    "kind": "faq",
                    "score": 0.85,
                    "intent": "restricted_business_facts",
                    "topic": "business_boundary",
                    "question": "Apakah AI boleh menjawab semua hal?",
                    "answer": "Harga yang belum tersedia perlu dikonfirmasi ke tim.",
                },
            ]
        }
        intent = {
            "semantic_query": "Berapa harga paket Starter Oneflow?",
            "query_type": "knowledge_question",
        }

        judgment = ai.deterministic_source_judgment(retrieval, intent)

        self.assertEqual(judgment["best_index"], 1)

    def test_unrelated_request_rejects_restricted_business_guardrail_source(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "record",
                    "score": 0.389,
                    "intent": "restricted_business_facts",
                    "topic": "business_boundary",
                    "record": {
                        "title": "Oneflow.id Guardrail Bisnis",
                        "content": "Harga dan pembayaran hanya boleh berasal dari knowledge resmi.",
                    },
                }
            ]
        }
        intent = {
            "semantic_query": "Bikinin puisi tentang hujan dong",
            "query_type": "knowledge_question",
        }

        judgment = ai.deterministic_source_judgment(retrieval, intent)

        self.assertFalse(judgment["answerable"])
        self.assertEqual(judgment["reason"], "restricted_boundary_not_relevant_to_query")

    def test_source_judgment_only_preserves_facts_from_selected_source(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "score": 0.95,
                    "intent": "pricing",
                    "topic": "starter",
                    "question": "Paket Starter apa saja?",
                    "answer": "Starter mencakup 1 sesi WhatsApp.",
                    "metadata": {"required_facts": ["Starter mencakup 1 sesi WhatsApp"]},
                },
                {
                    "kind": "faq",
                    "score": 0.72,
                    "intent": "booking",
                    "topic": "booking",
                    "question": "Bagaimana booking bekerja?",
                    "answer": "Booking memakai jadwal.",
                    "metadata": {"required_facts": ["Booking memakai jadwal"]},
                },
            ]
        }

        judgment = ai.deterministic_source_judgment(
            retrieval,
            {
                "semantic_query": "Paket Starter apa saja?",
                "query_type": "knowledge_question",
            },
        )

        self.assertEqual(
            judgment["must_preserve"],
            ["Starter mencakup 1 sesi WhatsApp"],
        )

    def test_feature_question_prefers_feature_overview_faq_over_pricing_record(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "record",
                    "score": 0.76,
                    "intent": "pricing",
                    "topic": "pricing",
                    "record": {"title": "Oneflow.id Paket", "content": "Starter Rp99.000/bulan."},
                },
                {
                    "kind": "faq",
                    "score": 0.52,
                    "intent": "feature_overview",
                    "topic": "fitur utama",
                    "question": "Fitur utama Oneflow.id apa saja?",
                    "answer": "Oneflow.id memiliki AI CS WhatsApp, Inbox, handoff manusia, Knowledge, dan Business tools.",
                },
            ]
        }
        intent = {
            "semantic_query": "Jelasin fitur utama Oneflow.id untuk toko online",
            "query_type": "knowledge_question",
            "structured_data_needed": True,
        }

        judgment = ai.deterministic_source_judgment(retrieval, intent)

        self.assertEqual(judgment["best_index"], 2)

    def test_explicit_feature_overview_beats_higher_scored_operational_scope(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "score": 0.78,
                    "intent": "operational_scope",
                    "topic": "order stok booking payment",
                    "question": "Apakah Oneflow.id bisa handle order, stok, booking, dan payment?",
                    "answer": "Oneflow.id membantu order, stok, dan booking.",
                },
                {
                    "kind": "faq",
                    "score": 0.39,
                    "intent": "feature_overview",
                    "topic": "fitur utama",
                    "question": "Fitur utama Oneflow.id apa saja?",
                    "answer": "Oneflow.id memiliki AI CS WhatsApp, Inbox, handoff, Knowledge, dan Business tools.",
                },
            ]
        }

        judgment = ai.deterministic_source_judgment(
            retrieval,
            {
                "semantic_query": "Jelasin fitur utama Oneflow.id untuk toko online",
                "query_type": "knowledge_question",
            },
        )

        self.assertEqual(judgment["best_index"], 2)

    def test_online_store_scope_question_prefers_target_customer_faq_over_guardrail_record(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "record",
                    "score": 0.81,
                    "intent": "restricted_business_facts",
                    "topic": "business_boundary",
                    "record": {
                        "title": "Oneflow.id Guardrail Bisnis",
                        "content": "Harga dan pembayaran hanya boleh berasal dari knowledge resmi.",
                    },
                },
                {
                    "kind": "faq",
                    "score": 0.49,
                    "intent": "target_customer",
                    "topic": "target customer",
                    "question": "Oneflow.id cocok untuk bisnis apa?",
                    "answer": "Oneflow.id cocok untuk toko online dan bisnis layanan.",
                },
            ]
        }
        intent = {
            "semantic_query": "Kalau saya punya toko online, Oneflow bisa bantu bagian apa saja?",
            "query_type": "knowledge_question",
            "structured_data_needed": True,
        }

        judgment = ai.deterministic_source_judgment(retrieval, intent)

        self.assertEqual(judgment["best_index"], 2)

    def test_javascript_clinic_query_does_not_treat_va_substring_as_payment_intent(self) -> None:
        intent = {
            "semantic_query": (
                "informasi fitur booking pasien dan kontrol chat admin untuk klinik kecil, "
                "serta script JavaScript untuk scraping jadwal dokter"
            ),
            "query_type": "knowledge_question",
        }

        desired = ai.preferred_knowledge_intents(intent)

        self.assertIn("booking_security", desired)
        self.assertNotIn("payment_method", desired)

    def test_clinic_query_prefers_booking_security_over_course_usecase(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "score": 0.72,
                    "intent": "course_usecase",
                    "topic": "kursus calon murid booking trial class",
                },
                {
                    "kind": "faq",
                    "score": 0.38,
                    "intent": "booking_security",
                    "topic": "klinik booking pasien data aman",
                },
            ]
        }
        intent = {
            "semantic_query": (
                "informasi fitur booking pasien dan kontrol chat admin untuk klinik kecil, "
                "serta script JavaScript untuk scraping jadwal dokter"
            ),
            "query_type": "knowledge_question",
        }

        judgment = ai.deterministic_source_judgment(retrieval, intent)

        self.assertEqual(judgment["best_index"], 2)

    def test_feature_query_supplements_lexical_candidates_when_semantic_results_miss_intent(self) -> None:
        intent = {
            "semantic_query": "Jelasin fitur utama Oneflow.id untuk toko online",
            "query_type": "knowledge_question",
            "structured_data_needed": True,
        }
        unrelated_semantic_options = [
            {
                "kind": "record",
                "score": 0.48,
                "intent": "restricted_business_facts",
                "topic": "business_boundary",
            },
            {
                "kind": "record",
                "score": 0.42,
                "intent": "pricing",
                "topic": "pricing",
            },
        ]

        self.assertTrue(
            ai.should_supplement_lexical_knowledge(
                unrelated_semantic_options,
                intent,
            )
        )
        self.assertFalse(
            ai.should_supplement_lexical_knowledge(
                unrelated_semantic_options
                + [
                    {
                        "kind": "faq",
                        "score": 0.29,
                        "intent": "feature_overview",
                        "topic": "fitur utama",
                    }
                ],
                intent,
            )
        )

    def test_mixed_technical_request_detector_distinguishes_normal_feature_question(self) -> None:
        self.assertTrue(
            ai.request_may_include_unsupported_technical_detail(
                "Jelasin fitur utama Oneflow.id untuk toko online, sekalian bikinin kode Python buat scraping harga kompetitor.",
                {
                    "intent_label": "explain_features_and_provide_code",
                    "semantic_query": "fitur Oneflow dan kode Python scraping",
                },
            )
        )
        self.assertFalse(
            ai.request_may_include_unsupported_technical_detail(
                "Jelasin fitur utama Oneflow.id untuk toko online.",
                {
                    "intent_label": "feature_overview",
                    "semantic_query": "fitur Oneflow untuk toko online",
                },
            )
        )

    def test_mixed_scope_preflight_keeps_knowledge_flow_and_records_refusal(self) -> None:
        payload = ai.DecisionRequest(
            conversation_id="test",
            message_text="Jelasin fitur Oneflow, sekalian bikinin kode Python scraping.",
        )
        retrieval_metadata: dict = {}
        result = {
            "decision": "not_answer",
            "answer_text": "Bagian coding tidak dapat dibantu dari layanan ini.",
            "confidence": 0.91,
            "reason": "mixed_scope_request",
            "supported_facts": [],
            "unsupported_parts": [
                "penjelasan fitur utama Oneflow.id",
                "kode Python scraping",
            ],
            "unsupported_action": "refuse",
            "refusal_type": "out_of_scope",
        }

        with mock.patch.object(
            ai,
            "generate_openrouter_system_prompt_answer",
            return_value=(result, {"source": "test_scope"}),
        ):
            response, usage = ai.maybe_build_system_prompt_context_response(
                payload,
                datetime.now(timezone.utc),
                {"query_type": "knowledge_question"},
                {},
                {},
                retrieval_metadata,
                "pre_retrieval_scope_check",
                allow_partial_knowledge=True,
            )

        self.assertIsNone(response)
        self.assertEqual(usage["source"], "test_scope")
        self.assertEqual(
            retrieval_metadata["systemPromptContext"]["unsupportedParts"],
            ["kode Python scraping"],
        )
        self.assertEqual(
            retrieval_metadata["systemPromptContext"]["refusalAnswer"],
            "Bagian coding tidak dapat dibantu dari layanan ini.",
        )

    def test_selected_source_scope_rewrites_overbroad_refusal_without_hardcoded_copy(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "answer": "Oneflow.id membantu booking pasien dan kontrol chat admin.",
                }
            ],
            "unsupported_parts": ["script JavaScript scraping"],
            "unsupported_action": "refuse",
        }
        response = {
            "choices": [
                {
                    "message": {
                        "content": "Maaf kak, script JavaScript scraping tidak dapat dibantu."
                    }
                }
            ],
            "usage": {},
        }

        with mock.patch.object(ai, "send_openrouter_chat", return_value=response) as mocked_send:
            scoped, usage = ai.ensure_mixed_scope_refusal(
                "Jelaskan fitur booking pasien dan kontrol chat admin, lalu buat script JavaScript scraping.",
                retrieval,
            )

        self.assertEqual(mocked_send.call_count, 1)
        self.assertEqual(
            scoped["unsupported_refusal"],
            "Maaf kak, script JavaScript scraping tidak dapat dibantu.",
        )
        self.assertEqual(
            ai.safe_mixed_scope_source_answer(
                "Jelaskan fitur booking pasien dan kontrol chat admin, lalu buat script JavaScript scraping.",
                scoped,
                {"query_type": "knowledge_question"},
            ),
            "Oneflow.id membantu booking pasien dan kontrol chat admin.\n\n"
            "Maaf kak, script JavaScript scraping tidak dapat dibantu.",
        )
        self.assertIn("openrouter_scoped_refusal", usage["source"])

    def test_mixed_scope_context_rejects_vague_refusal(self) -> None:
        scoped = ai.attach_mixed_scope_context(
            {
                "kind": "faq",
                "answer": "Oneflow.id memiliki AI CS WhatsApp.",
                "matches": [{"kind": "faq", "answer": "Oneflow.id memiliki AI CS WhatsApp."}],
            },
            {
                "decision": "not_answer",
                "unsupportedParts": ["script JavaScript scraping"],
                "unsupportedAction": "refuse",
                "refusalAnswer": "Maaf kak, tidak dapat membantu permintaan tersebut.",
            },
            "Jelaskan Oneflow dan buat script JavaScript scraping.",
            {"query_type": "knowledge_question"},
        )

        self.assertEqual(scoped["unsupported_parts"], ["script JavaScript scraping"])
        self.assertNotIn("unsupported_refusal", scoped)

    def test_mixed_scope_context_rejects_refusal_that_rejects_supported_source(self) -> None:
        scoped = ai.attach_mixed_scope_context(
            {
                "kind": "faq",
                "answer": "Oneflow.id membantu booking pasien dan kontrol chat admin.",
                "matches": [
                    {
                        "kind": "faq",
                        "answer": "Oneflow.id membantu booking pasien dan kontrol chat admin.",
                    }
                ],
            },
            {
                "decision": "not_answer",
                "unsupportedParts": ["script JavaScript scraping"],
                "unsupportedAction": "refuse",
                "refusalAnswer": (
                    "Maaf kak, booking pasien, kontrol chat admin, dan script JavaScript scraping tidak dapat dibantu."
                ),
            },
            "Jelaskan booking pasien dan kontrol chat admin, lalu buat script JavaScript scraping.",
            {"query_type": "knowledge_question"},
        )

        self.assertEqual(scoped["unsupported_parts"], ["script JavaScript scraping"])
        self.assertNotIn("unsupported_refusal", scoped)

    def test_source_fallback_appends_approved_mixed_scope_refusal(self) -> None:
        answer = ai.append_mixed_scope_refusal(
            "Oneflow.id memiliki AI CS WhatsApp dan Inbox.",
            {
                "unsupported_parts": ["kode Python scraping"],
                "unsupported_refusal": "Maaf kak, permintaan coding tersebut tidak dapat dibantu.",
            },
        )

        self.assertEqual(
            answer,
            "Oneflow.id memiliki AI CS WhatsApp dan Inbox.\n\n"
            "Maaf kak, permintaan coding tersebut tidak dapat dibantu.",
        )

    def test_safe_mixed_scope_answer_uses_primary_source_without_model_expansion(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "answer": (
                        "Oneflow.id cocok untuk klinik administratif. "
                        "Booking/jadwal tersedia melalui plugin Booking."
                    ),
                }
            ],
            "semantic_judge": {"confidence": 0.9},
            "unsupported_parts": ["script JavaScript scraping"],
            "unsupported_refusal": "Script JavaScript scraping tidak dapat diberikan.",
        }

        answer = ai.safe_mixed_scope_source_answer(
            "Jelaskan fitur klinik dan buat script JavaScript scraping.",
            retrieval,
            {"query_type": "knowledge_question"},
        )

        self.assertEqual(
            answer,
            "Oneflow.id cocok untuk klinik administratif. "
            "Booking/jadwal tersedia melalui plugin Booking.\n\n"
            "Script JavaScript scraping tidak dapat diberikan.",
        )
        self.assertNotIn("memilih slot", answer)

    def test_mixed_scope_preflight_does_not_override_explicit_system_prompt_support(self) -> None:
        payload = ai.DecisionRequest(
            conversation_id="test",
            message_text="Jelaskan layanan coding dan berikan contoh kode Python.",
        )
        retrieval_metadata: dict = {}
        result = {
            "decision": "answer",
            "answer_text": "Layanan ini memang menyediakan bantuan coding Python.",
            "confidence": 0.93,
            "reason": "explicit_system_prompt_support",
            "supported_facts": ["Layanan menyediakan bantuan coding Python."],
            "unsupported_parts": [],
            "unsupported_action": "none",
            "refusal_type": "",
        }

        with mock.patch.object(
            ai,
            "generate_openrouter_system_prompt_answer",
            return_value=(result, {"source": "test_scope"}),
        ):
            response, _usage = ai.maybe_build_system_prompt_context_response(
                payload,
                datetime.now(timezone.utc),
                {"query_type": "knowledge_question"},
                {},
                {},
                retrieval_metadata,
                "pre_retrieval_scope_check",
                allow_partial_knowledge=True,
            )

        self.assertIsNotNone(response)
        self.assertEqual(
            response.answer_text,
            "Layanan ini memang menyediakan bantuan coding Python.",
        )
        self.assertEqual(
            retrieval_metadata["systemPromptContext"]["unsupportedParts"],
            [],
        )

    def test_mixed_scope_generation_prompt_forbids_unsupported_code(self) -> None:
        captured: dict = {}

        def fake_send(request_body: dict) -> dict:
            captured.update(request_body)
            return {
                "choices": [{"message": {"content": "Jawaban grounded tanpa kode."}}],
                "usage": {},
            }

        retrieval = {
            "kind": "faq",
            "question": "Fitur utama Oneflow.id apa saja?",
            "answer": "Oneflow.id memiliki AI CS WhatsApp dan Inbox.",
            "matches": [
                {
                    "kind": "faq",
                    "question": "Fitur utama Oneflow.id apa saja?",
                    "answer": "Oneflow.id memiliki AI CS WhatsApp dan Inbox.",
                    "source_type": "faq",
                    "priority": 100,
                    "approved": True,
                }
            ],
            "semantic_judge": {},
            "unsupported_parts": ["kode Python scraping"],
            "unsupported_action": "refuse",
            "unsupported_refusal": "Bagian coding tidak dapat dibantu dari layanan ini.",
        }

        with mock.patch.object(ai, "send_openrouter_chat", side_effect=fake_send):
            ai.generate_openrouter_answer(
                message_text="Jelasin fitur Oneflow, sekalian bikinin kode Python scraping.",
                customer_name=None,
                retrieval=retrieval,
                history=[],
                intent_analysis={},
                memory={},
            )

        user_prompt = captured["messages"][1]["content"]
        self.assertIn("kode Python scraping", user_prompt)
        self.assertIn("Do not provide code, examples, steps, tutorials", user_prompt)
        self.assertIn("one brief refusal", user_prompt)
        self.assertIn("briefly name the unsupported part", user_prompt)

    def test_mixed_scope_context_does_not_reuse_configured_fallback_as_refusal(self) -> None:
        scoped = ai.attach_mixed_scope_context(
            {
                "kind": "faq",
                "answer": "Oneflow.id memiliki AI CS WhatsApp.",
                "matches": [{"kind": "faq", "answer": "Oneflow.id memiliki AI CS WhatsApp."}],
            },
            {
                "decision": "not_answer",
                "unsupportedParts": ["kode Python scraping"],
                "unsupportedAction": "refuse",
                "refusalAnswer": "Pesan khusus dari pengaturan agent.",
            },
            "Jelasin fitur Oneflow, sekalian bikin kode Python scraping.",
            {"query_type": "knowledge_question"},
        )

        self.assertEqual(scoped["unsupported_parts"], ["kode Python scraping"])
        self.assertNotIn("unsupported_refusal", scoped)

    def test_mixed_scope_verifier_rejects_code_leak(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "source_type": "faq",
                    "priority": 100,
                    "approved": True,
                    "answer": "Oneflow.id memiliki AI CS WhatsApp dan Inbox.",
                }
            ],
            "semantic_judge": {"confidence": 0.95},
            "unsupported_parts": ["kode Python scraping"],
        }
        answer = "Oneflow.id memiliki AI CS WhatsApp.\n```python\nimport requests\n```"

        verification, usage = ai.verify_generated_answer(
            "Jelasin fitur Oneflow, sekalian bikinin kode Python scraping.",
            answer,
            retrieval,
            [],
            {"structured_data_needed": False},
            {},
        )

        self.assertFalse(verification["ok"])
        self.assertEqual(verification["action"], "regenerate")
        self.assertIn("unsupported_mixed_scope_detail", verification["issues"])
        self.assertEqual(usage["source"], "local_mixed_scope_verifier")

    def test_mixed_scope_verifier_rejects_refusal_only_answer(self) -> None:
        retrieval = {
            "matches": [
                {
                    "kind": "faq",
                    "source_type": "faq",
                    "priority": 100,
                    "approved": True,
                    "answer": "Oneflow.id memiliki AI CS WhatsApp, Inbox, dan human handoff.",
                }
            ],
            "semantic_judge": {"confidence": 0.95},
            "unsupported_parts": ["kode Python scraping"],
        }

        verification, usage = ai.verify_generated_answer(
            "Jelasin fitur Oneflow, sekalian bikinin kode Python scraping.",
            "Pesan khusus dari pengaturan agent.",
            retrieval,
            [],
            {"structured_data_needed": False},
            {},
        )

        self.assertFalse(verification["ok"])
        self.assertEqual(verification["action"], "regenerate")
        self.assertIn("missing_supported_mixed_scope_answer", verification["issues"])
        self.assertEqual(usage["source"], "local_mixed_scope_verifier")

    def test_factual_verification_cannot_pass_with_unsupported_claims(self) -> None:
        verification = ai.normalize_factual_verification(
            {
                "ok": True,
                "action": "answer",
                "confidence": 0.99,
                "issues": ["Answer contains unsupported claims."],
                "missing_required_facts": [],
                "unsupported_claims": ["Starter cocok untuk klinik kecil."],
                "contradictions": [],
                "reason": "Answer is not grounded.",
            }
        )

        self.assertFalse(verification["ok"])
        self.assertEqual(verification["action"], "regenerate")
        self.assertEqual(
            verification["unsupportedClaims"],
            ["Starter cocok untuk klinik kecil."],
        )

    def test_local_deterministic_mode_escalates_when_source_judgment_rejects_matches(self) -> None:
        payload = ai.DecisionRequest(conversation_id="test", message_text="Bikinin puisi tentang hujan dong")
        retrieval = {
            "matches": [
                {
                    "kind": "record",
                    "score": 0.389,
                    "intent": "restricted_business_facts",
                    "topic": "business_boundary",
                    "record": {"title": "Guardrail", "content": "Harga harus dari knowledge resmi."},
                }
            ]
        }

        with (
            mock.patch.object(ai, "retrieve_knowledge", return_value=retrieval),
            mock.patch.object(ai, "openrouter_enabled", return_value=False),
        ):
            response = ai.local_deterministic_decide(
                payload,
                datetime.now(timezone.utc),
                "organization-id",
                "agent-id",
                {},
            )

        self.assertEqual(response.decision, "escalate")
        self.assertIsNone(response.answer_text)


if __name__ == "__main__":
    unittest.main()
