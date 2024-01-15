package vs

type VariableSource interface {
	GetName() string
	GetVariables() any
	Init() error
}
