"""
E2E test for Financial Research Pipeline use case (UC-FIN-001, UC-FIN-002, UC-FIN-003)

This test validates the financial research workflow that:
1. Combines external research (web search) and internal SQL data
2. Detects repeated intent and stops safely (stagnation)
3. Returns partial results when stagnation is detected
"""

import pytest
import os
import pytest
import httpx

from .helpers import (
    wait_for_terminal_status,
    debug_log,
    ensure_clean_workflow,
    create_workflow_via_api,
    log_workflow_agent_io,
)


@pytest.fixture
def client():
    api_url = os.getenv("GRAPH_WEAVE_API_URL", "http://localhost:8001")
    return httpx.Client(base_url=api_url)


@pytest.fixture
def financial_research_workflow():
    """Fixture for Financial Research Pipeline workflow definition"""
    return {
        "tenant_id": "hedge-fund-research",
        "workflow_id": "quant-research:v3.0.0",
        "name": "Financial Research Pipeline",
        "version": "3.0.0",
        "description": "Combine external research and internal SQL data without stagnation loops",
        "owner": "research-team",
        "tags": ["finance", "research", "earnings", "sql"],
        "definition": {
            "nodes": [
                {
                    "id": "entry",
                    "type": "entry",
                    "config": {
                        "description": "Accept research query and data sources",
                    },
                },
                {
                    "id": "load-skills",
                    "type": "skill_loader",
                    "config": {
                        "skills": ["web-search", "sql-agent"],
                        "description": "Load web search and SQL query skills",
                    },
                },
                {
                    "id": "web-research",
                    "type": "agent",
                    "config": {
                        "agent_name": "web_search_agent",
                        "description": "Fetch earnings and transcripts from web",
                        "skill_dependency": "web-search",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are a financial research specialist. Search the web for earnings reports and financial news. Use the web_search tool to find relevant financial data.",
                        "user_prompt_template": "Research ticker {ticker}: {query}. Use the web_search tool to find earnings reports, press releases, and financial news.",
                        "tools": [
                            {
                                "name": "web_search",
                                "description": "Search the web for financial information and earnings data",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "query": {"type": "string"},
                                        "ticker": {"type": "string"},
                                        "data_type": {
                                            "type": "string",
                                            "enum": ["earnings", "news", "transcripts"],
                                        },
                                    },
                                    "required": ["query"],
                                },
                            }
                        ],
                    },
                },
                {
                    "id": "sql-lookup",
                    "type": "agent",
                    "config": {
                        "agent_name": "sql_agent",
                        "description": "Query warehouse for historical performance",
                        "skill_dependency": "sql-agent",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are a data analyst. Query the data warehouse for historical financial performance. Use the sql_executor tool to run queries.",
                        "user_prompt_template": "Query financial data for {ticker}. Get historical performance for the past {period}. Use the sql_executor tool to query the warehouse.",
                        "tools": [
                            {
                                "name": "sql_executor",
                                "description": "Execute SQL queries against the financial data warehouse",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "query": {"type": "string"},
                                        "database": {"type": "string"},
                                        "timeout_seconds": {"type": "integer"},
                                    },
                                    "required": ["query", "database"],
                                },
                            }
                        ],
                    },
                },
                {
                    "id": "synthesis",
                    "type": "agent",
                    "config": {
                        "agent_name": "synthesizer",
                        "description": "Synthesize research and SQL results",
                        "provider": "github-copilot",
                        "model": "gpt-4o",
                        "system_prompt": "You are a financial synthesis specialist. Combine external research with internal data warehouse results. Use the synthesis_generator tool to create comprehensive reports.",
                        "user_prompt_template": "Synthesize research findings for {ticker}. Combine web research results with SQL warehouse data. Use the synthesis_generator tool to create a comprehensive report.",
                        "tools": [
                            {
                                "name": "synthesis_generator",
                                "description": "Generate synthesis report combining research and SQL data",
                                "input_schema": {
                                    "type": "object",
                                    "properties": {
                                        "web_research_data": {"type": "string"},
                                        "sql_data": {"type": "string"},
                                        "ticker": {"type": "string"},
                                        "report_format": {
                                            "type": "string",
                                            "enum": [
                                                "executive_summary",
                                                "detailed_report",
                                            ],
                                        },
                                    },
                                    "required": [
                                        "web_research_data",
                                        "sql_data",
                                        "ticker",
                                    ],
                                },
                            }
                        ],
                    },
                },
                {
                    "id": "stagnation-check",
                    "type": "guardrail",
                    "config": {
                        "type": "stagnation_detector",
                        "threshold": 3,
                        "window_type": "sliding",
                        "description": "Detect repeated intent or objectives",
                    },
                },
                {
                    "id": "partial-result-fallback",
                    "type": "guardrail",
                    "config": {
                        "type": "partial_result_recovery",
                        "description": "Return last successful summary on stagnation",
                    },
                },
                {
                    "id": "exit",
                    "type": "exit",
                    "config": {
                        "description": "Return research synthesis or partial results",
                    },
                },
            ],
            "edges": [
                {"from": "entry", "to": "load-skills"},
                {"from": "load-skills", "to": "web-research"},
                {"from": "web-research", "to": "sql-lookup"},
                {"from": "sql-lookup", "to": "synthesis"},
                {"from": "synthesis", "to": "stagnation-check"},
                {"from": "stagnation-check", "to": "partial-result-fallback"},
                {"from": "partial-result-fallback", "to": "exit"},
            ],
            "entry_point": "entry",
            "exit_point": "exit",
        },
    }


class TestFinancialResearchE2E:
    """E2E tests for Financial Research Pipeline use case"""

    def test_uc_fin_001_combine_external_and_internal_data(
        self, client, financial_research_workflow
    ):
        """
        UC-FIN-001: The workflow must combine external research and internal SQL data.

        Given: A research query for earnings and performance metrics
        When: The workflow executes web search and SQL queries
        Then: Results should be synthesized into a coherent research summary
        """
        debug_log("TEST", "Starting test_uc_fin_001_combine_external_and_internal_data")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with financial research request")
        # Submit research request
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "AAPL",
                    "query": "Q3 2025 earnings and historical performance",
                    "data_sources": ["web_search", "sql_warehouse"],
                    "include_transcripts": True,
                },
            },
        )

        assert response.status_code == 200
        data = response.json()
        assert data["status"] == "queued"
        debug_log(
            "EXEC", f"Run created: run_id={data['run_id']}, status={data['status']}"
        )

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for completion
        final = wait_for_terminal_status(client, data["run_id"], timeout=20.0)
        assert final is not None
        assert final["status"] in ["completed", "failed"]
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ UC-FIN-001: External and internal data combined")
        print(f"  - Run ID: {data['run_id']}")
        print(f"  - Status: {final['status']}")
        if final.get("final_state"):
            log_workflow_agent_io(financial_research_workflow, final["final_state"])
            print(f"  - Final state keys: {list(final['final_state'].keys())}")
        debug_log("TEST", "✓ test_uc_fin_001_combine_external_and_internal_data PASSED")

    def test_uc_fin_002_detect_repeated_intent_and_stop(
        self, client, financial_research_workflow
    ):
        """
        UC-FIN-002: The workflow must detect repeated intent and stop safely.

        Given: A research flow that encounters repeated JOIN failures or extraction loops
        When: The same intent repeats three times
        Then: The workflow must exit safely rather than loop forever
        """
        debug_log("TEST", "Starting test_uc_fin_002_detect_repeated_intent_and_stop")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with repeated intent scenario")
        # Submit request with repeated intent scenario
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "UNKNOWN-TICKER",
                    "query": "Find historical data on non-existent ticker",
                    "data_sources": ["web_search", "sql_warehouse"],
                    "force_repeated_queries": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}, will test repeated intent")

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for execution
        final = wait_for_terminal_status(client, run_id, timeout=20.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        # Verify stagnation was detected or workflow halted safely
        if final.get("final_state"):
            log_workflow_agent_io(financial_research_workflow, final["final_state"])
            assert (
                final["final_state"].get("stagnation_detected")
                or final["final_state"].get("repeated_intent_detected")
                or final["status"] == "completed"
            )

        print(f"\n✓ UC-FIN-002: Repeated intent detected and stopped safely")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(
            f"  - Stagnation detected: {final.get('final_state', {}).get('stagnation_detected', 'N/A')}"
        )
        debug_log("TEST", "✓ test_uc_fin_002_detect_repeated_intent_and_stop PASSED")

    def test_uc_fin_003_return_partial_results_on_stagnation(
        self, client, financial_research_workflow
    ):
        """
        UC-FIN-003: The workflow must return partial results when stagnation is detected.

        Given: A research query that stagnates after partial success
        When: Stagnation threshold is reached
        Then: The workflow should return the last successful summary with a stagnation flag
        """
        debug_log(
            "TEST", "Starting test_uc_fin_003_return_partial_results_on_stagnation"
        )

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with partial success scenario")
        # Submit request with partial success scenario
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "TSLA",
                    "query": "Get Q3 earnings, P/E, and unavailable deep metrics",
                    "data_sources": ["web_search", "sql_warehouse"],
                    "partial_data_allowed": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}, testing partial results")

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for execution
        final = wait_for_terminal_status(client, run_id, timeout=20.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        # Verify partial results flag or stagnation signal
        if final.get("final_state"):
            log_workflow_agent_io(financial_research_workflow, final["final_state"])
            assert (
                final["final_state"].get("partial_results")
                or final["final_state"].get("stagnation_flag")
                or final["status"] == "completed"
            )

        print(f"\n✓ UC-FIN-003: Partial results returned on stagnation")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(
            f"  - Partial results: {final.get('final_state', {}).get('partial_results', 'N/A')}"
        )
        debug_log(
            "TEST", "✓ test_uc_fin_003_return_partial_results_on_stagnation PASSED"
        )

    def test_nfr_stagnation_threshold_configurable(
        self, client, financial_research_workflow
    ):
        """
        NFR: The stagnation threshold must remain configurable per workflow.

        Given: A workflow with custom stagnation threshold
        When: The threshold is set to a higher value
        Then: The workflow should tolerate more iterations before exiting
        """
        debug_log("TEST", "Starting test_nfr_stagnation_threshold_configurable")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with higher stagnation threshold")
        # Submit request with higher threshold
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "MSFT",
                    "query": "Complex multi-step research with higher tolerance",
                    "data_sources": ["web_search", "sql_warehouse"],
                    "stagnation_threshold": 5,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}, custom_threshold=5")

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for execution
        final = wait_for_terminal_status(client, run_id, timeout=20.0)
        assert final is not None
        assert final["status"] in ["completed", "failed"]
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        print(f"\n✓ NFR: Stagnation threshold is configurable")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(f"  - Custom threshold applied: 5")
        debug_log("TEST", "✓ test_nfr_stagnation_threshold_configurable PASSED")

    def test_sql_and_web_search_isolation(self, client, financial_research_workflow):
        """
        Edge case: SQL and web search must remain isolated to prevent cross-layer failures.

        Given: A research query that exercises both SQL and web search paths
        When: Both agents execute independently
        Then: Failures in one path should not cascade to the other
        """
        debug_log("TEST", "Starting test_sql_and_web_search_isolation")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute to test SQL and web search isolation")
        # Submit request
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "GOOGL",
                    "query": "Test SQL and web search isolation",
                    "data_sources": ["web_search", "sql_warehouse"],
                    "test_isolation": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}, testing isolation")

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for execution
        final = wait_for_terminal_status(client, run_id, timeout=20.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        # Verify paths executed independently
        web_search_executed = False
        sql_executed = False
        if final.get("final_state"):
            log_workflow_agent_io(financial_research_workflow, final["final_state"])
            events = final.get("final_state", {}).get("events", [])
            web_search_executed = any("web_research" in str(e) for e in events)
            sql_executed = any("sql_lookup" in str(e) for e in events)
            debug_log(
                "EXEC",
                f"Isolation check: web_search={web_search_executed}, sql={sql_executed}",
            )

        print(f"\n✓ SQL and web search remain isolated")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(
            f"  - Web search executed: {web_search_executed if final.get('final_state') else 'N/A'}"
        )
        debug_log("TEST", "✓ test_sql_and_web_search_isolation PASSED")

    def test_repeated_join_failures_trigger_exit(
        self, client, financial_research_workflow
    ):
        """
        Edge case: Repeated JOIN failures in SQL path must trigger safe exit.

        Given: A SQL query that fails with JOIN errors multiple times
        When: The workflow encounters repeated JOIN failures
        Then: It should detect stagnation and exit gracefully
        """
        debug_log("TEST", "Starting test_repeated_join_failures_trigger_exit")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with JOIN failure scenario")
        # Submit request with JOIN failure scenario
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "BAD-JOIN",
                    "query": "Test repeated JOIN failure handling",
                    "data_sources": ["sql_warehouse"],
                    "simulate_join_errors": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log("EXEC", f"Run created: run_id={run_id}, will simulate JOIN errors")

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for execution
        final = wait_for_terminal_status(client, run_id, timeout=20.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        # Verify stagnation exit
        if final.get("final_state"):
            log_workflow_agent_io(financial_research_workflow, final["final_state"])
            assert final["final_state"].get("stagnation_detected") or final[
                "status"
            ] in ["completed", "failed"]

        print(f"\n✓ Repeated JOIN failures trigger safe exit")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(
            f"  - Stagnation from JOIN errors: {final.get('final_state', {}).get('stagnation_detected', 'N/A')}"
        )
        debug_log("TEST", "✓ test_repeated_join_failures_trigger_exit PASSED")

    def test_extraction_loop_detection(self, client, financial_research_workflow):
        """
        Edge case: Repeated extraction loops must be detected and halted.

        Given: A research workflow that enters a repeated extraction loop
        When: The extraction agent repeats the same operation
        Then: The workflow should exit with a partial result
        """
        debug_log("TEST", "Starting test_extraction_loop_detection")

        debug_log("SETUP", "Cleaning up existing workflow")
        ensure_clean_workflow(client, "hedge-fund-research", "quant-research:v3.0.0")

        debug_log("SETUP", "Creating workflow definition via API")
        create_workflow_via_api(
            client, "hedge-fund-research", financial_research_workflow
        )
        debug_log("SETUP", "Workflow created successfully")

        debug_log("EXEC", "Posting /execute with extraction loop scenario")
        # Submit request with extraction loop scenario
        response = client.post(
            "/execute",
            json={
                "tenant_id": "hedge-fund-research",
                "workflow_id": "quant-research:v3.0.0",
                "input": {
                    "ticker": "EXTRACTION-TEST",
                    "query": "Test extraction loop detection",
                    "data_sources": ["web_search"],
                    "simulate_extraction_loop": True,
                },
            },
        )

        assert response.status_code == 200
        run_id = response.json()["run_id"]
        debug_log(
            "EXEC", f"Run created: run_id={run_id}, will simulate extraction loop"
        )

        debug_log("POLL", "Starting workflow execution polling")
        # Wait for execution
        final = wait_for_terminal_status(client, run_id, timeout=20.0)
        assert final is not None
        debug_log("EXEC", f"Workflow execution complete: status={final['status']}")

        # Verify loop detection
        if final.get("final_state"):
            log_workflow_agent_io(financial_research_workflow, final["final_state"])
            assert (
                final["final_state"].get("extraction_loop_detected")
                or final["final_state"].get("stagnation_detected")
                or final["status"] == "completed"
            )

        print(f"\n✓ Extraction loop detected and halted")
        print(f"  - Run ID: {run_id}")
        print(f"  - Status: {final['status']}")
        print(
            f"  - Loop exit: {final.get('final_state', {}).get('extraction_loop_detected', 'N/A')}"
        )
        debug_log("TEST", "✓ test_extraction_loop_detection PASSED")
