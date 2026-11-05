package proto

import "google.golang.org/protobuf/encoding/protowire"

// RawBytes retains the original protobuf message. Profile writes emit this
// message verbatim while its decoded view is unchanged, preserving fields
// added by newer firmware.
type RawBytes struct {
	Raw []byte `json:"-"`
}

func rawBytes(pb []byte) RawBytes {
	return RawBytes{Raw: append([]byte(nil), pb...)}
}

// walkFields calls fn with the field body in the form returned by
// protowire.ConsumeFieldValue. Unknown fields and known fields with an
// unexpected wire type are simply ignored by the typed decoders.
func walkFields(pb []byte, fn func(protowire.Number, protowire.Type, []byte) error) error {
	for len(pb) > 0 {
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return protowire.ParseError(n)
		}
		pb = pb[n:]
		m := protowire.ConsumeFieldValue(num, typ, pb)
		if m < 0 {
			return protowire.ParseError(m)
		}
		if err := fn(num, typ, pb[:m]); err != nil {
			return err
		}
		pb = pb[m:]
	}
	return nil
}

func fieldVarint(body []byte) (uint64, bool) {
	v, n := protowire.ConsumeVarint(body)
	return v, n >= 0
}

func fieldBytes(body []byte) ([]byte, bool) {
	v, n := protowire.ConsumeBytes(body)
	return v, n >= 0
}

func decodeChild(body []byte, decode func([]byte) error) error {
	pb, ok := fieldBytes(body)
	if !ok {
		return nil
	}
	return decode(pb)
}
