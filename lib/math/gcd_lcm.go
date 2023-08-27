package math

func GCD(a, b int64) int64 {
	for a > 0 && b > 0 {
		if a >= b {
			a = a % b
		} else {
			b = b % a
		}
	}
	if a > b {
		return a
	}
	return b
}

func GCDM(weights ...int64) int64 {
	l := len(weights)
	if l < 2 {
		return 0
	}
	res := GCD(weights[l-2], weights[l-1])
	if l == 2 {
		return res
	}
	return GCD(GCDM(weights[:l-1]...), res)
}

func LCM(a, b int64) int64 {
	return (a * b) / GCD(a, b)
}

func LCMM(a ...int64) int64 {
	l := len(a)
	if l < 2 {
		return 0
	}
	res := LCM(a[l-2], a[l-1])
	if l == 2 {
		return res
	}
	return LCM(LCMM(a[:l-1]...), res)
}
