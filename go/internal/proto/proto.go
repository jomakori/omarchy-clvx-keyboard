// Package proto encodes and decodes the CLVX Request/Response envelope using
// low-level protobuf wire helpers. Field numbers are taken from the vendor
// ts-proto schema (app-protocol.ts): Request.type=1, oneof fields 2..16;
// Response.type=1, oneof fields 2..16 (badRequest=7).
package proto

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

// Request_Type values (app-protocol.ts Request_Type).
const (
	ReqGetSettings        = 0
	ReqSetSettings        = 1
	ReqGetDeviceInfo      = 2
	ReqHeartBeat          = 3
	ReqSetProfileSettings = 4
	ReqGetProfileSettings = 5
	ReqSetOSMode          = 8
	ReqGetDefaultSettings = 9
	ReqPerformFullReset   = 10
	ReqPerformRestart     = 12
	ReqGetDiagnostics     = 13
	ReqGetStatistics      = 14
)

// Request oneof field numbers (Request.<name>).
var reqField = map[uint32]protowire.Number{
	ReqGetSettings:        2,
	ReqSetSettings:        3,
	ReqSetProfileSettings: 4,
	ReqGetDeviceInfo:      5,
	ReqHeartBeat:          6,
	ReqGetProfileSettings: 7,
	ReqSetOSMode:          10,
	ReqGetDefaultSettings: 11,
	ReqPerformFullReset:   12,
	ReqPerformRestart:     14,
	ReqGetDiagnostics:     15,
	ReqGetStatistics:      16,
}

// Response_Type value for BAD_REQUEST.
const RespBadRequest = 4

// EncodeHeartBeat builds Request{type: HEARTBEAT, heartBeat: {activeProfile}}.
func EncodeHeartBeat(activeProfile uint32) []byte {
	var hb []byte
	if activeProfile != 0 {
		hb = protowire.AppendTag(hb, 1, protowire.VarintType)
		hb = protowire.AppendVarint(hb, uint64(activeProfile))
	}
	return encodeEnvelope(ReqHeartBeat, hb)
}

// EncodeEmptyRequest builds a Request with an empty oneof body (e.g. GET_SETTINGS).
func EncodeEmptyRequest(reqType uint32) ([]byte, error) {
	if reqType == ReqHeartBeat {
		return EncodeHeartBeat(0), nil
	}
	if _, ok := reqField[reqType]; !ok {
		return nil, fmt.Errorf("proto: unknown request type %d", reqType)
	}
	return encodeEnvelope(reqType, nil), nil
}

func encodeEnvelope(reqType uint32, body []byte) []byte {
	var b []byte
	if reqType != 0 { // proto3 default omitted
		b = protowire.AppendTag(b, 1, protowire.VarintType)
		b = protowire.AppendVarint(b, uint64(reqType))
	}
	b = protowire.AppendTag(b, reqField[reqType], protowire.BytesType)
	b = protowire.AppendBytes(b, body)
	return b
}

// Response is a generically decoded envelope: the Response_Type plus the
// raw oneof body (field number and bytes) for typed decoding by callers.
type Response struct {
	Type  uint32
	Field protowire.Number
	Body  []byte
}

// DecodeResponse parses Response.type and the single oneof body.
func DecodeResponse(pb []byte) (Response, error) {
	var r Response
	for len(pb) > 0 {
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return r, protowire.ParseError(n)
		}
		pb = pb[n:]
		m := protowire.ConsumeFieldValue(num, typ, pb)
		if m < 0 {
			return r, protowire.ParseError(m)
		}
		switch {
		case num == 1 && typ == protowire.VarintType:
			v, _ := protowire.ConsumeVarint(pb)
			r.Type = uint32(v)
		case typ == protowire.BytesType:
			body, _ := protowire.ConsumeBytes(pb)
			r.Field = num
			r.Body = body
		}
		pb = pb[m:]
	}
	return r, nil
}

// Fields walks a message and returns its top-level fields in order, for
// schema-free inspection (used by the CLI until typed decoders are generated).
type Field struct {
	Num   protowire.Number
	Type  protowire.Type
	Value any // uint64 for varint, []byte for bytes, uint32/uint64 for fixed
}

func Fields(pb []byte) ([]Field, error) {
	var out []Field
	for len(pb) > 0 {
		num, typ, n := protowire.ConsumeTag(pb)
		if n < 0 {
			return out, protowire.ParseError(n)
		}
		pb = pb[n:]
		switch typ {
		case protowire.VarintType:
			v, m := protowire.ConsumeVarint(pb)
			if m < 0 {
				return out, protowire.ParseError(m)
			}
			out = append(out, Field{num, typ, v})
			pb = pb[m:]
		case protowire.BytesType:
			v, m := protowire.ConsumeBytes(pb)
			if m < 0 {
				return out, protowire.ParseError(m)
			}
			out = append(out, Field{num, typ, v})
			pb = pb[m:]
		default:
			m := protowire.ConsumeFieldValue(num, typ, pb)
			if m < 0 {
				return out, protowire.ParseError(m)
			}
			out = append(out, Field{num, typ, nil})
			pb = pb[m:]
		}
	}
	return out, nil
}
