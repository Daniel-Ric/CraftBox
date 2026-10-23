package utils




func normalizeLegacyJSONNumbers(input []byte) []byte {
	output := make([]byte, 0, len(input))
	boundary := true
	for i := 0; i < len(input); {
		start := i
		switch input[i] {
		case '"':
			i++
			for i < len(input) {
				if input[i] == '\\' {
					i++
					if i < len(input) {
						i++
					}
					continue
				}
				if input[i] == '"' {
					i++
					break
				}
				i++
			}
			output = append(output, input[start:i]...)
			boundary = false
			continue
		case '/':
			if i+1 < len(input) && input[i+1] == '/' {
				i += 2
				for i < len(input) && input[i] != '\n' {
					i++
				}
				output = append(output, input[start:i]...)
				continue
			}
			if i+1 < len(input) && input[i+1] == '*' {
				i += 2
				for i+1 < len(input) && !(input[i] == '*' && input[i+1] == '/') {
					i++
				}
				if i+1 < len(input) {
					i += 2
				} else {
					i = len(input)
				}
				output = append(output, input[start:i]...)
				continue
			}
		}
		digit := i
		if input[i] == '-' {
			digit++
		}
		if boundary && digit < len(input) && input[digit] >= '0' && input[digit] <= '9' {
			end := digit
			for end < len(input) && input[end] >= '0' && input[end] <= '9' {
				end++
			}
			first := digit
			for first+1 < end && input[first] == '0' {
				first++
			}
			output = append(output, input[i:digit]...)
			output = append(output, input[first:end]...)
			i = end
			boundary = false
			continue
		}
		output = append(output, input[i])
		switch input[i] {
		case ':', ',', '[':
			boundary = true
		case ' ', '\t', '\r', '\n':
		default:
			boundary = false
		}
		i++
	}
	return output
}
