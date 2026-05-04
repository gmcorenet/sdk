package gmcore_crud

import "strings"

func DisplayLabelOrKey(displayValue, keyValue string) string {
	display := strings.TrimSpace(displayValue)
	if display != "" {
		return display
	}
	return strings.TrimSpace(keyValue)
}

func DisplayLabelWithSecondaryOrKey(displayValue, secondaryValue, keyValue string) string {
	display := strings.TrimSpace(displayValue)
	if display != "" {
		return display
	}
	secondary := strings.TrimSpace(secondaryValue)
	if secondary != "" {
		return secondary
	}
	return strings.TrimSpace(keyValue)
}

func DisplayNameEmailOrKey(displayName, email, keyValue string) string {
	name := strings.TrimSpace(displayName)
	mail := strings.TrimSpace(email)
	if name != "" && mail != "" {
		return name + " <" + mail + ">"
	}
	return DisplayLabelWithSecondaryOrKey(name, mail, keyValue)
}
