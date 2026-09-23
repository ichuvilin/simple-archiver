package main

type SimpleArchiver struct {
	inputPath  string
	outputPath string
	buffer     []byte
}

func NewArchiver(inputPath string) *SimpleArchiver {
	return &SimpleArchiver{
		inputPath: inputPath,
		buffer:    make([]byte, 1024*8),
	}
}

func (sa *SimpleArchiver) compressEmpty(data []byte) []byte {
	if len(data) == 0 {
		return []byte{}
	}

	return data
}

func (sa *SimpleArchiver) countRepeating(data []byte) []byte {
	l := 0
	r := 0
	result := make([]byte, 0)

	for r < len(data) {
		if data[l] != data[r] {
			result = append(result, byte(r-l))
			result = append(result, data[l])
			l = r
		}
		r += 1
	}
	result = append(result, byte(len(data)-l))
	result = append(result, data[l])

	return result
}

func (sa *SimpleArchiver) createControlByte(count int, isCompressed bool) byte {
	if count > 127 {
		count = 127
	}

	if isCompressed {
		return byte(128 + count)
	}

	return byte(count)
}

func (sa *SimpleArchiver) encode(data []byte) []byte {
	result := make([]byte, 0)

	for i := 0; i < len(data); {
		repeatCount := countRepeating(data, i)

		if repeatCount >= 4 {
			count := repeatCount

			if count > 127 {
				count = 127
			}

			result = append(
				result,
				sa.createControlByte(count, true),
			)
			result = append(result, data[i])

			i += count
			continue
		}

		start := i
		length := 0

		for i < len(data) && length < 127 {
			repeatCount = countRepeating(data, i)

			if repeatCount >= 3 && length > 0 {
				break
			}

			i++
			length++
		}

		result = append(
			result,
			sa.createControlByte(length, false),
		)
		result = append(result, data[start:i]...)
	}

	return result
}

func countRepeating(data []byte, start int) int {
	count := 1

	for start+count < len(data) &&
		data[start] == data[start+count] {
		count++
	}

	return count
}

func main() {
}
