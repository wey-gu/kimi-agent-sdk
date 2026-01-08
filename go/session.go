package kimi

import (
	"context"
	"errors"
	"io"
	"net/rpc"
	"os/exec"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/MoonshotAI/kimi-agent-sdk/go/wire"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire/jsonrpc2"
	"github.com/MoonshotAI/kimi-agent-sdk/go/wire/transport"
)

var (
	tpname = reflect.TypeOf((*transport.Transport)(nil)).Elem().Name()
	title  = cases.Title(language.English)
)

func NewSession(options ...Option) (*Session, error) {
	args := []string{"kimi"}
	for _, f := range options {
		args = append(args, f(args)...)
	}
	args = append(args, "--wire")
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	codec := jsonrpc2.NewCodec(&stdio{stdin, stdout},
		jsonrpc2.ClientMethodRenamer(jsonrpc2.RenamerFunc(func(method string) string {
			return strings.ToLower(strings.TrimPrefix(method, tpname+"."))
		})),
		jsonrpc2.ServerMethodRenamer(jsonrpc2.RenamerFunc(func(method string) string {
			return tpname + "." + title.String(method)
		})),
	)
	session := &Session{
		cmd:   cmd,
		codec: codec,
		msgs:  new(atomic.Pointer[chan wire.Message]),
		usrc:  new(atomic.Pointer[chan wire.RequestResponse]),
		tp:    transport.NewTransportClient(rpc.NewClientWithCodec(codec)),
	}
	go session.serve()
	watch := func() {
		cmd.Wait()
		stdin.Close()
		stdout.Close()
	}
	go watch()
	return session, nil
}

type Session struct {
	cmd   *exec.Cmd
	codec *jsonrpc2.Codec
	msgs  *atomic.Pointer[chan wire.Message]
	usrc  *atomic.Pointer[chan wire.RequestResponse]
	tp    transport.Transport
}

func (c *Session) serve() {
	responder := transport.NewTransportServer(&Responder{msgs: c.msgs, usrc: c.usrc})
	server := rpc.NewServer()
	server.RegisterName(tpname, responder)
	for {
		if err := server.ServeRequest(c.codec); err != nil {
			return
		}
	}
}

func wait(codec *jsonrpc2.Codec) {
	for {
		pending := codec.PendingRequests()
		if pending == 0 {
			return
		}
		time.Sleep(time.Duration(pending) * time.Second)
	}
}

func (c *Session) RoundTrip(ctx context.Context, content wire.Content) (*Turn, error) {
	var (
		bg     sync.WaitGroup
		msgs   = make(chan wire.Message)
		usrc   = make(chan wire.RequestResponse, 1)
		errc1  = make(chan error, 1)
		errc2  = make(chan error, 1)
		resc   = make(chan struct{}, 1)
		result = new(atomic.Pointer[wire.PromptResult])
	)
	defer close(errc1)
	defer close(errc2)
	defer close(resc)
	c.msgs.Store(&msgs)
	c.usrc.Store(&usrc)
	bg.Go(func() {
		msg0 := <-msgs
		if _, ok := msg0.(wire.TurnBegin); !ok {
			errc1 <- ErrTurnNotFound
			return
		}
		resc <- struct{}{}
	})
	bg.Go(func() {
		defer close(msgs)
		defer wait(c.codec)
		result.Store(&wire.PromptResult{Status: wire.PromptResultStatusPending})
		rpcresult, err := c.tp.Prompt(&wire.PromptParams{
			UserInput: content,
		})
		if err != nil {
			errc2 <- err
			return
		}
		result.Store(rpcresult)
	})
	exit := func(err error) error {
		for range msgs {
		}
		bg.Wait()
		if state := c.cmd.ProcessState; state.ExitCode() > 0 {
			return errors.New(state.String())
		}
		if err != nil {
			return err
		}
		return nil
	}
	select {
	case <-resc:
		return turnBegin(ctx, c.tp, result, msgs, usrc, exit), nil
	case err := <-errc1:
		return nil, exit(err)
	case err := <-errc2:
		return nil, exit(err)
	case <-ctx.Done():
		return nil, exit(ctx.Err())
	}
}

type Responder struct {
	transport.Transport
	msgs *atomic.Pointer[chan wire.Message]
	usrc *atomic.Pointer[chan wire.RequestResponse]
}

func (r *Responder) Event(event *wire.EventParams) (*wire.EventResult, error) {
	msgs := r.msgs.Load()
	if msgs == nil {
		return &wire.EventResult{}, nil
	}
	*msgs <- event.Payload
	return &wire.EventResult{}, nil
}

func (r *Responder) Request(request *wire.RequestParams) (*wire.RequestResult, error) {
	msgs := r.msgs.Load()
	if msgs == nil {
		return &wire.RequestResult{
			RequestID: request.Payload.RequestID(),
			Response:  wire.RequestResponseReject,
		}, nil
	}
	usrc := *r.usrc.Load()
	var wr wire.Request
	switch payload := request.Payload.(type) {
	case wire.ApprovalRequest:
		payload.Responder = ResponderFunc(func(rr wire.RequestResponse) error {
			usrc <- rr
			return nil
		})
		wr = payload
	}
	*msgs <- wr
	return &wire.RequestResult{
		RequestID: request.Payload.RequestID(),
		Response:  <-*r.usrc.Load(),
	}, nil
}

func (c *Session) Close() error {
	return errors.Join(
		c.codec.Close(),
		c.cmd.Cancel(),
	)
}

type stdio struct {
	io.WriteCloser
	io.ReadCloser
}

func (s *stdio) Close() error {
	return errors.Join(
		s.WriteCloser.Close(),
		s.ReadCloser.Close(),
	)
}

type ResponderFunc func(wire.RequestResponse) error

func (f ResponderFunc) Respond(r wire.RequestResponse) error {
	return f(r)
}
