import pytest
from src.adapters.langgraph.nodes.agent.json_utils import repair_split_arrays, extract_json

def test_basic_split_arrays():
    bad_json = '{"key": ["a"], ["b"], ["c"]}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"key": ["a", "b", "c"]}'
    assert extract_json(bad_json) == {"key": ["a", "b", "c"]}

def test_brackets_inside_strings():
    # String contains brackets - repair should ignore brackets inside strings
    bad_json = '{"key": ["value containing [brackets] and \\"escaped quotes\\""], ["another [bracket]"]}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"key": ["value containing [brackets] and \\"escaped quotes\\"", "another [bracket]"]}'
    
    # Valid json containing brackets in strings should remain completely untouched
    valid_json = '{"valid_array_with_brackets": ["[foo]", "[bar]"]}'
    assert repair_split_arrays(valid_json) == valid_json
    assert extract_json(valid_json) == {"valid_array_with_brackets": ["[foo]", "[bar]"]}

def test_nested_objects_with_split_arrays():
    bad_json = '{"parent": {"child": ["x"], ["y"]}}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"parent": {"child": ["x", "y"]}}'
    assert extract_json(bad_json) == {"parent": {"child": ["x", "y"]}}

def test_multiple_fields_with_split_arrays():
    bad_json = '{"field1": ["a"], ["b"], "field2": ["c"], ["d"]}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"field1": ["a", "b"], "field2": ["c", "d"]}'
    assert extract_json(bad_json) == {"field1": ["a", "b"], "field2": ["c", "d"]}

def test_arrays_of_objects_split():
    bad_json = '{"items": [{"id": 1}], [{"id": 2}]}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"items": [{"id": 1}, {"id": 2}]}'
    assert extract_json(bad_json) == {"items": [{"id": 1}, {"id": 2}]}

def test_real_world_markdown_content():
    content = """Here is the json output:
```json
{
  "sources_markdown": [
    "source 1 [info]"
  ],
  "drafts_markdown": [
    "draft 1"
  ],
  [
    "draft 2"
  ]
}
```

And some python code:
```python
def foo():
    return [1, 2]
```
"""
    parsed = extract_json(content)
    assert parsed == {
        "sources_markdown": ["source 1 [info]"],
        "drafts_markdown": ["draft 1", "draft 2"]
    }

def test_escaped_quotes_inside_strings():
    bad_json = '{"text": "He said, \\"hello\\".", "arr": ["val 1"], ["val 2"]}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"text": "He said, \\"hello\\".", "arr": ["val 1", "val 2"]}'
    assert extract_json(bad_json) == {
        "text": 'He said, "hello".',
        "arr": ["val 1", "val 2"]
    }

def test_mixed_valid_and_split_fields():
    bad_json = '{"valid_array": ["a", "b"], "split_array": ["c"], ["d"]}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"valid_array": ["a", "b"], "split_array": ["c", "d"]}'
    assert extract_json(bad_json) == {
        "valid_array": ["a", "b"],
        "split_array": ["c", "d"]
    }

def test_deeply_nested_structures():
    bad_json = '{"a": {"b": {"c": ["x"], ["y"]}}}'
    repaired = repair_split_arrays(bad_json)
    assert repaired == '{"a": {"b": {"c": ["x", "y"]}}}'
    assert extract_json(bad_json) == {"a": {"b": {"c": ["x", "y"]}}}
