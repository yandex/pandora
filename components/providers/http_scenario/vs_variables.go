package httpscenario

type VariableSourceVariables struct {
	Name      string
	Variables map[string]any
}

func (v *VariableSourceVariables) GetName() string {
	return v.Name
}

func (v *VariableSourceVariables) GetVariables() any {
	return v.Variables
}

func (v *VariableSourceVariables) Init() error {
	return nil
}
