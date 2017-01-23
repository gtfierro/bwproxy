package main

import (
	"encoding/json"
	"fmt"

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
