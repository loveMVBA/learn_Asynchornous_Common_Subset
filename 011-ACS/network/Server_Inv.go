package network

type Inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}
