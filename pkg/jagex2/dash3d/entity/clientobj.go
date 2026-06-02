package entity

type ClientObj struct {
	Index int
	Count int
}

func NewClientObj() *ClientObj {
	return new(ClientObj)
}
