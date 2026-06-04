import pytest
from src.adapters.langgraph.nodes.agent.schema_utils import coerce_to_output_schema

def test_coerce_array_wrapping():
    schema = {
        "type": "object",
        "properties": {
            "tags": {
                "type": "array",
                "items": {"type": "string"}
            }
        },
        "required": ["tags"]
    }
    
    # Input is a single string instead of a list of strings
    input_data = {"tags": "finance"}
    result = coerce_to_output_schema(input_data, schema)
    
    assert result == {"tags": ["finance"]}


def test_coerce_string_to_boolean():
    schema = {
        "type": "object",
        "properties": {
            "is_valid": {"type": "boolean"},
            "has_warnings": {"type": "boolean"}
        },
        "required": ["is_valid", "has_warnings"]
    }
    
    # Input has string-encoded truth values
    input_data = {
        "is_valid": "passed",
        "has_warnings": "no"
    }
    result = coerce_to_output_schema(input_data, schema)
    
    assert result == {
        "is_valid": True,
        "has_warnings": False
    }


def test_coerce_string_to_number():
    schema = {
        "type": "object",
        "properties": {
            "confidence": {"type": "number"},
            "retries": {"type": "integer"}
        },
        "required": ["confidence", "retries"]
    }
    
    # Input has string-encoded numeric values
    input_data = {
        "confidence": "0.95",
        "retries": "3.0"
    }
    result = coerce_to_output_schema(input_data, schema)
    
    assert result == {
        "confidence": 0.95,
        "retries": 3
    }


def test_auto_fill_missing_required_fields():
    schema = {
        "type": "object",
        "properties": {
            "recommendations": {
                "type": "object",
                "properties": {
                    "placement_review": {
                        "type": "array",
                        "items": {"type": "string"}
                    }
                },
                "required": ["placement_review"]
            },
            "publishability": {
                "type": "object",
                "properties": {
                    "status": {"type": "string"},
                    "blocking_items": {
                        "type": "array",
                        "items": {"type": "string"}
                    }
                },
                "required": ["status", "blocking_items"]
            }
        },
        "required": ["recommendations", "publishability"]
    }
    
    # Input completely omits the "publishability" object and "placement_review" list
    input_data = {
        "recommendations": {}
    }
    
    result = coerce_to_output_schema(input_data, schema)
    
    # Verify that recommendations.placement_review is auto-filled as []
    # Verify that publishability is auto-filled as an object with status "" and blocking_items []
    assert result == {
        "recommendations": {
            "placement_review": []
        },
        "publishability": {
            "status": "",
            "blocking_items": []
        }
    }


def test_coerce_list_to_multi_field_object_sources():
    schema = {
        "type": "object",
        "properties": {
            "sources_markdown": {
                "type": "array",
                "items": {"type": "string"}
            },
            "drafts_markdown": {
                "type": "array",
                "items": {"type": "string"}
            }
        },
        "required": ["sources_markdown", "drafts_markdown"]
    }
    
    # Raw list should be coerced to the first array field, others auto-filled
    input_data = ["card1", "card2"]
    result = coerce_to_output_schema(input_data, schema)
    
    assert result == {
        "sources_markdown": ["card1", "card2"],
        "drafts_markdown": []
    }


def test_coerce_empty_list_to_multi_field_object():
    schema = {
        "type": "object",
        "properties": {
            "sources_markdown": {
                "type": "array",
                "items": {"type": "string"}
            },
            "drafts_markdown": {
                "type": "array",
                "items": {"type": "string"}
            }
        },
        "required": ["sources_markdown", "drafts_markdown"]
    }
    
    # Empty list should be coerced
    input_data = []
    result = coerce_to_output_schema(input_data, schema)
    
    assert result == {
        "sources_markdown": [],
        "drafts_markdown": []
    }


def test_coerce_dict_with_correct_keys_unchanged():
    schema = {
        "type": "object",
        "properties": {
            "sources_markdown": {
                "type": "array",
                "items": {"type": "string"}
            },
            "drafts_markdown": {
                "type": "array",
                "items": {"type": "string"}
            }
        },
        "required": ["sources_markdown", "drafts_markdown"]
    }
    
    # Dict with keys already present should pass through unchanged
    input_data = {
        "sources_markdown": ["card1"],
        "drafts_markdown": ["draft1"]
    }
    result = coerce_to_output_schema(input_data, schema)
    
    assert result == input_data

