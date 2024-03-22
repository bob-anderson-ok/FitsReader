package main

//func conv8ByteSliceToFloat64(p []byte, offset int) (float64, uint64) {
//	valueUint64 := binary.LittleEndian.Uint64(p[offset : offset+8])
//	return float64(valueUint64), valueUint64
//}

//func conv4ByteSliceToFloat64(p []byte, offset int) (float64, uint32) {
//	valueUint32 := binary.LittleEndian.Uint32(p[offset : offset+4])
//	return float64(valueUint32), valueUint32
//}

//func conv2ByteSliceToFloat64(p []byte, offset int) (float64, uint16) {
//	valueUint16 := binary.LittleEndian.Uint16(p[offset : offset+2])
//	return float64(valueUint16), valueUint16
//}

//func conv1ByteSliceToFloat64(p []byte, offset int) (float64, uint8) {
//	return float64(p[offset]), p[offset]
//}

//func convByteSliceToFloat64(p []byte, offset int) (float64, uint64) {
//	valueUint64 := binary.LittleEndian.Uint64(p[offset : offset+8])
//	return float64(valueUint64), valueUint64
//}

//func convUint32ToBytes(b []byte, v uint32) []byte {
//	// LittleEndian order
//	return append(b,
//		byte(v),
//		byte(v>>8),
//		byte(v>>16),
//		byte(v>>24),
//	)
//}

//func convUint64ToBytes(b []byte, v uint64) []byte {
//	// LittleEndian order
//	return append(b,
//		byte(v),
//		byte(v>>8),
//		byte(v>>16),
//		byte(v>>24),
//		byte(v>>32),
//		byte(v>>40),
//		byte(v>>48),
//		byte(v>>56),
//	)
//}
