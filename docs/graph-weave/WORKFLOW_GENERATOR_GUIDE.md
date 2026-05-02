# Workflow Generator Guide

The **Workflow Generator** is a built-in meta-workflow (`workflow-generator:v1.0.0`) in GraphWeave that automatically converts natural language intents into validated, ready-to-deploy GraphWeave DAGs (Directed Acyclic Graphs).

This guide covers best practices, rules, and a sample use case for utilizing the generator effectively.

## 🧠 How It Works

The workflow generator processes your intent through a rigorous multi-agent pipeline:

1. **Planner Agent**: Decomposes the intent into deterministic data transformation steps.
2. **Node Builder Agent**: Generates the GraphWeave nodes with strict JSON schemas.
3. **Edge Router Agent**: Generates a cycle-free DAG connecting the transformation pipeline.
4. **Assembler & Quality Validator**: Assembles the final workflow and enforces quality standards (e.g., rejecting if any node lacks an output schema).
5. **Validation Gate**: Loops back to the planner if validation fails.

## 📝 How to Craft an Intent

The workflow generator performs best when given structured, predictable instructions. An ideal intent follows a clear `Trigger → Pipeline Steps → Final Output` formula.

### The Intent Formula

> `"On [Trigger Event], [Action 1], [Action 2], ..., and [Final Output Action]."`

**Examples of Good vs. Bad Intents:**

✅ **Good (Deterministic & Structured):**

> _"When a Cloudflare WAF rule triggers, fetch the blocked request details, classify the threat type, and update the blocklist rule automatically."_
> **Why it works:** Clear trigger (WAF rule), discrete transformation steps (fetch, classify), and a tangible output (update blocklist).

❌ **Bad (Vague & Open-ended):**

> _"Look at our web traffic and if you see something weird, analyze it to see if it's a threat and then fix it."_
> **Why it fails:** Uses non-deterministic terms like "look at", "weird", "analyze", and "fix it". The workflow generator does not know what specific data to process or what the exact output format should be.

---

## ⚖️ Rules for Writing Intents

To ensure the generator produces a valid and useful DAG, you **must** adhere to the following rules when crafting your natural language query:

1. **Focus on Deterministic Data Transformation**
   - **DO USE**: `extract`, `classify`, `transform`, `validate`, `aggregate`.
   - **DO NOT USE**: `analyze`, `think`, `reflect`, `explore`. The system is designed to build data pipelines, not open-ended reasoning loops.

2. **Specify Clear Inputs and Outputs**
   - Explicitly define what triggers the workflow (the input data) and what the final state should look like (the output).
   - Example: _"When an EKS pod enters a CrashLoopBackOff state, extract the error pattern, classify the root cause, and post a structured incident summary to Slack with recommended remediation steps."_

3. **Enforce Structured Thinking**
   - The generator strictly expects structured outputs. Your intent should map well to a pipeline of discrete steps. Keep the scope limited to a 3-6 step pipeline.

## 🚀 Best Practices

- **Domain Hinting**: Always provide a relevant `domain` hint (e.g., `devops`, `finance`, `research`) alongside your intent. This helps the agents contextualize the node generation.
- **Avoid Ambiguity**: Be explicit about the tools or systems involved (e.g., "AWS Secrets Manager", "Slack", "Grafana").
- **Review and Register**: The generator outputs a JSON DAG. Always review the `generated_workflow` output before registering it via the POST `/workflows` endpoint.

## 📖 Programmatic Sample Use Case: AWS Credential Rotation

Here is an end-to-end example of generating and registering a workflow using Python and `httpx`, mirroring our actual E2E test suite.

```python
import httpx
import time

TENANT_ID = "platform-eng"
WORKFLOW_ID = "workflow-generator:v1.0.0"
API_URL = "http://localhost:8001"

client = httpx.Client(base_url=API_URL, timeout=120.0)

# Step 1: Submit the Intent
intent = (
    "Rotate an expiring AWS IAM access key: detect keys older than 90 days, "
    "generate new key pair, update the secret in AWS Secrets Manager, "
    "notify the owning team via Slack, and archive the old key."
)

print("Submitting intent...")
response = client.post(
    "/execute",
    json={
        "tenant_id": TENANT_ID,
        "workflow_id": WORKFLOW_ID,
        "input": {
            "intent": intent,
            "domain": "devops",
            "correction_attempts": 0,
        },
    },
)
response.raise_for_status()
run_id = response.json()["run_id"]
print(f"Run started with ID: {run_id}")

# Step 2: Poll for Completion
print("Waiting for workflow generator to complete...")
while True:
    # Use the real /status endpoint
    status_response = client.get(f"/execute/{run_id}/status")
    status_data = status_response.json()
    if status_data["status"] in ["completed", "failed"]:
        break
    time.sleep(2)

# Step 3: Extract and Register the Generated DAG
if status_data["status"] == "completed":
    state = status_data.get("workflow_state", {})
    generated_workflow = state.get("generated_workflow")

    if generated_workflow:
        print("Successfully generated DAG. Registering new workflow...")

        new_workflow_id = "aws-credential-rotation:v1.0.0"
        payload = {
            "tenant_id": TENANT_ID,
            "workflow_id": new_workflow_id,
            "name": "AWS Credential Rotation",
            "version": "1.0.0",
            "description": "Generated by workflow-generator",
            "owner": "platform-eng",
            "tags": ["generated", "devops"],
            "definition": generated_workflow,
        }

        reg_response = client.post("/workflows", json=payload)
        reg_response.raise_for_status()
        print(f"Registered new workflow '{new_workflow_id}' successfully!")
else:
    print(f"Workflow generation failed. Final status: {status_data['status']}")
```

Once registered, you can immediately execute your new `aws-credential-rotation:v1.0.0` workflow!

## 💻 Sample Use Case via cURL

You can also interact with the workflow generator and register the workflow using standard shell commands.

**1. Submit the Intent**

```bash
curl -X POST http://localhost:8001/execute \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "platform-eng",
    "workflow_id": "workflow-generator:v1.0.0",
    "input": {
      "intent": "When an EKS pod enters a CrashLoopBackOff state, fetch the pod logs, analyze the error pattern, and post a structured incident summary to Slack.",
      "domain": "devops",
      "correction_attempts": 0
    }
  }'
```

**2. Poll for Completion and Extract DAG**
Once the run status is `completed`, extract the generated DAG:

```bash
curl -s http://localhost:8001/execute/<RUN_ID>/status | jq '.workflow_state.generated_workflow' > dag.json
```

**3. Register the Workflow**
It is recommended to use a payload file to handle the complex DAG structure:

```bash
# Create the registration payload
cat <<EOF > payload.json
{
  "tenant_id": "platform-eng",
  "workflow_id": "eks-incident-handler:v1.0.0",
  "name": "EKS Incident Handler",
  "version": "1.0.0",
  "description": "Generated by workflow-generator",
  "owner": "platform-eng",
  "tags": ["generated", "devops"],
  "definition": $(cat dag.json)
}
EOF

# Submit to GraphWeave
curl -X POST http://localhost:8001/workflows \
  -H "Content-Type: application/json" \
  -d @payload.json
```
