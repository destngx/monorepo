import os
import pytest
from unittest.mock import patch, MagicMock
from src.adapters.fs_tool import FileSystemTool, FileSystemToolError

@pytest.fixture
def tmp_workspace(tmp_path):
    workspace = tmp_path / "workspace"
    workspace.mkdir()
    trash = tmp_path / "trash"
    trash.mkdir()
    return str(workspace), str(trash)

@pytest.fixture
def fs_tool(tmp_workspace):
    workspace, trash = tmp_workspace
    return FileSystemTool(allowed_paths=[workspace], trash_path=trash)

def test_init_creates_trash(tmp_workspace):
    workspace, _ = tmp_workspace
    tool = FileSystemTool(allowed_paths=[workspace])
    assert os.path.exists(tool.trash_path)
    assert tool.trash_path == os.path.join(workspace, ".trash")

def test_validate_path_allowed(fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    test_path = os.path.join(workspace, "test.txt")
    assert fs_tool._validate_path(test_path) == test_path

def test_validate_path_denied(fs_tool):
    with pytest.raises(FileSystemToolError, match="Access denied"):
        fs_tool._validate_path("/etc/passwd")

def test_validate_path_traversal(fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    with pytest.raises(FileSystemToolError, match="Path traversal detected"):
        fs_tool._validate_path(os.path.join(workspace, "../outside.txt"))

@patch("subprocess.run")
def test_list_files(mock_run, fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    mock_run.return_value = MagicMock(returncode=0, stdout="file1.txt\n", stderr="")
    
    # Force eza availability for test
    fs_tool.has_eza = True
    
    result = fs_tool.list_files(workspace)
    assert result["success"] is True
    assert result["stdout"] == "file1.txt\n"
    
    # Verify the command called
    mock_run.assert_called_once()
    called_args = mock_run.call_args[0][0]
    assert called_args[0] == "eza"
    assert workspace in called_args

@patch("subprocess.run")
def test_search_files(mock_run, fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    mock_run.return_value = MagicMock(returncode=0, stdout="match\n", stderr="")
    
    fs_tool.has_rg = True
    
    result = fs_tool.search_files("pattern", workspace)
    assert result["success"] is True
    
    mock_run.assert_called_once()
    called_args = mock_run.call_args[0][0]
    assert called_args[0] == "rg"
    assert "pattern" in called_args

@patch("subprocess.run")
def test_delete_file_soft(mock_run, fs_tool, tmp_workspace):
    workspace, trash = tmp_workspace
    file_path = os.path.join(workspace, "to_delete.txt")
    
    # Create the file so os.path.exists passes
    with open(file_path, "w") as f:
        f.write("test")
        
    mock_run.return_value = MagicMock(returncode=0, stdout="", stderr="")
    
    result = fs_tool.delete_file(file_path)
    assert result["success"] is True
    
    mock_run.assert_called_once()
    called_args = mock_run.call_args[0][0]
    assert called_args[0] == "mv"
    assert called_args[1] == file_path
    assert called_args[2].startswith(trash)
    assert "to_delete.txt" in called_args[2]
    assert ".trash" in called_args[2]

@patch("subprocess.run")
def test_cli_injection_prevention(mock_run, fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    # We pass a malicious path. Since _validate_path uses os.path.abspath, 
    # it treats it as a weird filename.
    malicious_path = os.path.join(workspace, "; rm -rf /etc")
    
    # Mock exists so read_file proceeds
    with patch("os.path.exists", return_value=True), \
         patch("os.path.isfile", return_value=True):
        fs_tool.read_file(malicious_path)
        
    called_args = mock_run.call_args[0][0]
    # The command is passed as a list to subprocess.run, 
    # which inherently prevents shell injection unless shell=True is used.
    # We still check the args.
    assert called_args == ["cat", malicious_path]

def test_write_file_atomic(fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    target_path = os.path.join(workspace, "written.txt")
    
    result = fs_tool.write_file(target_path, "new content")
    assert result["success"] is True
    
    with open(target_path, "r") as f:
        assert f.read() == "new content"
        
    # Check that tmp files are cleaned up
    tmp_files = [f for f in os.listdir(workspace) if ".tmp." in f]
    assert len(tmp_files) == 0

def test_replace_in_file(fs_tool, tmp_workspace):
    workspace, _ = tmp_workspace
    target_path = os.path.join(workspace, "sed_test.txt")
    
    with open(target_path, "w") as f:
        f.write("hello world")
        
    # We actually run the sed command
    result = fs_tool.replace_in_file(target_path, "world", "universe")
    assert result["success"] is True
    
    with open(target_path, "r") as f:
        assert f.read() == "hello universe"
