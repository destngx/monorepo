import re
import html as html_lib

def extract_text_from_html(html: str) -> str:
    """
    Perform basic text extraction from HTML without external dependencies.
    Removes scripts, styles, and tags.
    """
    # Remove script and style elements
    text = re.sub(r'<(script|style|title|header|footer|nav).*?>.*?</\1>', '', html, flags=re.DOTALL | re.IGNORECASE)
    
    # Remove all other tags
    text = re.sub(r'<.*?>', ' ', text)
    
    # Replace entities
    text = html_lib.unescape(text.replace("&nbsp;", " "))
    
    # Clean up whitespace
    text = re.sub(r'\s+', ' ', text).strip()
    
    return text
