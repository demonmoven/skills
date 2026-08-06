package internal

type BitMap struct {
	data []uint64
}

func NewBitMap(size int) *BitMap {
	return &BitMap{data: make([]uint64, (size+63)/64)}
}

func (b *BitMap) Add(i int) {
	for len(b.data)*64 <= i {
		b.data = append(b.data, 0)
	}
	b.data[i/64] |= uint64(1) << (i % 64)
}

func (b *BitMap) Has(i int) bool {
	return i < len(b.data)*64 && (b.data[i/64]&(uint64(1)<<(i%64))) != 0
}
