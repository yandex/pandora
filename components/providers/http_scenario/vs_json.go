package httpscenario

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
)

type VariableSourceJSON struct {
	Name  string `yaml:"name"`
	File  string `yaml:"file"`
	fs    afero.Fs
	store any
}

func (v *VariableSourceJSON) GetName() string {
	return v.Name
}

func (v *VariableSourceJSON) GetVariables() any {
	return v.store
}

func (v *VariableSourceJSON) Init() (err error) {
	const op = "VariableSourceJSON.Init"
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
	decoder := json.NewDecoder(file)
	var data any
	err = decoder.Decode(&data)
	if err != nil {
		return fmt.Errorf("%s decoder.Decode %w", op, err)
	}
	v.store = data

	return nil
}

func NewVSJson(cfg VariableSourceJSON, fs afero.Fs) (VariableSource, error) {
	cfg.fs = fs
	return &cfg, nil
}

var _ VariableSource = (*VariableSourceJSON)(nil)
