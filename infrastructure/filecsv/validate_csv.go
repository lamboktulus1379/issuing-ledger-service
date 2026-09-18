package filecsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/lamboktulus1379/issuing-ledger-service/infrastructure/logger"
)

type IValidateCsvInterface interface {
	AppendAllData(data [][]string)
	AppendData(data []string)
	ReadData() ([]string, error)
	Close()
}

type ValidateCsv struct {
	File *os.File
}

func NewValidateCsv(file *os.File) IValidateCsvInterface {
	return &ValidateCsv{File: file}
}

func (validateCsv *ValidateCsv) AppendData(data []string) {
	w := csv.NewWriter(validateCsv.File)
	err := w.Write(data)
	if err != nil {
		return
	}

	if err := w.Error(); err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while AppendData to CSV")
	}
	fmt.Println("Appending succeed")
}

func (validateCsv *ValidateCsv) AppendAllData(data [][]string) {
	w := csv.NewWriter(validateCsv.File)
	err := w.WriteAll(data)
	if err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while AppendAllData to CSV")
	}

	if err := w.Error(); err != nil {
		logger.GetLogger().WithField("error", err).Error("Error while AppendAllData to CSV")
	}
	fmt.Println("Appending all succeed")
}

func (validateCsv *ValidateCsv) ReadData() ([]string, error) {
	var refNumbers []string

	r := csv.NewReader(validateCsv.File)

	// defer validateCsv.File.Close()

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error while reading CSV file")
			return nil, err
		}
		refNumbers = append(refNumbers, record[0])
	}
	logger.GetLogger().WithField("refNumbers", refNumbers).Info("Ref Numbers from CSV")
	return refNumbers, nil
}

func (validateCsv *ValidateCsv) Close() {
	defer func(File *os.File) {
		err := File.Close()
		if err != nil {
			logger.GetLogger().WithField("error", err).Error("Error while closing CSV file")
		}
	}(validateCsv.File)
}
