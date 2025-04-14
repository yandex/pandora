package filter

const (
	FilterAll     = "all"
	FilterWarning = "warning"
	FilterError   = "error"
)

var errorGrpcCodes = []int64{2, 4, 12, 13, 14, 15} // UNKNOWN, DEADLINE_EXCEEDED, UNIMPLEMENTED, INTERNAL, UNAVAILABLE, DATA_LOSS
