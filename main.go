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

func main() {
}
