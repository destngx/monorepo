import os
import re
import shlex
import subprocess
from typing import Dict, Any, List, Optional
from pathlib import Path

from src.app_logging import get_logger

logger = get_logger(__name__)


class BashToolError(Exception):
    """Exception raised for errors in the BashTool."""
    pass


class BashTool:
    """
    Executes bash commands with safety limits including path allow-lists,
    command blacklists, timeouts, and smart output truncation.
    """

    ALLOWED_COMMANDS_BASE = {
        "ls", "grep", "cat", "echo", "date", "pwd", "mkdir", "find", 
        "sed", "awk", "python3", "bun", "pip", "npm", "nx", "git"
    }

    # Forbidden patterns like rm -r, sudo, curl, wget, writing to system dirs
    FORBIDDEN_PATTERNS = [
        re.compile(r"rm\s+-r"),
        re.compile(r"sudo\b"),
        re.compile(r"curl\b"),
        re.compile(r"wget\b"),
        re.compile(r"mv\s+.*\/"),
        re.compile(r">\s*\/etc"),
        re.compile(r">>\s*\/etc"),
        re.compile(r">\s*\/var"),
        re.compile(r">\s*\/usr"),
    ]

    def __init__(self, allowed_paths: Optional[List[str]] = None):
        """
        Initialize the BashTool.
        
        Args:
            allowed_paths: List of allowed absolute root paths. Defaults to the CWD if None.
        """
        if allowed_paths is None:
            self.allowed_paths = [os.path.abspath(os.getcwd())]
        else:
            self.allowed_paths = [os.path.abspath(p) for p in allowed_paths]

    def _is_path_allowed(self, path: str) -> bool:
        """Check if an absolute path falls within any allowed paths."""
        abs_path = os.path.abspath(path)
        return any(abs_path.startswith(allowed) for allowed in self.allowed_paths)

    def _validate_command(self, command: str) -> None:
        """
        Validate the command string against forbidden patterns and 
        absolute path access restrictions.
        """
        # Check forbidden patterns
        for pattern in self.FORBIDDEN_PATTERNS:
            if pattern.search(command):
                raise BashToolError(f"Command rejected by forbidden pattern: {pattern.pattern}")

        # Basic check for absolute paths in the command string that are outside allowed paths
        # This is a naive regex to find potential absolute paths
        potential_paths = re.findall(r"(?:^|\s)(/[a-zA-Z0-9_\-\./]+)", command)
        for p in potential_paths:
            if not self._is_path_allowed(p):
                raise BashToolError(f"Command attempts to access forbidden path: {p}")
                
        # While we allow pipes and redirects, we could optionally enforce the base command here.
        # For LLM flexibility, we rely more on the forbidden patterns and path restrictions.

    def _smart_truncate(self, text: str, threshold: int = 4000) -> str:
        """
        Truncate output if it exceeds threshold. 
        Keeps the first 1000 and the last 3000 chars roughly.
        """
        if not text:
            return ""
            
        if len(text) <= threshold:
            return text

        head_chars = 1000
        tail_chars = threshold - head_chars - 50 # 50 for the omission message
        
        head = text[:head_chars]
        tail = text[-tail_chars:] if tail_chars > 0 else ""
        
        omitted_len = len(text) - len(head) - len(tail)
        
        return f"{head}\n\n... [{omitted_len} characters omitted] ...\n\n{tail}"

    def execute_bash(self, command: str, timeout: int = 30, cwd: Optional[str] = None) -> Dict[str, Any]:
        """
        Execute a bash command safely.
        
        Args:
            command: The command string to execute.
            timeout: Execution timeout in seconds.
            cwd: Current working directory for execution.
            
        Returns:
            Dict containing success status, stdout, stderr, and exit code.
        """
        try:
            self._validate_command(command)
        except BashToolError as e:
            logger.warning(f"BashTool validation failed: {e}")
            return {
                "success": False,
                "error": str(e),
                "exit_code": -1
            }

        target_cwd = cwd if cwd is not None else os.getcwd()
        abs_cwd = os.path.abspath(os.path.expanduser(target_cwd))
        
        if not self._is_path_allowed(abs_cwd):
            return {
                "success": False,
                "error": f"Requested cwd '{cwd}' is outside allowed paths.",
                "exit_code": -1
            }

        try:
            # Environment Isolation: Do not inherit the full parent environment.
            # Only provide basic PATH to ensure allowed commands can be found.
            clean_env = {
                "PATH": os.environ.get("PATH", "/usr/bin:/bin:/usr/sbin:/sbin:/usr/local/bin"),
                "TERM": "xterm-256color",
                "LANG": os.environ.get("LANG", "en_US.UTF-8"),
                "PYTHONPATH": os.environ.get("PYTHONPATH", ".")
            }

            # We use shell=True to support pipes and typical bash operations
            # safety relies on _validate_command and container/env limits if any.
            result = subprocess.run(
                command,
                shell=True,
                cwd=abs_cwd,
                env=clean_env,
                capture_output=True,
                text=True,
                timeout=timeout
            )
            
            stdout = self._smart_truncate(result.stdout)
            stderr = self._smart_truncate(result.stderr)
            
            return {
                "success": result.returncode == 0,
                "stdout": stdout,
                "stderr": stderr,
                "exit_code": result.returncode,
                "cwd": abs_cwd
            }
            
        except subprocess.TimeoutExpired:
            logger.warning(f"Command timed out after {timeout} seconds: {command}")
            return {
                "success": False,
                "error": f"Command execution timed out after {timeout} seconds.",
                "exit_code": -1
            }
        except Exception as e:
            logger.error(f"Error executing command '{command}': {e}")
            return {
                "success": False,
                "error": f"Internal execution error: {str(e)}",
                "exit_code": -1
            }
