package jsonrpc2

import (
	"encoding/json"
	"errors"
	"net/rpc"
)

func ParseError(err error) (Error, bool) {
	return ParseServerError[Error](err)
}

type ErrorCode int

const (
	ErrorCodeInvalidRequest ErrorCode = -32600
	ErrorCodeMethodNotFound ErrorCode = -32601
	ErrorCodeInvalidParams  ErrorCode = -32602
	ErrorCodeInternalError  ErrorCode = -32603
)

type Error struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
}

func (e Error) Error() string {
	// SAFETY: Error only contains int and string fields, which cannot fail to marshal.
	jsonerror, _ := json.Marshal(&e)
	return string(jsonerror)
}

func ParseServerError[E error](err error) (e E, ok bool) {
	if err == nil {
		return e, false
	}
	var srverr rpc.ServerError
	if errors.As(err, &srverr) {
		if err := json.Unmarshal([]byte(srverr), &e); err == nil {
			return e, true
		}
	}
	return e, false
}

type wraperror struct {
	error
}

func (e *wraperror) Unwrap() error {
	return e.error
}
