package ammo

type Ammo struct {
	Tag       string                 `json:"tag"`
	Call      string                 `json:"call"`
	Metadata  map[string]string      `json:"metadata"`
	Payload   map[string]interface{} `json:"payload"`
	id        uint64
	isInvalid bool
}

func (a *Ammo) Reset(tag string, call string, metadata map[string]string, payload map[string]interface{}) {
	*a = Ammo{tag, call, metadata, payload, 0, false}
}

func (a *Ammo) SetID(id uint64) {
	a.id = id
}

func (a *Ammo) ID() uint64 {
	return a.id
}

func (a *Ammo) Invalidate() {
	a.isInvalid = true
}

func (a *Ammo) IsInvalid() bool {
	return a.isInvalid
}

func (a *Ammo) IsValid() bool {
	return !a.isInvalid
}
