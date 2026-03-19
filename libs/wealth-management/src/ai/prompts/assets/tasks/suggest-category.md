Your task is to accurately suggest the best category for a transaction based on the payee name.
Payee: "{{payee}}"

Pick the most suitable category for this transaction from the following list:
{{categories}}

RULES:

- Return ONLY the exact category name from the list.
- Do not include any explanation or punctuation.
- If no category seems a perfect fit, pick the closest one.
- If it is completely ambiguous, return the first item in the list.
