package filecsv

import (
	"fmt"
	"os"
)

type IValidateFile interface {
	AppendAllData(data [][]string)
	AppendData(data []string)
	ReadData() ([]string, error)
	Close()
}

type ValidateFile struct {
	File *os.File
}

func NewValidateFile(file *os.File) IValidateFile {
	return &ValidateFile{File: file}
}

func (validateFile *ValidateFile) AppendData(data []string) {
	_, err := validateFile.File.WriteString(data[0] + "\n")
	if err != nil {
		return
	}
}

func (validateFile *ValidateFile) AppendAllData(data [][]string) {
	for _, row := range data {
		writeString, err := validateFile.File.WriteString(row[0] + "\n")
		if err != nil {
			return
		}
		fmt.Println(string(rune(writeString)))
	}
}

func (validateFile *ValidateFile) ReadData() ([]string, error) {
	var refNumbers []string

	// defer validateFile.File.Close()

	// Iterate through the records
	for {
		// Read each record from csv
		record := make([]byte, 10)
		_, err := validateFile.File.Read(record)
		if err != nil {
			break
		}

		refNumbers = append(refNumbers, string(record))
	}

	return refNumbers, nil
}

func (validateFile *ValidateFile) Close() {
	err := validateFile.File.Close()
	if err != nil {
		return
	}
}
