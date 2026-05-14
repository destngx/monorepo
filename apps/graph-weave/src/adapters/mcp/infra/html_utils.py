from bs4 import BeautifulSoup
import re
import html as html_lib

def extract_text_from_html(html: str) -> str:
    """
    Perform high-quality text extraction from HTML using BeautifulSoup.
    Removes scripts, styles, navigation, sidebars, and other non-content elements.
    """
    if not html:
        return ""

    # Use lxml for speed and better handling of broken HTML if available
    try:
        soup = BeautifulSoup(html, "lxml")
    except Exception:
        soup = BeautifulSoup(html, "html.parser")

    # 1. Remove non-content elements
    for element in soup(["script", "style", "title", "header", "footer", "nav", "aside", 
                         "form", "svg", "noscript", "iframe", "button", "input", "textarea"]):
        element.decompose()

    # 2. Target common noise containers by ID or Class
    # Patterns for sidebars, ads, social, sharing, and navigation
    # We use boundary checks to avoid accidental matches (e.g., "pubnotepad" containing "ad")
    noise_patterns = re.compile(
        r"(^|[-_])(sidebar|ads?|social|share|top-?bar|bottom-?bar|menu|nav|related|newsletter|cookie|popup|modal)([-_]|$)",
        re.IGNORECASE
    )
    
    # Remove elements where ID or any class matches noise patterns
    for element in soup.find_all(attrs={"id": noise_patterns}):
        element.decompose()
        
    for element in soup.find_all(attrs={"class": noise_patterns}):
        # Double check that we aren't decomposing something just because it has a small class
        # but is actually a main content container (unlikely with these patterns but defensive)
        element.decompose()

    # 3. Specifically clean StackOverflow (very common use case)
    # SO uses #mainbar for content and #sidebar for noise
    if "stackoverflow.com" in str(soup.find("link", rel="canonical")):
        for so_noise in soup.find_all(id=re.compile(r"sidebar|sidebar-ads|hot-network-questions")):
            so_noise.decompose()

    # 4. Extract text
    # We use get_text with a separator to avoid merging words from different blocks
    text = soup.get_text(separator="\n")

    # 5. Final cleaning
    # Replace entities (BS4 usually does this, but being safe)
    text = html_lib.unescape(text)
    
    # Normalize whitespace: 
    # - convert non-breaking spaces to regular spaces
    # - collapse multiple spaces
    # - collapse multiple newlines into double newlines (paragraphs)
    # - trim
    text = text.replace("\xa0", " ")
    text = re.sub(r"[ \t]+", " ", text)
    
    lines = [line.strip() for line in text.splitlines()]
    # Remove empty lines but keep double newlines for paragraph separation
    cleaned_lines = []
    last_was_empty = False
    for line in lines:
        if line:
            cleaned_lines.append(line)
            last_was_empty = False
        elif not last_was_empty:
            cleaned_lines.append("")
            last_was_empty = True
            
    return "\n".join(cleaned_lines).strip()
