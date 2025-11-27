package datastruct

type HashTable struct {
	Field288    int32
	Field289    bool
	BucketCount int32
	Buckets     []*Linkable
}

func NewHashTable(arg0 int32, arg1 int32) *HashTable {
	t := &HashTable{
		Field288:    4277,
		Field289:    false,
		BucketCount: arg1,
		Buckets:     make([]*Linkable, arg1),
	}
	if arg0 < 9 || arg0 > 9 {
		t.Field289 = !t.Field289
	}
	for var3 := int32(0); var3 < arg1; var3++ {
		t.Buckets[var3] = new(Linkable)
		var4 := t.Buckets[var3]
		var4.Next = var4
		var4.Prev = var4
	}
	return t
}

func (t *HashTable) Get(arg0 int64) *Linkable {
	var3 := t.Buckets[arg0&(int64(t.BucketCount)-1)]
	for var4 := var3.Next; var4 != var3; var4 = var4.Next {
		if var4.Key == arg0 {
			return var4
		}
	}
	return nil
}

func (t *HashTable) Put(arg0 int64, arg1 int32, arg2 *Linkable) {
	if arg2.Prev != nil {
		arg2.Unlink()
	}
	var5 := t.Buckets[arg0&(int64(t.BucketCount)-1)]
	arg2.Prev = var5.Prev
	if arg1 < 0 {
		arg2.Next = var5
		arg2.Prev.Next = arg2
		arg2.Next.Prev = arg2
		arg2.Key = arg0
	}
}
