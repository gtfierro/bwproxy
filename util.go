package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	bw2 "gopkg.in/immesys/bw2bind.v5"
)

func toString(v interface{}) string {
	if val, ok := v.(string); ok {
		return val
	}
	return fmt.Sprintf("%s", v)
}

func getString(key string, m map[string]interface{}) string {
	if val_if, found := m[key]; found {
		return toString(val_if)
	}
	return ""
}

func getBool(key string, m map[string]interface{}) bool {
	if val_if, found := m[key]; found {
		b, err := strconv.ParseBool(toString(val_if))
		if err != nil {
			return false
		}
		return b
	}
	return false
}

func isType(po, df string) bool {
	parts := strings.SplitN(df, "/", 2)
	var mask int
	var err error
	if len(parts) != 2 {
		mask = 32
	} else {
		mask, err = strconv.Atoi(parts[1])
		if err != nil {
			panic("malformed masked dot form")
		}
	}
	ponum := bw2.FromDotForm(parts[0])
	mypo := bw2.FromDotForm(po)
	return (ponum >> uint(32-mask)) == (mypo >> uint(32-mask))
}

func po2iface(po bw2.PayloadObject) (interface{}, error) {

	if po.IsTypeDF(bw2.PODFMaskMsgPack) {
		var thing interface{}
		if err := po.(bw2.MsgPackPayloadObject).ValueInto(&thing); err != nil {
			return []byte{}, errors.Wrap(err, "Could not unpack msgpack")
		}
		return thing, nil
	} else if po.IsTypeDF(bw2.PODFMaskText) {
		return po.(bw2.TextPayloadObject).Value(), nil
	} else {
		log.Error("Returning text for", po.GetPODotNum())
		return po.TextRepresentation(), nil
	}

	return []byte{}, errors.New("Cannot unmarshal")
}

func iface2po(ponum string, v interface{}) (bw2.PayloadObject, error) {
	if isType(ponum, bw2.PODFMaskMsgPack) {
		return bw2.CreateMsgPackPayloadObject(bw2.FromDotForm(ponum), v)
	} else if isType(ponum, bw2.PODFMaskText) {
		return bw2.CreateTextPayloadObject(bw2.FromDotForm(ponum), toString(v)), nil
	}
	return nil, nil
}

func datums2json(datums []interface{}) ([]byte, error) {
	for idx, datum := range datums {
		if m, ok := datum.(map[interface{}]interface{}); ok {
			new_m := make(map[string]interface{})
			for k, v := range m {
				new_m[toString(k)] = v
			}
			datums[idx] = new_m
		}
	}
	return json.Marshal(datums)
}
