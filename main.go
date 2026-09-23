package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
)

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

func (sa *SimpleArchiver) compress(data []byte) []byte {
	data = sa.compressEmpty(data)

	result := make([]byte, 0)

	for i := 0; i < len(data); {
		count := countRepeating(data, i)

		// Сжимаем последовательность из 4+ одинаковых байт.
		if count >= 4 {
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
			count = countRepeating(data, i)

			if count >= 3 && length > 0 {
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

func (sa *SimpleArchiver) decompress(data []byte) []byte {
	data = sa.compressEmpty(data)

	i := 0

	result := make([]byte, 0)

	for i < len(data) {
		control := data[i]
		i++

		isCompressed := control&0x80 != 0
		length := int(control & 0x7F)

		fmt.Printf("Управляющий байт: 0x%02X\n", control)

		if isCompressed {
			fmt.Printf("  Тип: сжатая, длина: %d\n\n", length)

			value := data[i]
			i++

			for j := 0; j < length; j++ {
				result = append(result, value)
			}
		} else {
			fmt.Printf("  Тип: несжатая, длина: %d\n\n", length)
			result = append(result, data[i:i+length]...)
			i += length
		}
	}

	return result
}

func (sa *SimpleArchiver) CompressFile(inputPath, outputPath string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open input file: %w", err)
	}
	defer input.Close()

	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	defer writer.Flush()

	filename := filepath.Base(inputPath)
	filenameBytes := []byte(filename)

	err = writer.WriteByte(byte(len(filenameBytes)))
	if err != nil {
		return fmt.Errorf("error during write byte: %w", err)
	}
	_, err = writer.Write(filenameBytes)
	if err != nil {
		return fmt.Errorf("error during write: %w", err)
	}

	return nil
}

func main() {
	archiver := SimpleArchiver{}

	tests := [][]byte{
		{},
		{'A'},
		{'A', 'B', 'C'},
		{'A', 'A', 'A', 'A'},
		{'A', 'A', 'A', 'A', 'B', 'C'},
	}

	for _, data := range tests {
		compressed := archiver.compress(data)
		decompressed := archiver.decompress(compressed)

		if !bytes.Equal(data, decompressed) {
			fmt.Printf("original=%v, result=%v\n", data, decompressed)
			continue
		}

		fmt.Printf("%v\n", data)
	}
}
