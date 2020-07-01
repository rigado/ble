package hci

type bondInfo struct {
	longTermKey []byte
	ediv        uint16
	randVal     uint64
	legacy      bool
}

type BondManager interface {
	Find(addr string) (BondInfo, error)
	Save(string, BondInfo) error
	Exists(addr string) bool
	Delete(addr string) error
}

type BondInfo interface {
	LongTermKey() []byte
	EDiv() uint16
	Random() uint64
	Legacy() bool
}

func NewBondInfo(longTermKey []byte, ediv uint16, random uint64, legacy bool) BondInfo {
	return &bondInfo{
		longTermKey: longTermKey,
		ediv:        ediv,
		randVal:     random,
		legacy:      legacy,
	}
}

func (b *bondInfo) LongTermKey() []byte {
	return b.longTermKey
}

func (b *bondInfo) EDiv() uint16 {
	return b.ediv
}

func (b *bondInfo) Random() uint64 {
	return b.randVal
}

func (b *bondInfo) Legacy() bool {
	return b.legacy
}
