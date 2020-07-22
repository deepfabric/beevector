package util

// EncodeXID encode xid
func EncodeXID(uid, pid uint64) uint64 {
	return (uid << 34) + pid
}

// DecodeXID decode xid
func DecodeXID(xid uint64) (uint64, uint64) {
	return xid >> 34, xid & 0x3FFFFFFFF
}
