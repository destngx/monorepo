package domain

import (
	"encoding/json"
	"fmt"
	"strings"
)

// fillListLimit bounds how many names of one list the model sees.
const fillListLimit = 300

// FillInstruction is the model's task for a form, built from its schema
// (WM-FILL-2). guide is the optional tasks/form-fill prompt.
func FillInstruction(form FillForm, context FillContext, current map[string]any, guide string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "You fill in the %q form of a personal finance app from what the user wrote or the images show. Today is %s (%s).\n",
		form.Title, context.Today.Format("2006-01-02"), context.Today.Weekday())
	b.WriteString(`Rules:
- Read only what the request says or the images show. Never invent a value; leave a field out when it is not given.
- Resolve relative dates from today ("yesterday", "next month", "June 2027").
- Money is a plain number in the field's currency, VND unless the field says otherwise: "50M" or "50tr" is 50000000, "260k" is 260000.
- A name field must be copied from its list exactly when the request means one of them.
- When something is unclear or missing, say so in notes, in one short sentence each, in English.
`)
	if len(current) > 0 {
		b.WriteString("- The form already holds the values below. Return only the fields the request changes; leave every other field out.\n")
	}
	if form.Guide != "" {
		b.WriteString("- " + form.Guide + "\n")
	}
	if guide = strings.TrimSpace(guide); guide != "" {
		b.WriteString("\nThe user's guidance:\n" + guide + "\n")
	}
	b.WriteString("\nFields:\n")
	writeFillFields(&b, form.Fields)
	if len(form.Rows) > 0 {
		fmt.Fprintf(&b, "\nEach %s in \"rows\" has these fields:\n", form.RowsLabel)
		writeFillFields(&b, form.Rows)
	}
	for _, list := range form.lists() {
		names := context.Lists[list]
		if len(names) == 0 {
			fmt.Fprintf(&b, "\nThe %s list is empty.\n", list)
			continue
		}
		if len(names) > fillListLimit {
			names = names[:fillListLimit]
		}
		fmt.Fprintf(&b, "\nThe %s list:\n%s\n", list, strings.Join(names, "\n"))
	}
	if facts := strings.TrimSpace(context.Facts); facts != "" {
		b.WriteString("\nWhat the app knows:\n" + facts + "\n")
	}
	if len(current) > 0 {
		payload, _ := json.Marshal(current)
		b.WriteString("\nThe form now holds:\n" + string(payload) + "\n")
	}
	b.WriteString("\nIMPORTANT: Respond with ONLY a valid JSON object (no markdown, no backticks, no explanation) in this format:\n")
	example := `{"values": {"<field>": <value>}`
	if len(form.Rows) > 0 {
		example += `, "rows": [{"<field>": <value>}]`
	}
	example += `, "notes": ["<short note>"]}`
	b.WriteString(example + "\n")
	return b.String()
}

func writeFillFields(b *strings.Builder, fields []FillField) {
	for _, field := range fields {
		fmt.Fprintf(b, "- %s (%s): %s", field.Name, field.Label, fillTypeRule(field))
		if field.Hint != "" {
			b.WriteString(". " + field.Hint)
		}
		b.WriteString("\n")
	}
}

func fillTypeRule(field FillField) string {
	switch field.Type {
	case FillMoney:
		return "an amount as a number"
	case FillNumber:
		return "a number"
	case FillPercent:
		return "a percentage as a number, 5 for 5%"
	case FillDay:
		return "a day of the month, 1-31"
	case FillDate:
		return "a date, yyyy-mm-dd"
	case FillMonth:
		return "a month, yyyy-mm"
	case FillChoice:
		return "one of " + strings.Join(field.Choices, ", ")
	case FillFlag:
		return "true or false"
	case FillName:
		if field.New {
			return fmt.Sprintf("a name from the %s list, or a new name as written", field.List)
		}
		return fmt.Sprintf("a name from the %s list", field.List)
	case FillNames:
		if field.New {
			return fmt.Sprintf("a JSON list of names from the %s list, or new names as written", field.List)
		}
		return fmt.Sprintf("a JSON list of names from the %s list", field.List)
	}
	return "text"
}
