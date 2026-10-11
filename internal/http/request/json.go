package request

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

const MaxBodyBytes = 1 << 20

func Decode(r *http.Request, dst any) error {
	defer r.Body.Close()
	body, err := io.ReadAll(io.LimitReader(r.Body, MaxBodyBytes+1))
	if err != nil {
		return errors.New("could not read request")
	}
	if len(body) > MaxBodyBytes {
		return errors.New("request body too large")
	}
	body = bytes.TrimSpace(body)
	if len(body) == 0 || body[0] != '{' {
		return errors.New("request must contain a JSON object")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return errors.New("invalid JSON request")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return errors.New("request must contain one JSON object")
	}
	return nil
}
