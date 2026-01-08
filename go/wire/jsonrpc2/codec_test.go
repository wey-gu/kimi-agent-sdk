package jsonrpc2

import (
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/rpc"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const testServiceName = "Wire"

func testClientRenamer(serviceMethod string) string {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot >= 0 {
		return strings.ToLower(serviceMethod[dot+1:])
	}
	return strings.ToLower(serviceMethod)
}

func testServerRenamer(method string) string {
	return testServiceName + "." + cases.Title(language.English).String(method)
}

func newTestCodec(rwc io.ReadWriteCloser) *Codec {
	seq := atomic.Uint64{}
	return NewCodec(
		rwc,
		ClientMethodRenamer(RenamerFunc(testClientRenamer)),
		ServerMethodRenamer(RenamerFunc(testServerRenamer)),
		JSONIDGenerator(GeneratorFunc[string](func() string {
			return strconv.FormatUint(seq.Add(1), 10)
		})),
	)
}

type TestArgs struct {
	UserInput string
}

type TestReply struct {
	Echo string
}

type testJSONErr struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e testJSONErr) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

type TestWireService struct{}

func (TestWireService) Prompt(args *TestArgs, reply *TestReply) error {
	reply.Echo = args.UserInput
	return nil
}

func (TestWireService) Failplain(_ *struct{}, _ *struct{}) error {
	return errors.New("bad")
}

func (TestWireService) Failjson(_ *struct{}, _ *struct{}) error {
	return testJSONErr{Code: 123, Message: "bad"}
}

func startRPCServer(t *testing.T, codec *Codec, rcvr any) <-chan struct{} {
	t.Helper()

	srv := rpc.NewServer()
	if err := srv.RegisterName("Wire", rcvr); err != nil {
		t.Fatalf("RegisterName: %v", err)
	}

	done := make(chan struct{})
	go func() {
		srv.ServeCodec(codec)
		close(done)
	}()
	return done
}

func newRPCClient(t *testing.T, rcvr any) *rpc.Client {
	t.Helper()

	c1, c2 := net.Pipe()
	clientCodec := newTestCodec(c1)
	serverCodec := newTestCodec(c2)
	done := startRPCServer(t, serverCodec, rcvr)

	client := rpc.NewClientWithCodec(clientCodec)
	t.Cleanup(func() {
		_ = client.Close()
		select {
		case <-done:
		case <-time.After(1 * time.Second):
			t.Fatalf("rpc server did not exit")
		}
	})
	return client
}

func waitUntil(t *testing.T, timeout time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(1 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for condition")
}

type failWriter struct {
	err error
}

func (w failWriter) Write(p []byte) (int, error) {
	return 0, w.err
}

type pipeRWC struct {
	r *io.PipeReader
	w io.Writer
}

func (p *pipeRWC) Read(b []byte) (int, error) {
	return p.r.Read(b)
}

func (p *pipeRWC) Write(b []byte) (int, error) {
	return p.w.Write(b)
}

func (p *pipeRWC) Close() error {
	_ = p.r.Close()
	if c, ok := p.w.(io.Closer); ok {
		_ = c.Close()
	}
	return nil
}

func TestCodec_RPC_RoundTrip_Success(t *testing.T) {
	client := newRPCClient(t, TestWireService{})

	var reply TestReply
	if err := client.Call("Wire.Prompt", &TestArgs{UserInput: "hello"}, &reply); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if reply.Echo != "hello" {
		t.Fatalf("unexpected reply: %+v", reply)
	}
}

func TestCodec_RPC_Error_PlainStringIsJSONEncodedString(t *testing.T) {
	client := newRPCClient(t, TestWireService{})

	err := client.Call("Wire.Failplain", &struct{}{}, &struct{}{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if got, want := err.Error(), "\"bad\""; got != want {
		t.Fatalf("unexpected error string: got %q want %q", got, want)
	}
}

func TestCodec_RPC_Error_JSONObject_PreservedAndParseable(t *testing.T) {
	client := newRPCClient(t, TestWireService{})

	err := client.Call("Wire.Failjson", &struct{}{}, &struct{}{})
	if err == nil {
		t.Fatalf("expected error")
	}

	parsed, ok := ParseServerError[testJSONErr](err)
	if !ok {
		t.Fatalf("expected ParseServerError ok=true, err=%v", err)
	}
	if parsed.Code != 123 || parsed.Message != "bad" {
		t.Fatalf("unexpected parsed error: %+v", parsed)
	}
}

func TestCodec_Notification_NoID_NoResponse(t *testing.T) {
	serverConn, peerConn := net.Pipe()
	serverCodec := newTestCodec(serverConn)
	done := startRPCServer(t, serverCodec, TestWireService{})

	enc := json.NewEncoder(peerConn)
	params, err := json.Marshal(&TestArgs{UserInput: "hello"})
	if err != nil {
		t.Fatalf("Marshal params: %v", err)
	}

	req := map[string]any{
		"jsonrpc": JSONRPC2Version,
		"method":  "prompt",
		"params":  json.RawMessage(params),
	}
	if err := enc.Encode(req); err != nil {
		t.Fatalf("Encode request: %v", err)
	}

	_ = peerConn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	dec := json.NewDecoder(peerConn)
	var resp Payload
	derr := dec.Decode(&resp)
	if derr == nil {
		t.Fatalf("expected no response, got: %+v", resp)
	}
	if ne, ok := derr.(net.Error); !ok || !ne.Timeout() {
		t.Fatalf("expected timeout, got: %T %v", derr, derr)
	}

	_ = peerConn.Close()
	select {
	case <-done:
	case <-time.After(1 * time.Second):
		t.Fatalf("rpc server did not exit")
	}
}

func TestCodec_EOF_AfterClose_ReturnsBareEOF(t *testing.T) {
	c1, c2 := net.Pipe()
	codec := newTestCodec(c1)
	_ = c2.Close()
	_ = codec.Close()

	if err := codec.ReadRequestHeader(&rpc.Request{}); err != io.EOF {
		t.Fatalf("ReadRequestHeader: got %T %v want io.EOF", err, err)
	}
	if err := codec.ReadRequestBody(&struct{}{}); err != io.EOF {
		t.Fatalf("ReadRequestBody: got %T %v want io.EOF", err, err)
	}
	if err := codec.WriteRequest(&rpc.Request{ServiceMethod: "Wire.Prompt", Seq: 1}, &TestArgs{UserInput: "x"}); err != io.EOF {
		t.Fatalf("WriteRequest: got %T %v want io.EOF", err, err)
	}
	if err := codec.ReadResponseHeader(&rpc.Response{}); err != io.EOF {
		t.Fatalf("ReadResponseHeader: got %T %v want io.EOF", err, err)
	}
	if err := codec.ReadResponseBody(&struct{}{}); err != io.EOF {
		t.Fatalf("ReadResponseBody: got %T %v want io.EOF", err, err)
	}

	codec.srvlock.Lock()
	codec.srvreqids[2] = "2"
	codec.srvlock.Unlock()
	if err := codec.WriteResponse(&rpc.Response{Seq: 2}, &struct{}{}); err != io.EOF {
		t.Fatalf("WriteResponse: got %T %v want io.EOF", err, err)
	}
}

func TestCodec_EOF_RemoteClose_ReadRequestHeaderReturnsEOF(t *testing.T) {
	c1, c2 := net.Pipe()
	codec := newTestCodec(c1)
	_ = c2.Close()
	defer codec.Close()

	ch := make(chan error, 1)
	go func() {
		ch <- codec.ReadRequestHeader(&rpc.Request{})
	}()

	select {
	case err := <-ch:
		if err != io.EOF {
			t.Fatalf("got %T %v want io.EOF", err, err)
		}
	case <-time.After(1 * time.Second):
		t.Fatalf("ReadRequestHeader did not return")
	}
}

func TestCodec_ReadResponseHeader_KnownID_CleansMaps(t *testing.T) {
	c1, c2 := net.Pipe()
	codec := newTestCodec(c1)
	defer codec.Close()
	defer c2.Close()

	codec.clilock.Lock()
	codec.reqmeth["rid"] = "Wire.Prompt"
	codec.clireqids["rid"] = 42
	codec.clilock.Unlock()

	_, _ = io.WriteString(c2, "{\"jsonrpc\":\"2.0\",\"id\":\"rid\",\"result\":{}}\n")

	var r rpc.Response
	if err := codec.ReadResponseHeader(&r); err != nil {
		t.Fatalf("ReadResponseHeader: %v", err)
	}
	if r.ServiceMethod != "Wire.Prompt" {
		t.Fatalf("unexpected ServiceMethod: %q", r.ServiceMethod)
	}
	if r.Seq != 42 {
		t.Fatalf("unexpected Seq: %d", r.Seq)
	}

	codec.clilock.Lock()
	_, okMeth := codec.reqmeth["rid"]
	_, okSeq := codec.clireqids["rid"]
	codec.clilock.Unlock()
	if okMeth || okSeq {
		t.Fatalf("expected reqmeth/clireqids entries to be deleted")
	}
}

func TestCodec_DecodeError_InvalidJSON_PropagatesNonEOF(t *testing.T) {
	c1, c2 := net.Pipe()
	codec := newTestCodec(c1)
	defer codec.Close()
	defer c2.Close()

	_, _ = io.WriteString(c2, "}\n")

	waitUntil(t, 1*time.Second, func() bool {
		return codec.err.Load() != nil
	})

	err := codec.ReadRequestHeader(&rpc.Request{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if err == io.EOF {
		t.Fatalf("expected non-EOF error, got io.EOF")
	}

	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected json syntax error, got %T %v", err, err)
	}
}

func TestCodec_RPC_UnknownMethod_DiscardBodyAndError(t *testing.T) {
	client := newRPCClient(t, TestWireService{})

	err := client.Call("Wire.Unknown", &TestArgs{UserInput: "x"}, &TestReply{})
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "can't find method") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseServerError_EdgeCases(t *testing.T) {
	if _, ok := ParseServerError[testJSONErr](nil); ok {
		t.Fatalf("expected ok=false for nil error")
	}
	if _, ok := ParseServerError[testJSONErr](errors.New("x")); ok {
		t.Fatalf("expected ok=false for non-ServerError")
	}
	if _, ok := ParseServerError[testJSONErr](rpc.ServerError("not json")); ok {
		t.Fatalf("expected ok=false for invalid JSON")
	}
}

func TestCodec_WriteRequest_MarshalError_ReturnsUnsupportedTypeError(t *testing.T) {
	c1, c2 := net.Pipe()
	codec := newTestCodec(c1)
	defer c2.Close()
	defer codec.Close()

	err := codec.WriteRequest(&rpc.Request{ServiceMethod: "Wire.Prompt", Seq: 1}, func() {})
	if err == nil {
		t.Fatalf("expected error")
	}
	var ute *json.UnsupportedTypeError
	if !errors.As(err, &ute) {
		t.Fatalf("expected *json.UnsupportedTypeError, got %T %v", err, err)
	}

	codec.clilock.Lock()
	pending := len(codec.reqmeth) + len(codec.clireqids)
	codec.clilock.Unlock()
	if pending != 0 {
		t.Fatalf("expected no pending mappings after marshal error")
	}
}

func TestCodec_WriteResponse_MarshalError_ReturnsUnsupportedTypeErrorAndCleansReqid(t *testing.T) {
	c1, c2 := net.Pipe()
	codec := newTestCodec(c1)
	defer c2.Close()
	defer codec.Close()

	codec.srvlock.Lock()
	codec.srvreqids[1] = "1"
	codec.srvlock.Unlock()

	err := codec.WriteResponse(&rpc.Response{Seq: 1}, func() {})
	if err == nil {
		t.Fatalf("expected error")
	}
	var ute *json.UnsupportedTypeError
	if !errors.As(err, &ute) {
		t.Fatalf("expected *json.UnsupportedTypeError, got %T %v", err, err)
	}

	codec.srvlock.Lock()
	_, ok := codec.srvreqids[1]
	codec.srvlock.Unlock()
	if ok {
		t.Fatalf("expected reqids entry to be deleted")
	}
}

func TestCodec_Send_EncodeError_SetsErrAndSubsequentCallsFail(t *testing.T) {
	writeErr := errors.New("write fail")
	pr, pw := io.Pipe()
	codec := newTestCodec(&pipeRWC{r: pr, w: failWriter{err: writeErr}})
	defer codec.Close()
	defer pw.Close()

	err := codec.WriteRequest(&rpc.Request{ServiceMethod: "Wire.Prompt", Seq: 1}, &TestArgs{UserInput: "x"})
	if err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}

	waitUntil(t, 1*time.Second, func() bool {
		return codec.err.Load() != nil
	})

	err = codec.WriteRequest(&rpc.Request{ServiceMethod: "Wire.Prompt", Seq: 2}, &TestArgs{UserInput: "y"})
	if err == nil {
		t.Fatalf("expected error")
	}
	if errors.Is(err, io.EOF) {
		t.Fatalf("expected non-EOF error, got %T %v", err, err)
	}
	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error, got %T %v", err, err)
	}

	codec.clilock.Lock()
	delete(codec.reqmeth, "1")
	delete(codec.clireqids, "1")
	codec.clilock.Unlock()
}
