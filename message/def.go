package message

func definitionOutputFormat(format string) string {
	if format == "" {
		return ""
	}
	// Please strictly adhere to this output format:
	// The return format is as follows, where "{}" represents a placeholder.
	// Please provide your response in JSON format:
	// - Respond using JSON
	return `## Output Format
Please strictly adhere to this output format, do not include any extra content, where "{}" represents a placeholder:

` + format
}
