package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	av "github.com/elodina/go-avro"
	"github.com/stretchr/testify/require"
)

var (
	typString    = reflect.TypeOf("string")
	typByteSlice = reflect.TypeOf([]byte{})
	typInt       = reflect.TypeOf(int32(1))
	typLong      = reflect.TypeOf(int64(1))
	typFloat     = reflect.TypeOf(float32(1.1))
	typDouble    = reflect.TypeOf(float64(1.2))
	typBool      = reflect.TypeOf(true)
	typNil       = reflect.TypeOf(nil)
)

func AvroSchemaType(schema av.Schema) reflect.Type {
	switch schema.Type() {
	case av.Null:
		return typNil
	case av.Boolean:
		return typBool
	case av.Int:
		return typInt
	case av.Long:
		return typLong
	case av.Float:
		return typFloat
	case av.Double:
		return typDouble
	case av.Bytes:
		return typByteSlice
	case av.String:
		return typString
	case av.Enum:
		return typString
	case av.Array:
		shArr := schema.(*av.ArraySchema)
		return reflect.SliceOf(AvroSchemaType(shArr.Items))
	case av.Map:
		shMap := schema.(*av.MapSchema)
		return reflect.MapOf(typString, AvroSchemaType(shMap.Values))
	case av.Record:
		shRec := schema.(*av.RecordSchema)
		fields := []reflect.StructField{}
		for i, f := range shRec.Fields {
			sf := reflect.StructField{
				Name: fmt.Sprintf("Field%v", i),
				Tag:  reflect.StructTag(fmt.Sprintf(`json:%v`, f.Name)),
				Type: AvroSchemaType(f.Type),
			}
			fields = append(fields, sf)
		}
		return reflect.StructOf(fields)
	}

	panic(fmt.Sprintf("unsupported avro schema: %#v", schema))
}

func AvroEncode(schema string, in []byte) ([]byte, error) {
	var (
		err error
		avs av.Schema
		typ reflect.Type
		ptr interface{}
		val interface{}
		wrt = av.NewGenericDatumWriter()
		buf = new(bytes.Buffer)
		enc = av.NewBinaryEncoder(buf)
	)

	if avs, err = av.ParseSchema(schema); err != nil {
		return []byte{}, fmt.Errorf("failed to parse schema err=%v", err)
	}
	wrt.SetSchema(avs)

	typ = AvroSchemaType(avs)

	if typ != nil {
		ptr = reflect.New(typ).Interface()
	}

	// double & for nil case
	if err = json.Unmarshal(in, &ptr); err != nil {
		return []byte{}, fmt.Errorf("failed unmarshal %s into %#v err=%v", in, ptr, err)
	}

	if ptr != nil {
		val = reflect.ValueOf(ptr).Elem().Interface()
	}

	if err = wrt.Write(val, enc); err != nil {
		return []byte{}, fmt.Errorf("failed to avro encode err=%v", err)
	}

	return buf.Bytes(), nil
}

func AvroDecode(schema string, in []byte) ([]byte, error) {
	var (
		err error
		out interface{}
		avs av.Schema
		red = av.NewGenericDatumReader()
		dec = av.NewBinaryDecoder(in)
	)

	if avs, err = av.ParseSchema(schema); err != nil {
		return []byte{}, fmt.Errorf("failed to parse schema err=%v", err)
	}

	if len(in) == 0 && avs.Type() == av.Null {
		return json.Marshal(nil)
	}

	red.SetSchema(avs)
	if err = red.Read(&out, dec); err != nil {
		return []byte{}, fmt.Errorf("failed to avro decode err=%v", err)
	}

	return json.Marshal(out)
}

func TestAvro(t *testing.T) {
	data := []struct {
		name   string
		schema string
		in     string
	}{
		{name: "null", schema: `{"type": "null"}`, in: `null`},
		{name: "boolean", schema: `{"type": "boolean"}`, in: `true`},
		{name: "int", schema: `{"type": "int"}`, in: `123`},
		{name: "long", schema: `{"type": "long"}`, in: `123`},
		{name: "float", schema: `{"type": "float"}`, in: `123.1`},
		{name: "double", schema: `{"type": "double"}`, in: `123.2`},
		{name: "bytes", schema: `{"type": "bytes"}`, in: `"SEVMTE8="`},
		{name: "string", schema: `{"type": "string"}`, in: `"DONE"`},
		{name: "enum", schema: `{"type": "enum", "name":"suit", "symbols":["SPADES", "HEARTS"]}`, in: `"HEARTS"`},
		{name: "map of longs", schema: `{"type": "map", "values": "long"}`, in: `{"abc":123}`},
		{name: "map of strings", schema: `{"type": "map", "values": "string"}`, in: `{"abc":"hans"}`},
		{name: "array of bool", schema: `{"type": "array", "items": "boolean"}`, in: `[true,false]`},
		{name: "record", schema: `{"type": "record", "name": "person","fields": [{"name":"first-name","type":"string"}]}`, in: `{"first-name": "hans"}`},
	}

	// TODO:
	// record
	// union uh??
	// fixed

	for _, d := range data {
		encoded, err := AvroEncode(d.schema, []byte(d.in))
		require.Nil(t, err, d.name)

		decoded, err := AvroDecode(d.schema, encoded)
		require.Nil(t, err, d.name)
		require.Equal(t, d.in, string(decoded), d.name)
	}
}
