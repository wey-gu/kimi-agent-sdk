package kimi

import (
	"encoding/json"
	"reflect"
	"testing"
)

// StringResult implements fmt.Stringer for test return values
type StringResult string

func (s StringResult) String() string {
	return string(s)
}

// JSONResult implements fmt.Stringer by marshaling to JSON
type JSONResult map[string]any

func (j JSONResult) String() string {
	data, _ := json.Marshal(j)
	return string(data)
}

type SearchParams struct {
	Query string `json:"query" description:"The search query"`
	Limit int    `json:"limit,omitempty" description:"Max results"`
}

func Search(params SearchParams) (JSONResult, error) {
	return JSONResult{"results": []string{params.Query}}, nil
}

func TestCreateTool_Basic(t *testing.T) {
	tool, err := CreateTool(Search)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	if tool.def.Type != "function" {
		t.Errorf("expected type=function, got %s", tool.def.Type)
	}
	if !tool.def.Function.Valid {
		t.Fatal("expected Function.Valid=true")
	}
	// Function name includes package path with '.' replaced by '_'
	if tool.def.Function.Value.Name == "" {
		t.Error("expected non-empty name")
	}
}

func TestCreateTool_WithOptions(t *testing.T) {
	tool, err := CreateTool(Search,
		WithName("custom_search"),
		WithDescription("A custom search tool"),
		WithFieldDescription("Query", "Custom query description"),
	)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	if tool.def.Function.Value.Name != "custom_search" {
		t.Errorf("expected name=custom_search, got %s", tool.def.Function.Value.Name)
	}
	if tool.def.Function.Value.Description != "A custom search tool" {
		t.Errorf("expected description='A custom search tool', got %s", tool.def.Function.Value.Description)
	}
}

func TestCreateTool_Schema(t *testing.T) {
	tool, err := CreateTool(Search)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(tool.def.Function.Value.Parameters, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	if schema["type"] != "object" {
		t.Errorf("expected type=object, got %v", schema["type"])
	}

	props, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be a map")
	}

	queryProp, ok := props["query"].(map[string]any)
	if !ok {
		t.Fatal("expected query property")
	}
	if queryProp["type"] != "string" {
		t.Errorf("expected query.type=string, got %v", queryProp["type"])
	}
	if queryProp["description"] != "The search query" {
		t.Errorf("expected query.description='The search query', got %v", queryProp["description"])
	}

	limitProp, ok := props["limit"].(map[string]any)
	if !ok {
		t.Fatal("expected limit property")
	}
	if limitProp["type"] != "integer" {
		t.Errorf("expected limit.type=integer, got %v", limitProp["type"])
	}

	required, ok := schema["required"].([]any)
	if !ok {
		t.Fatal("expected required to be an array")
	}
	// Only query should be required (limit has omitempty)
	if len(required) != 1 || required[0] != "query" {
		t.Errorf("expected required=[query], got %v", required)
	}
}

func TestCreateTool_Call(t *testing.T) {
	tool, err := CreateTool(Search)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	args := json.RawMessage(`{"query":"test","limit":10}`)
	result, err := tool.call(args)
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	var res map[string]any
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	results, ok := res["results"].([]any)
	if !ok {
		t.Fatal("expected results to be an array")
	}
	if len(results) != 1 || results[0] != "test" {
		t.Errorf("expected results=[test], got %v", results)
	}
}

type NestedParams struct {
	User    UserInfo `json:"user"`
	Tags    []string `json:"tags,omitempty"`
	Options *Options `json:"options,omitempty"`
}

type UserInfo struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Options struct {
	Debug bool `json:"debug"`
}

func ProcessNested(params NestedParams) (StringResult, error) {
	return StringResult(params.User.Name), nil
}

func TestCreateTool_NestedStruct(t *testing.T) {
	tool, err := CreateTool(ProcessNested)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(tool.def.Function.Value.Parameters, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	// Check nested user struct
	userProp := props["user"].(map[string]any)
	if userProp["type"] != "object" {
		t.Errorf("expected user.type=object, got %v", userProp["type"])
	}
	userProps := userProp["properties"].(map[string]any)
	if _, ok := userProps["name"]; !ok {
		t.Error("expected user to have name property")
	}
	if _, ok := userProps["age"]; !ok {
		t.Error("expected user to have age property")
	}

	// Check array type
	tagsProp := props["tags"].(map[string]any)
	if tagsProp["type"] != "array" {
		t.Errorf("expected tags.type=array, got %v", tagsProp["type"])
	}
	tagsItems := tagsProp["items"].(map[string]any)
	if tagsItems["type"] != "string" {
		t.Errorf("expected tags.items.type=string, got %v", tagsItems["type"])
	}

	// Check pointer type (should be object)
	optionsProp := props["options"].(map[string]any)
	if optionsProp["type"] != "object" {
		t.Errorf("expected options.type=object, got %v", optionsProp["type"])
	}

	// Check required - user should be required, tags and options should not
	required := schema["required"].([]any)
	if len(required) != 1 || required[0] != "user" {
		t.Errorf("expected required=[user], got %v", required)
	}
}

func TestCreateTool_WithFieldDescriptionOverride(t *testing.T) {
	tool, err := CreateTool(Search,
		WithFieldDescription("Query", "Overridden description"),
	)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(tool.def.Function.Value.Parameters, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)
	queryProp := props["query"].(map[string]any)

	// WithFieldDescription should override struct tag
	if queryProp["description"] != "Overridden description" {
		t.Errorf("expected description='Overridden description', got %v", queryProp["description"])
	}
}

type AllTypesParams struct {
	BoolField    bool    `json:"bool_field"`
	IntField     int     `json:"int_field"`
	Int64Field   int64   `json:"int64_field"`
	Float32Field float32 `json:"float32_field"`
	Float64Field float64 `json:"float64_field"`
	StringField  string  `json:"string_field"`
}

func ProcessAllTypes(params AllTypesParams) (StringResult, error) {
	return StringResult("ok"), nil
}

func TestCreateTool_AllTypes(t *testing.T) {
	tool, err := CreateTool(ProcessAllTypes)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	var schema map[string]any
	if err := json.Unmarshal(tool.def.Function.Value.Parameters, &schema); err != nil {
		t.Fatalf("failed to unmarshal schema: %v", err)
	}

	props := schema["properties"].(map[string]any)

	tests := []struct {
		field    string
		expected string
	}{
		{"bool_field", "boolean"},
		{"int_field", "integer"},
		{"int64_field", "integer"},
		{"float32_field", "number"},
		{"float64_field", "number"},
		{"string_field", "string"},
	}

	for _, tt := range tests {
		prop := props[tt.field].(map[string]any)
		if prop["type"] != tt.expected {
			t.Errorf("expected %s.type=%s, got %v", tt.field, tt.expected, prop["type"])
		}
	}
}

func TestWithName(t *testing.T) {
	opt := &toolOption{}
	WithName("test_name")(opt)

	if opt.name != "test_name" {
		t.Errorf("expected name=test_name, got %s", opt.name)
	}
}

func TestWithDescription(t *testing.T) {
	opt := &toolOption{}
	WithDescription("test description")(opt)

	if opt.description != "test description" {
		t.Errorf("expected description='test description', got %s", opt.description)
	}
}

func TestWithFieldDescription(t *testing.T) {
	opt := &toolOption{}
	WithFieldDescription("Field1", "desc1")(opt)
	WithFieldDescription("Field2", "desc2")(opt)

	expected := map[string]string{
		"Field1": "desc1",
		"Field2": "desc2",
	}
	if !reflect.DeepEqual(opt.fieldDescriptions, expected) {
		t.Errorf("expected fieldDescriptions=%v, got %v", expected, opt.fieldDescriptions)
	}
}

type UnsupportedParams struct {
	Callback func() `json:"callback"`
}

func ProcessUnsupported(params UnsupportedParams) (StringResult, error) {
	return "", nil
}

func TestCreateTool_UnsupportedType(t *testing.T) {
	_, err := CreateTool(ProcessUnsupported)
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

type InterfaceParams struct {
	Data any `json:"data"`
}

func ProcessInterface(params InterfaceParams) (StringResult, error) {
	return "", nil
}

func TestCreateTool_InterfaceType(t *testing.T) {
	_, err := CreateTool(ProcessInterface)
	if err == nil {
		t.Error("expected error for interface type, got nil")
	}
}

func ProcessString(params string) (StringResult, error) {
	return StringResult(params), nil
}

func TestCreateTool_NonStructParam(t *testing.T) {
	_, err := CreateTool(ProcessString)
	if err == nil {
		t.Error("expected error for non-struct parameter, got nil")
	}
}

// Test stringifyResult with different return types

type SimpleArgs struct {
	Input string `json:"input"`
}

// Test 1: string return type
func ReturnString(args SimpleArgs) (string, error) {
	return "direct string: " + args.Input, nil
}

func TestCreateTool_ReturnString(t *testing.T) {
	tool, err := CreateTool(ReturnString)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	result, err := tool.call(json.RawMessage(`{"input":"test"}`))
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	expected := "direct string: test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 2: fmt.Stringer return type (already covered by existing tests, but explicit)
func ReturnStringer(args SimpleArgs) (StringResult, error) {
	return StringResult("stringer: " + args.Input), nil
}

func TestCreateTool_ReturnStringer(t *testing.T) {
	tool, err := CreateTool(ReturnStringer)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	result, err := tool.call(json.RawMessage(`{"input":"test"}`))
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	expected := "stringer: test"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}
}

// Test 3: struct return type (JSON serialized)
type StructResult struct {
	Output string `json:"output"`
	Count  int    `json:"count"`
}

func ReturnStruct(args SimpleArgs) (StructResult, error) {
	return StructResult{Output: args.Input, Count: len(args.Input)}, nil
}

func TestCreateTool_ReturnStruct(t *testing.T) {
	tool, err := CreateTool(ReturnStruct)
	if err != nil {
		t.Fatalf("CreateTool failed: %v", err)
	}

	result, err := tool.call(json.RawMessage(`{"input":"hello"}`))
	if err != nil {
		t.Fatalf("call failed: %v", err)
	}

	var res StructResult
	if err := json.Unmarshal([]byte(result), &res); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}

	if res.Output != "hello" {
		t.Errorf("expected output=hello, got %s", res.Output)
	}
	if res.Count != 5 {
		t.Errorf("expected count=5, got %d", res.Count)
	}
}
