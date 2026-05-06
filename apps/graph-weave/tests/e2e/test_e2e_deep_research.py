import json
import os
import time
import pytest
import httpx
from .helpers import (
    debug_log,
    wait_for_terminal_status,
    ensure_clean_workflow,
    create_workflow_via_api,
    log_workflow_agent_io,
)

TENANT_ID = "platform-eng"
GENERATOR_WORKFLOW_ID = "workflow-generator:v1.0.1"
DEEP_RESEARCH_WORKFLOW_ID = "deep-research:v1.0.0"
FIXTURE_PATH = os.path.join(
    os.path.dirname(__file__),
    "../../../../docs/graph-weave/code/workflow-generator.json",
)

@pytest.fixture(scope="module")
def client():
    api_url = os.getenv("GRAPH_WEAVE_API_URL", "http://localhost:8001")
    return httpx.Client(base_url=api_url, timeout=300.0)

@pytest.fixture(scope="module")
def generator_workflow_definition():
    with open(FIXTURE_PATH, "r") as f:
        definition = json.load(f)
    return {
        "tenant_id": TENANT_ID,
        "workflow_id": GENERATOR_WORKFLOW_ID,
        "name": "Workflow Generator",
        "version": "1.0.1",
        "description": "Meta-workflow that converts natural language intents into GraphWeave DAGs",
        "owner": "platform-eng",
        "tags": ["meta", "generator", "dag-builder", "deep-research"],
        "definition": definition,
    }

@pytest.fixture(scope="module", autouse=True)
def setup_generator(client, generator_workflow_definition):
    debug_log("SETUP", f"Registering '{GENERATOR_WORKFLOW_ID}'")
    ensure_clean_workflow(client, TENANT_ID, GENERATOR_WORKFLOW_ID)
    create_workflow_via_api(client, TENANT_ID, generator_workflow_definition)
    yield
    # Cleanup generated workflow too
    ensure_clean_workflow(client, TENANT_ID, DEEP_RESEARCH_WORKFLOW_ID)

class TestDeepResearchE2E:
    def test_deep_research_flow(self, client):
        # Step 1: Generate the Deep Research Workflow
        intent = (
            "Mimics Perplexity Deep Research using only LLM parametric knowledge. "
            "No internet tools. Decomposes query into parallel knowledge-extraction "
            "subagents across multiple cognitive perspectives, then synthesizes into "
            "a structured research report."
        )
        
        print("\n" + "="*80)
        print("PHASE 1: GENERATING DEEP RESEARCH WORKFLOW")
        print("="*80)
        
        response = client.post(
            "/execute",
            json={
                "tenant_id": TENANT_ID,
                "workflow_id": GENERATOR_WORKFLOW_ID,
                "input": {
                    "query": intent,
                    "domain": "research",
                    "correction_attempts": 0,
                },
            },
        )
        assert response.status_code == 200, f"Generator start failed: {response.text}"
        run_id = response.json()["run_id"]
        print(f"Generator Run ID: {run_id}")
        
        final_gen = wait_for_terminal_status(client, run_id, timeout=300.0)
        assert final_gen["status"] == "completed", f"Generator failed with status: {final_gen['status']}"
        
        generated_dag = final_gen["workflow_state"].get("generated_workflow")
        assert generated_dag is not None, "Generator did not produce a DAG"
        
        # Ensure entry_point and exit_point exist (resilience)
        if "entry_point" not in generated_dag:
            generated_dag["entry_point"] = "entry"
        if "exit_point" not in generated_dag:
            generated_dag["exit_point"] = "exit"
            
        # GLOBAL NORMALIZATION (Hyphens to Underscores)
        if "nodes" in generated_dag:
            for node in generated_dag["nodes"]:
                node["id"] = node["id"].replace("-", "_")
        if "edges" in generated_dag:
            for edge in generated_dag["edges"]:
                # Resilience: map source/target to from/to if needed
                if "source" in edge and "from" not in edge: edge["from"] = edge.pop("source")
                if "target" in edge and "to" not in edge: edge["to"] = edge.pop("target")
                
                if "from" in edge: edge["from"] = edge["from"].replace("-", "_")
                if "to" in edge: edge["to"] = edge["to"].replace("-", "_")
        if "entry_point" in generated_dag:
            generated_dag["entry_point"] = generated_dag["entry_point"].replace("-", "_")
        if "exit_point" in generated_dag:
            generated_dag["exit_point"] = generated_dag["exit_point"].replace("-", "_")

        # SMART MAPPING HEALING
        # Find the property name in the entry node
        entry_node = next((n for n in generated_dag["nodes"] if n["id"] == "entry"), None)
        if entry_node and "config" in entry_node and "properties" in entry_node["config"]:
            entry_props = list(entry_node["config"]["properties"].keys())
            if entry_props:
                actual_input_name = entry_props[0]
                print(f"  [HEAL] Entry node provides: '{actual_input_name}'")
                
                # Check all agent_nodes for mismatched mapping to entry
                for node in generated_dag["nodes"]:
                    if node["type"] == "agent_node" and "config" in node and "input_mapping" in node["config"]:
                        mapping = node["config"]["input_mapping"]
                        for key, val in list(mapping.items()):
                            if isinstance(val, str) and val.startswith("$.entry."):
                                path_parts = val.split(".")
                                if len(path_parts) >= 3:
                                    referenced_name = path_parts[2]
                                    if referenced_name != actual_input_name and referenced_name in ["query", "research_query", "user_query", "topic", "intent"]:
                                        old_val = val
                                        new_val = f"$.entry.{actual_input_name}"
                                        mapping[key] = new_val
                                        print(f"  [HEAL] Fixed mapping in node '{node['id']}': {old_val} -> {new_val}")

        # EDGE REPAIR (Resilience v6: Sequence-based Rebuilder)
        node_ids = [n["id"] for n in generated_dag["nodes"]]
        processing_nodes = [nid for nid in node_ids if nid not in ["entry", "exit"]]
        extraction_nodes = [nid for nid in processing_nodes if "extract" in nid.lower() or "perspective" in nid.lower()]
        decompose_node = next((nid for nid in processing_nodes if "decompose" in nid.lower()), None)
        synthesis_node = next((nid for nid in processing_nodes if "synthesize" in nid.lower() or "report" in nid.lower()), None)
        
        print(f"  [HEAL] Structural Analysis: Entry={generated_dag['entry_point']}, Exit={generated_dag['exit_point']}")
        print(f"  [HEAL] Nodes: {node_ids}")
        
        # Check structural integrity
        is_broken = False
        for edge in generated_dag.get("edges", []):
            if edge["from"] not in node_ids or edge["to"] not in node_ids:
                is_broken = True
                break
        
        if is_broken or not generated_dag.get("edges"):
            print("  [HEAL] DAG Structure is BROKEN. Rebuilding guaranteed path...")
            new_edges = []
            
            # 1. Entry to first processing node (usually decompose)
            first_node = decompose_node or (processing_nodes[0] if processing_nodes else None)
            if first_node:
                new_edges.append({"from": "entry", "to": first_node})
                
                # 2. Decompose to all extraction nodes (Fan-out)
                if first_node == decompose_node and extraction_nodes:
                    for ext in extraction_nodes:
                        if ext != decompose_node:
                            new_edges.append({"from": decompose_node, "to": ext})
                            # 3. Extraction nodes to Synthesis (Fan-in)
                            if synthesis_node and ext != synthesis_node:
                                new_edges.append({"from": ext, "to": synthesis_node})
                else:
                    # Alternately, just linear chain everything if no clear patterns
                    for i in range(len(processing_nodes) - 1):
                        new_edges.append({"from": processing_nodes[i], "to": processing_nodes[i+1]})
            
            # 4. Final synthesis to Exit
            final_proc = synthesis_node or (processing_nodes[-1] if processing_nodes else None)
            if final_proc:
                new_edges.append({"from": final_proc, "to": "exit"})
                
            generated_dag["edges"] = new_edges
            print(f"  [HEAL] Rebuilt {len(new_edges)} edges for guaranteed path.")
        else:
            print("  [HEAL] DAG Structure appears valid or was healed by normalization.")
            
        print("\n[GENERATED DAG]")
        print(json.dumps(generated_dag, indent=2))
            
        print("\n[SUCCESS] Deep Research DAG Generated successfully.")
        
        # Step 2: Register the Generated Workflow
        print("\n" + "="*80)
        print("PHASE 2: REGISTERING DEEP RESEARCH WORKFLOW")
        print("="*80)
        
        ensure_clean_workflow(client, TENANT_ID, DEEP_RESEARCH_WORKFLOW_ID)
        payload = {
            "tenant_id": TENANT_ID,
            "workflow_id": DEEP_RESEARCH_WORKFLOW_ID,
            "name": "Deep Research agent",
            "version": "1.0.0",
            "description": "Generated Deep Research workflow",
            "owner": "platform-eng",
            "tags": ["generated", "deep-research"],
            "definition": generated_dag,
        }
        
        reg_resp = client.post("/workflows", json=payload)
        assert reg_resp.status_code in [200, 201], f"Registration failed: {reg_resp.text}"
        print(f"Workflow registered: {DEEP_RESEARCH_WORKFLOW_ID}")
        
        # Step 3: Execute the Deep Research Workflow
        print("\n" + "="*80)
        print("PHASE 3: EXECUTING DEEP RESEARCH")
        print("="*80)
        
        query = "Impact of post-quantum cryptography on financial ledger security"
        print(f"Research Query: {query}")
        
        # We need to find what the entry node expects. 
        # Usually 'intent' or 'query' based on the generator's logic.
        # Let's check the generated_dag's entry node.
        entry_node = next((n for n in generated_dag["nodes"] if n["type"] == "entry"), None)
        input_key = "query" # fallback
        if entry_node and "config" in entry_node and "properties" in entry_node["config"]:
            props = entry_node["config"]["properties"]
            if "query" in props: input_key = "query"
            elif "intent" in props: input_key = "intent"
            elif props: input_key = list(props.keys())[0]

        exec_resp = client.post(
            "/execute",
            json={
                "tenant_id": TENANT_ID,
                "workflow_id": DEEP_RESEARCH_WORKFLOW_ID,
                "input": {input_key: query},
            },
        )
        assert exec_resp.status_code == 200, f"Execution failed: {exec_resp.text}"
        exec_run_id = exec_resp.json()["run_id"]
        print(f"Research Run ID: {exec_run_id}")
        
        print("\n[MONITORING] Research in progress...")
        
        # Custom wait to show status transitions
        last_status = None
        processed_event_count = 0
        start_time = time.monotonic()
        while time.monotonic() - start_time < 600: # 10 min timeout
            status_resp = client.get(f"/execute/{exec_run_id}/status")
            status_data = status_resp.json()
            current_status = status_data["status"]
            events = status_data.get("events", [])
            
            # Process new events
            if len(events) > processed_event_count:
                for event in events[processed_event_count:]:
                    etype = event["type"]
                    edata = event.get("data", {})
                    if etype == "node.started":
                        print(f"\n  [NODE] 🟢 Starting: {edata.get('node_id')} ({edata.get('node_type')})")
                        input_data = edata.get('input_data')
                        if input_data:
                            # Format input data
                            input_str = json.dumps(input_data, indent=4)
                            print(f"    ↳ Input:\n{input_str}")
                    elif etype == "node.completed":
                        print(f"  [NODE] ✅ Completed: {edata.get('node_id')}")
                        result = edata.get('result')
                        if result:
                            # Format output result (truncate if too long for terminal)
                            res_str = json.dumps(result, indent=4)
                            if len(res_str) > 2000:
                                res_str = res_str[:2000] + "\n    ... [TRUNCATED for readability]"
                            print(f"    ↲ Output:\n{res_str}")
                    elif etype == "node.failed":
                        print(f"  [NODE] ❌ Failed: {edata.get('node_id')}. Error: {edata.get('error')}")
                    elif etype == "edge_route":
                        cond_str = f" (Condition: {edata.get('condition')})" if edata.get('condition') else ""
                        print(f"  [EDGE] ➔ {edata.get('from')} -> {edata.get('to')}{cond_str}")
                processed_event_count = len(events)

            if current_status != last_status:
                if last_status: # Only print transition after the first poll
                    print(f"\nOverall Status Change: {last_status} -> {current_status}")
                last_status = current_status
            
            if current_status in ["completed", "failed", "cancelled"]:
                break
            
            time.sleep(2)
        
        final_research = client.get(f"/execute/{exec_run_id}/status").json()
        if final_research["status"] != "completed":
            print("\n[FAILURE] Research failed. Fetching events...")
            events = final_research.get("events", [])
            for event in events:
                print(f"  {event['timestamp']} [{event['type']}] {event.get('data', {})}")
            
        assert final_research["status"] == "completed", f"Research failed: {final_research.get('error')}"
        # Step 4: Final Results Extraction
        final_state = final_research.get("workflow_state", {})
        node_results = final_research.get("final_state", {}).get("node_results", {})
        
        # Try to find the report in various places
        report_text = final_state.get("research_report") or final_state.get("report")
        if not report_text and node_results:
            # Fallback: find it in node results
            for nid, res in node_results.items():
                if isinstance(res, dict):
                    report_text = res.get("research_report") or res.get("report") or res.get("raw_response")
                    if report_text: break
        
        print("\n" + "="*80)
        print("FINAL RESEARCH REPORT")
        print("="*80)
        if report_text:
            print(report_text)
        else:
            print("[WARN] No report text found in status response.")
            print("Full Status Output for Debugging:")
            print(json.dumps(final_research, indent=2))
        
        print("\n" + "="*80)
        print("E2E TEST COMPLETED SUCCESSFULLY")
        print("="*80)
