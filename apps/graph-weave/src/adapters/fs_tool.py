import os
import shlex
import shutil
import subprocess
import time
from typing import Dict, Any, List, Optional
from pathlib import Path

from src.app_logging import get_logger

logger = get_logger(__name__)


class FileSystemToolError(Exception):
    """Exception raised for errors in the FileSystemTool."""
    pass


class FileSystemTool:
    """
    Executes file system operations using CLI utilities (eza, rg, ls, grep, mv, cp)
    with strict path restrictions and soft-delete capabilities.
    """

    def __init__(self, allowed_paths: Optional[List[str]] = None, trash_path: Optional[str] = None):
        """
        Initialize the FileSystemTool.
        
        Args:
            allowed_paths: List of allowed absolute root paths.
            trash_path: Absolute path to the trash directory.
        """
        self.allowed_paths = [os.path.abspath(p) for p in (allowed_paths or [os.getcwd()])]
        
        # Configure trash path
        if trash_path:
            self.trash_path = os.path.abspath(trash_path)
        else:
            self.trash_path = os.path.abspath(os.path.join(self.allowed_paths[0], ".trash"))
            
        # Ensure trash directory exists if allowed
        if self._is_path_allowed(self.trash_path):
            os.makedirs(self.trash_path, exist_ok=True)

        # Detect available enhanced tools
        self.has_eza = shutil.which("eza") is not None
        self.has_rg = shutil.which("rg") is not None

    def _is_path_allowed(self, path: str) -> bool:
        """Check if an absolute path falls within any allowed paths."""
        abs_path = os.path.abspath(path)
        return any(abs_path.startswith(allowed) for allowed in self.allowed_paths)

    def _validate_path(self, path: str) -> str:
        """Validate and return the absolute path."""
        abs_path = os.path.abspath(path)
        
        # Prevent obvious traversal
        if ".." in path:
             raise FileSystemToolError(f"Path traversal detected: {path}")
             
        if not self._is_path_allowed(abs_path):
            raise FileSystemToolError(f"Access denied. Path outside allowed boundaries: {abs_path}")
            
        return abs_path

    def _run_cli(self, command: List[str], timeout: int = 30) -> Dict[str, Any]:
        """Execute a CLI command securely and return result."""
        cmd_str = " ".join(shlex.quote(str(arg)) for arg in command)
        logger.debug(f"Running FS CLI: {cmd_str}")
        try:
            result = subprocess.run(
                command,  # passing list avoids shell=True security issues, though we quoted anyway
                capture_output=True,
                text=True,
                timeout=timeout
            )
            return {
                "success": result.returncode == 0,
                "stdout": result.stdout,
                "stderr": result.stderr,
                "exit_code": result.returncode
            }
        except subprocess.TimeoutExpired:
            return {"success": False, "error": f"Timeout ({timeout}s)", "exit_code": -1}
        except Exception as e:
            return {"success": False, "error": str(e), "exit_code": -1}

    def list_files(self, path: str, recursive: bool = False) -> Dict[str, Any]:
        """List files using eza or ls."""
        valid_path = self._validate_path(path)
        
        if self.has_eza:
            cmd = ["eza", "-l", "--no-user", "--no-time"]
            if recursive:
                cmd.append("-R")
        else:
            cmd = ["ls", "-l"]
            if recursive:
                cmd.append("-R")
                
        cmd.append(valid_path)
        return self._run_cli(cmd)

    def search_files(self, pattern: str, path: str) -> Dict[str, Any]:
        """Search in files using rg or grep."""
        valid_path = self._validate_path(path)
        
        if self.has_rg:
            cmd = ["rg", "-n", pattern, valid_path]
        else:
            cmd = ["grep", "-rn", pattern, valid_path]
            
        return self._run_cli(cmd)

    def read_file(self, path: str) -> Dict[str, Any]:
        """Read a file using cat."""
        valid_path = self._validate_path(path)
        if not os.path.exists(valid_path) or not os.path.isfile(valid_path):
             return {"success": False, "error": "File not found or is a directory", "exit_code": 1}
        return self._run_cli(["cat", valid_path])

    def write_file(self, path: str, content: str) -> Dict[str, Any]:
        """Write to a file using temporary file and mv for atomicity."""
        valid_path = self._validate_path(path)
        tmp_path = valid_path + f".tmp.{int(time.time())}"
        
        try:
            # We use Python to securely write the content to the temp file
            with open(tmp_path, "w", encoding="utf-8") as f:
                f.write(content)
            
            # Use mv for atomic replacement
            result = self._run_cli(["mv", tmp_path, valid_path])
            if not result["success"]:
                 # Cleanup tmp if mv failed
                 if os.path.exists(tmp_path):
                     os.remove(tmp_path)
            return result
        except Exception as e:
            if os.path.exists(tmp_path):
                os.remove(tmp_path)
            return {"success": False, "error": str(e), "exit_code": -1}

    def replace_in_file(self, path: str, pattern: str, replacement: str) -> Dict[str, Any]:
        """Replace text in a file using sed."""
        valid_path = self._validate_path(path)
        # Using sed -i. On macOS, sed -i '' is required, on Linux it's sed -i.
        import platform
        if platform.system() == "Darwin":
            cmd = ["sed", "-i", "", f"s/{pattern}/{replacement}/g", valid_path]
        else:
            cmd = ["sed", "-i", f"s/{pattern}/{replacement}/g", valid_path]
        return self._run_cli(cmd)

    def move_file(self, source: str, destination: str) -> Dict[str, Any]:
        """Move a file using mv."""
        valid_src = self._validate_path(source)
        valid_dst = self._validate_path(destination)
        return self._run_cli(["mv", valid_src, valid_dst])

    def copy_file(self, source: str, destination: str) -> Dict[str, Any]:
        """Copy a file using cp."""
        valid_src = self._validate_path(source)
        valid_dst = self._validate_path(destination)
        return self._run_cli(["cp", "-r", valid_src, valid_dst])

    def delete_file(self, path: str) -> Dict[str, Any]:
        """Soft delete a file by moving it to the trash directory."""
        valid_path = self._validate_path(path)
        
        if not os.path.exists(valid_path):
             return {"success": False, "error": "File not found", "exit_code": 1}
             
        # Ensure trash exists
        if not os.path.exists(self.trash_path) and self._is_path_allowed(self.trash_path):
            os.makedirs(self.trash_path, exist_ok=True)
            
        base_name = os.path.basename(valid_path)
        timestamp = int(time.time())
        trash_dest = os.path.join(self.trash_path, f"{base_name}.{timestamp}.trash")
        
        return self._run_cli(["mv", valid_path, trash_dest])

    def make_directory(self, path: str) -> Dict[str, Any]:
        """Create a directory using mkdir -p."""
        valid_path = self._validate_path(path)
        return self._run_cli(["mkdir", "-p", valid_path])

    def get_file_info(self, path: str) -> Dict[str, Any]:
        """Get file stats using stat."""
        valid_path = self._validate_path(path)
        import platform
        if platform.system() == "Darwin":
             cmd = ["stat", "-x", valid_path]
        else:
             cmd = ["stat", valid_path]
        return self._run_cli(cmd)
