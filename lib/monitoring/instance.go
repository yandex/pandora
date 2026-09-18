package monitoring

import (
	"expvar"
	"strconv"
	"sync/atomic"
)

func NewInstanceTracker(name string) *InstanceTracker {
	v := &InstanceTracker{}
	expvar.Publish(name, v)
	return v
}

// Два счётчика вместо множества id под общим мьютексом: id не использовался,
// а OnStart/OnFinish зовутся дважды на каждый выстрел из всех инстансов сразу.
type InstanceTracker struct {
	cur atomic.Int64
	max atomic.Int64
}

func (u *InstanceTracker) String() string {
	return strconv.FormatInt(u.cur.Load(), 10)
}

func (u *InstanceTracker) OnStart(_ int) {
	cur := u.cur.Add(1)
	for {
		m := u.max.Load()
		if cur <= m || u.max.CompareAndSwap(m, cur) {
			return
		}
	}
}

func (u *InstanceTracker) OnFinish(_ int) {
	u.cur.Add(-1)
}

// Пара Load+Swap не атомарна: пик, набранный ровно между ними, уедет в следующее окно.
// Для гейджа "занято инстансов" это терпимо, мьютекс ради точности тут дороже.
func (u *InstanceTracker) Flush() int {
	return int(u.max.Swap(u.cur.Load()))
}
