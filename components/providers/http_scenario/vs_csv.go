package httpscenario

import (
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/afero"
)

type VariableSourceCsv struct {
	Name            string
	File            string
	Fields          []string
	IgnoreFirstLine bool `config:"ignore_first_line"`
	Delimiter       string
	fs              afero.Fs
	store           []map[string]string
}

func (v *VariableSourceCsv) GetName() string {
	return v.Name
}

func (v *VariableSourceCsv) GetVariables() any {
	return v.store
}

func (v *VariableSourceCsv) Init() (err error) {
	const op = "VariableSourceCsv.Init"
	var file afero.File
	file, err = v.fs.Open(v.File)
	if err != nil {
		return fmt.Errorf("%s fs.Open %w", op, err)
	}
	defer func() {
		closeErr := file.Close()
		if closeErr != nil {
			if err != nil {
				err = fmt.Errorf("%s multiple errors faced: %w, with close err: %s", op, err, closeErr)
			} else {
				err = fmt.Errorf("%s, %w", op, closeErr)
			}
		}
	}()

	store, err := readCsv(file, v.IgnoreFirstLine, v.Delimiter, v.Fields)
	if err != nil {
		return fmt.Errorf("%s readCsv %w", op, err)
	}
	v.store = store

	return nil
}

func readCsv(file afero.File, ignoreFirstLine bool, delimiter string, fields []string) ([]map[string]string, error) {
	const op = "readCsv"
	reader := csv.NewReader(file)
	if delimiter != "" {
		reader.Comma = rune(delimiter[0])
	}
	for i := range fields {
		fields[i] = strings.Replace(fields[i], " ", "_", -1)
	}
	result := make([]map[string]string, 0)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("%s csv.Read %w", op, err)
		}
		if len(fields) == 0 {
			fields = make([]string, len(record))
			for i := range record {
				fields[i] = strings.Replace(record[i], " ", "_", -1)
			}
		}
		if ignoreFirstLine {
			ignoreFirstLine = false
			continue
		}
		row := make(map[string]string)
		for i, field := range fields {
			if field == "" {
				field = strconv.Itoa(i)
			}
			if i >= len(record) {
				row[field] = ""
			} else {
				row[field] = record[i]
			}
		}
		result = append(result, row)
	}
	return result, nil
}

func NewVSCSV(cfg VariableSourceCsv, fs afero.Fs) (VariableSource, error) {
	cfg.fs = fs
	return &cfg, nil
}

var _ VariableSource = (*VariableSourceCsv)(nil)
