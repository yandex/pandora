package warmup

type WarmedUp interface {
	WarmUp(*Options) (interface{}, error)
}
