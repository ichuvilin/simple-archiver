package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type SimpleArchiver struct {
	inputPath  string
	outputPath string
	buffer     []byte
}

type model struct {
	archiver  *SimpleArchiver
	state     string
	inputPath string
	choices   []string
	cursor    int
	err       error
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

	reader := bufio.NewReader(input)
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

	for {
		n, err := reader.Read(sa.buffer)

		if n > 0 {
			compressed := sa.compress(sa.buffer[:n])
			blockSize := uint16(len(compressed))

			if err = writer.WriteByte(byte(blockSize >> 8)); err != nil {
				return fmt.Errorf("error during write block: %w", err)
			}

			if err = writer.WriteByte(byte(blockSize)); err != nil {
				return fmt.Errorf("error during write block: %w", err)
			}

			_, err = writer.Write(compressed)
			if err != nil {
				return fmt.Errorf("error during write compressed: %w", err)
			}
		}

		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			return fmt.Errorf("error during read: %w", err)
		}
	}

	return nil
}

func (sa *SimpleArchiver) DecompressFile(inputPath, outputDir string) error {
	input, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("open archive: %w", err)
	}
	defer input.Close()

	reader := bufio.NewReader(input)

	nameLength, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("read filename length: %w", err)
	}

	filenameBytes := make([]byte, int(nameLength))
	_, err = io.ReadFull(reader, filenameBytes)
	if err != nil {
		return fmt.Errorf("read filename: %w", err)
	}

	filename := string(filenameBytes)
	outputPath := filepath.Join(outputDir, filename)

	output, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer output.Close()

	writer := bufio.NewWriter(output)
	defer writer.Flush()

	for {
		high, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read block size: %w", err)
		}

		low, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("read block size: %w", err)
		}

		blockSize := uint16(high)<<8 | uint16(low)

		compressed := make([]byte, blockSize)

		_, err = io.ReadFull(reader, compressed)
		if err != nil {
			return fmt.Errorf("read compressed block: %w", err)
		}

		decompressed := sa.decompress(compressed)

		_, err = writer.Write(decompressed)
		if err != nil {
			return fmt.Errorf("write decompressed block: %w", err)
		}
	}

	return nil
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) updateMenu(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < len(m.choices)-1 {
			m.cursor++
		}

	case "enter":
		switch m.cursor {
		case 0:
			m.state = "compress"
		case 1:
			m.state = "decompress"
		case 2:
			return m, tea.Quit
		}

	case "q", "ctrl+c":
		return m, tea.Quit
	}

	return m, nil
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.state == "menu" {
			return m.updateMenu(msg)
		}
	}

	return m, nil
}

func (m model) viewMenu() string {
	var b strings.Builder

	b.WriteString("Простой архиватор\n\n")

	for i, choice := range m.choices {
		cursor := " "

		if m.cursor == i {
			cursor = ">"
		}

		fmt.Fprintf(&b, "%s %s\n", cursor, choice)
	}

	b.WriteString("\n↑/↓ — навигация • Enter — выбрать • q — выход\n")

	return b.String()
}

func (m model) View() string {
	switch m.state {
	case "menu":
		return m.viewMenu()
	default:
		return ""
	}
}

func initialModel() model {
	return model{
		archiver: NewArchiver("aadad"),
		state:    "menu",
		choices: []string{
			"Сжать файл",
			"Распаковать файл",
			"Выход",
		},
	}
}

func main() {
	p := tea.NewProgram(initialModel())

	if _, err := p.Run(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
