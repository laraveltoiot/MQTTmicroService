package utils

import (
	"strings"
)

// TopicMatchesFilter checks if a topic matches a filter
// The filter can contain wildcards:
// - '+' matches exactly one level
// - '#' matches zero or more levels (must be the last character)
func TopicMatchesFilter(topic, filter string) bool {
	// Split the topic and filter into levels
	topicLevels := strings.Split(topic, "/")
	filterLevels := strings.Split(filter, "/")

	// If the filter ends with #, it matches any number of levels
	if strings.HasSuffix(filter, "#") {
		// Remove the # from the filter
		filterLevels = filterLevels[:len(filterLevels)-1]

		// The topic must have at least as many levels as the filter (excluding the #)
		if len(topicLevels) < len(filterLevels) {
			return false
		}
	} else {
		// If the filter doesn't end with #, the topic must have exactly the same number of levels
		if len(topicLevels) != len(filterLevels) {
			return false
		}
	}

	// Check each level
	for i := 0; i < len(filterLevels); i++ {
		// If the filter level is +, it matches any topic level
		if filterLevels[i] == "+" {
			continue
		}

		// Otherwise, the levels must match exactly
		if filterLevels[i] != topicLevels[i] {
			return false
		}
	}

	return true
}
