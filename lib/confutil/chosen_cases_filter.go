package confutil

// Creates filter that returns true if ammo tag is in chosenCases. If no chosenCases provided - returns true
func IsChosenCase(checkCase string, chosenCases []string) bool {
	if len(chosenCases) == 0 {
		return true
	}

	for _, c := range chosenCases {
		if c == checkCase {
			return true
		}
	}
	return false
}
