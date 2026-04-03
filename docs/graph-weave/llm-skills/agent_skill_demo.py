"""
Simplified Tool-Based Skills Workflow

Skills are listed in the system prompt (NOT as LLM function tools).
Agent executes them via internal function routing.
"""

import os
import sys
import re
import json
import subprocess
import requests
import importlib.util
from pathlib import Path
from typing import Dict, List, Optional, Any
from dataclasses import dataclass


@dataclass
class Skill:
    """Skill metadata from SKILL.md frontmatter."""
    name: str
    description: str
    path: Path


class SkillsManager:
    """Discovers and executes skills."""

    def __init__(self, skills_dir: str):
        self.skills_dir = Path(skills_dir)
        self.skills: Dict[str, Skill] = {}
        self.cache: Dict[str, str] = {}

    def discover(self) -> List[Skill]:
        """Find all SKILL.md files and parse metadata."""
        if not self.skills_dir.exists():
            return []

        for folder in self.skills_dir.iterdir():
            skill_file = folder / "SKILL.md"
            if folder.is_dir() and skill_file.exists():
                skill = self._parse(skill_file)
                if skill:
                    self.skills[skill.name] = skill

        return list(self.skills.values())

    def _parse(self, path: Path) -> Optional[Skill]:
        """Extract name/description from YAML frontmatter."""
        try:
            text = path.read_text()
            match = re.match(r'^---\s*\n(.*?)\n---', text, re.DOTALL)
            if not match:
                return None

            front = match.group(1)
            name = re.search(r'name:\s*(.+)', front)
            desc = re.search(r'description:\s*(.+)', front)

            if name and desc:
                return Skill(name.group(1).strip(), desc.group(1).strip(), path)
        except Exception as e:
            print(f"Parse error: {e}")
        return None

    def to_xml(self) -> str:
        """Generate skills XML for system prompt."""
        if not self.skills:
            return ""

        lines = ["<available_skills>"]
        for s in self.skills.values():
            lines += [f"  <skill>", f"    <name>{s.name}</name>",
                     f"    <description>{s.description}</description>", "  </skill>"]
        lines.append("</available_skills>")
        return "\n".join(lines)

    def activate(self, name: str) -> Optional[str]:
        """Load full SKILL.md content (cached)."""
        if name not in self.skills:
            return None
        if name not in self.cache:
            self.cache[name] = self.skills[name].path.read_text()
        return self.cache[name]

    def execute(self, name: str, action: str, **params) -> Dict:
        """Execute skill action by dynamically importing and calling Python functions."""
        if name not in self.skills:
            return {"error": f"Skill '{name}' not found"}

        folder = self.skills[name].path.parent

        # First try to import and call the function directly
        result = self._import_and_call(folder, action, **params)
        if result is not None:
            return result

        # Fallback: try running as subprocess
        result = self._run_script(folder, action, **params)
        if result is not None:
            return result

        return {"error": f"No executable found for action '{action}' in skill '{name}'"}

    def _import_and_call(self, folder: Path, action: str, **params) -> Optional[Dict]:
        """Dynamically import Python modules and call the action function."""
        # Find all Python files in the folder
        py_files = list(folder.glob("*.py"))

        for py_file in py_files:
            if py_file.name.startswith("__"):
                continue

            try:
                # Dynamically import the module
                spec = importlib.util.spec_from_file_location(py_file.stem, py_file)
                if spec and spec.loader:
                    module = importlib.util.module_from_spec(spec)
                    sys.modules[py_file.stem] = module
                    spec.loader.exec_module(module)

                    # Check if the action function exists in this module
                    if hasattr(module, action):
                        func = getattr(module, action)
                        # Call the function with params
                        result = func(**params)

                        # Ensure result is a dict
                        if isinstance(result, dict):
                            return result
                        else:
                            return {"success": True, "result": result}
            except Exception as e:
                # Continue to next file if this one fails
                continue

        return None

    def _run_script(self, folder: Path, action: str, **params) -> Optional[Dict]:
        """Try to execute scripts as subprocess (fallback method)."""
        for script in [f"{action}.py", "run.py", f"{action}.sh"]:
            path = folder / script
            if not path.exists():
                continue

            try:
                # Use python to run .py scripts instead of direct execution
                if script.endswith('.py'):
                    args = [sys.executable, str(path), action]
                else:
                    args = [str(path), action]

                for k, v in params.items():
                    args.extend([f"--{k}", str(v)])

                result = subprocess.run(args, capture_output=True, text=True,
                                       timeout=30, cwd=str(folder))
                try:
                    return json.loads(result.stdout)
                except json.JSONDecodeError:
                    return {"success": result.returncode == 0,
                           "stdout": result.stdout, "stderr": result.stderr}
            except Exception as e:
                return {"error": str(e)}
        return None


class Agent:
    """LLM agent with skill function calling."""

    SYSTEM_PROMPT = """You are an AI assistant with specialized skills.

WORKFLOW:
1. First, call activate_skill to load skill instructions
2. Then, call execute_skill to perform the actual task
3. Finally, provide the results to the user in plain text

CRITICAL: Only make ONE function call per response. Wait for the function result before making another call.

To call a function, respond with ONLY this JSON (nothing else):
{{"function_call": {{"name": "function_name", "arguments": {{...}}}}}}

Available functions:
- activate_skill: {{"skill_name": "string"}}
- execute_skill: {{"skill_name": "string", "action": "string", "params": {{}}}}

{skills}

When you receive function results, analyze them and either:
- Call another function if needed (ONE function only)
- Provide final answer with the actual results/data"""

    def __init__(self, skills: SkillsManager, api_key: str):
        self.skills = skills
        self.api_url = "https://inference.do-ai.run/v1/chat/completions"
        self.api_key = api_key
        self.max_turns = 10
        self.messages = []

    def chat(self, user_input: str) -> str:
        """Process user message with iterative function calling."""
        print(f"\n{'='*60}\nUSER: {user_input}\n{'='*60}\n")

        # Initialize conversation with system prompt
        system_msg = self.SYSTEM_PROMPT.format(skills=self.skills.to_xml())
        self.messages = [
            {"role": "system", "content": system_msg},
            {"role": "user", "content": user_input}
        ]

        for turn in range(self.max_turns):
            print(f"[Turn {turn + 1}]")

            # Call LLM
            response = self._call_llm()
            print(f"💭 LLM: {response}\n")

            # Check for function call
            func = self._parse_function(response)
            if func:
                # Execute the function
                result = self._execute(func)

                # Add assistant response and function result to conversation
                self.messages.append({"role": "assistant", "content": response})
                self.messages.append({"role": "user", "content": f"Function result: {json.dumps(result)}"})
            else:
                # No function call, this is the final response
                print(f"AGENT: {response}\n")
                return response

        return "Max iterations reached. Please try again."

    def _call_llm(self) -> str:
        """HTTP POST to DigitalOcean Serverless Inference API."""
        try:
            headers = {
                "Content-Type": "application/json",
                "Authorization": f"Bearer {self.api_key}"
            }
            data = {
                "model": "llama3.3-70b-instruct",
                "messages": self.messages,
                "max_tokens": 1000,
                "temperature": 0.7
            }

            resp = requests.post(self.api_url, headers=headers, json=data, timeout=30)
            resp.raise_for_status()
            return resp.json()['choices'][0]['message']['content'].strip()
        except Exception as e:
            return f"LLM Error: {e}"

    def _parse_function(self, text: str) -> Optional[Dict]:
        """Extract {"function_call": {...}} from response. Only parse if at start."""
        try:
            # Only look for function calls at the beginning of the response (first 100 chars)
            # This avoids parsing example function calls in explanatory text
            search_text = text[:200].strip()

            if not search_text.startswith('{'):
                return None

            # Find the first complete JSON object
            start = 0
            brace_count = 0
            for i in range(len(text)):
                if text[i] == '{':
                    brace_count += 1
                elif text[i] == '}':
                    brace_count -= 1
                    if brace_count == 0:
                        # Found complete JSON object
                        json_str = text[start:i+1]
                        parsed = json.loads(json_str)
                        return parsed.get('function_call')
        except:
            pass
        return None

    def _execute(self, func: Dict) -> Dict:
        """Route function call to appropriate skill method."""
        name = func.get('name', '')
        args = func.get('arguments', {})

        print(f"🔧 {name}({json.dumps(args)})")

        if name == 'activate_skill':
            content = self.skills.activate(args.get('skill_name', ''))
            result = {"success": bool(content), "instructions": content} if content else {"error": "Not found"}
        elif name == 'execute_skill':
            result = self.skills.execute(args.get('skill_name', ''),
                                        args.get('action', ''),
                                        **args.get('params', {}))
        else:
            result = {"error": f"Unknown function: {name}"}

        print(f"📥 {json.dumps(result, indent=2)}\n")
        return result


def main():
    """Interactive demo."""
    print("="*60)
    print("SIMPLE TOOL-BASED SKILLS EXAMPLE")
    print("="*60)

    # Setup
    api_key = "Add_Your_Inferrence_API_Key_Here"

    skills_dir = Path(__file__).parent / "skills"

    # Initialize
    skills = SkillsManager(str(skills_dir))
    discovered = skills.discover()
    print(f"\n✅ Found {len(discovered)} skills:")
    for s in discovered:
        print(f"   - {s.name}: {s.description}")

    agent = Agent(skills, api_key)

    # Chat loop
    print("\n" + "="*60)
    print("CHAT (type 'quit' to exit)")
    print("="*60)

    while True:
        try:
            user = input("\nYou: ").strip()
            if user.lower() in ['quit', 'exit', 'q']:
                break
            if user:
                agent.chat(user)
        except KeyboardInterrupt:
            break

    print("\nGoodbye!")


if __name__ == "__main__":
    main()
