import os
import pytest
from src.adapters.bash_tool import BashTool, BashToolError

@pytest.fixture
def workspace_root(tmp_path):
    """Fixture providing a temporary allowed workspace root."""
    return str(tmp_path)

@pytest.fixture
def bash_tool(workspace_root):
    return BashTool(allowed_paths=[workspace_root])

def test_initialization(workspace_root):
    tool = BashTool(allowed_paths=[workspace_root])
    assert tool.allowed_paths == [workspace_root]
    
    # Test default
    default_tool = BashTool()
    assert default_tool.allowed_paths == [os.path.abspath(os.getcwd())]

def test_is_path_allowed(bash_tool, workspace_root):
    # Allowed
    assert bash_tool._is_path_allowed(workspace_root)
    assert bash_tool._is_path_allowed(os.path.join(workspace_root, "subdir", "file.txt"))
    
    # Not allowed
    assert not bash_tool._is_path_allowed("/etc")
    assert not bash_tool._is_path_allowed("/tmp")

def test_validate_command_forbidden_patterns(bash_tool):
    forbidden_commands = [
        "rm -r /tmp/test",
        "rm -rf ./dir",
        "sudo apt-get install",
        "curl http://example.com",
        "wget http://example.com",
        "mv /tmp/file /etc/file",
        "echo 'hack' > /etc/passwd",
        "echo 'hack' >> /etc/shadow"
    ]
    for cmd in forbidden_commands:
        with pytest.raises(BashToolError, match="Command rejected by forbidden pattern"):
            bash_tool._validate_command(cmd)

def test_validate_command_absolute_paths(bash_tool, workspace_root):
    # Allowed absolute path
    allowed_cmd = f"cat {workspace_root}/file.txt"
    bash_tool._validate_command(allowed_cmd) # Should not raise
    
    # Forbidden absolute path
    forbidden_cmd = "cat /etc/shadow"
    with pytest.raises(BashToolError, match="Command attempts to access forbidden path"):
        bash_tool._validate_command(forbidden_cmd)

def test_smart_truncate(bash_tool):
    short_text = "Hello world"
    assert bash_tool._smart_truncate(short_text, threshold=100) == short_text
    
    # Exactly threshold
    exact_text = "A" * 100
    assert bash_tool._smart_truncate(exact_text, threshold=100) == exact_text
    
    # Exceed threshold
    long_text = "A" * 5000
    truncated = bash_tool._smart_truncate(long_text, threshold=4000)
    assert "characters omitted" in truncated
    assert len(truncated) < 5000
    assert truncated.startswith("A" * 1000)

def test_execute_bash_success(bash_tool, workspace_root):
    result = bash_tool.execute_bash("echo 'hello'", cwd=workspace_root)
    assert result["success"] is True
    assert result["stdout"].strip() == "hello"
    assert result["exit_code"] == 0

def test_execute_bash_out_of_bounds_cwd(bash_tool):
    result = bash_tool.execute_bash("ls", cwd="/etc")
    assert result["success"] is False
    assert "outside allowed paths" in result["error"]

def test_execute_bash_timeout(bash_tool, workspace_root):
    result = bash_tool.execute_bash("sleep 2", timeout=1, cwd=workspace_root)
    assert result["success"] is False
    assert "timed out" in result["error"]
