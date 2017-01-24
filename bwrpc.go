package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	bw2 "gopkg.in/immesys/bw2bind.v5"
)

// BOSSWAVE RPC:
// JSON object will describe the action to be taken, and will deserialize to a struct
// that can then be passed to the actual BOSSWAVE api call, making sure to run the call
// with the correct client

type Procedure uint

const (
	UNKNOWN Procedure = iota
	SUBSCRIBE
	PUBLISH
	QUERY
)

func (p *Procedure) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	s = strings.ToUpper(s)
	log.Debug(s, "|", string(b))
	switch s {
	case "SUBSCRIBE":
		*p = SUBSCRIBE
	case "PUBLISH":
		*p = PUBLISH
	case "QUERY":
		*p = QUERY
	default:
		*p = UNKNOWN
	}
	return nil
}

type BWRPCCall struct {
	// api key of the client
	Key string `json:"key"`
	// name of the BOSSWAVE procedure to call
	Proc Procedure `json:"proc"`
	// parameters to that function
	Params map[string]interface{} `json:"params"`
}

// runs the RPC call and returns the json-serialized result and any error
func doRPCCall(ctx context.Context, client *bw2.BW2Client, perms Permissions, params BWRPCCall) ([]byte, error) {
	var result []byte
	select {
	case <-ctx.Done():
		return result, ctx.Err()
	default:
		switch params.Proc {
		case QUERY:
			if perms.Query.Allowed {
				return doQuery(ctx, client, perms, params)
			} else {
				return result, errors.Errorf("Key has no permission to Query")
			}
		default:
			return result, errors.Errorf("No method found matching %v", params.Proc)
		}
	}
}

func doQuery(ctx context.Context, client *bw2.BW2Client, perms Permissions, params BWRPCCall) ([]byte, error) {
	var results []interface{}

	// params needed:
	// - uri
	uri := getString("uri", params.Params)
	ponum := getString("ponum", params.Params)
	msgs, err := client.Query(&bw2.QueryParams{
		URI: uri,
	})
	if err != nil {
		return []byte{}, errors.Wrap(err, "Could not query")
	}

	for msg := range msgs {
		select {
		case <-ctx.Done():
			return []byte{}, ctx.Err()
		default:
		}
		for _, po := range msg.POs {
			if ponum != "" && !po.IsTypeDF(ponum) {
				continue
			}
			datum, err := po2iface(po)
			if err != nil {
				return []byte{}, errors.Wrap(err, "Could not retrieve iface from PO")
			}
			results = append(results, datum)
		}
	}

	return datums2json(results)
}
