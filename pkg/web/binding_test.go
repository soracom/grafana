package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type StructWithInt struct {
	A int `binding:"Required"`
}

type StructWithPrimitives struct {
	A int     `binding:"Required"`
	B string  `binding:"Required"`
	C bool    `binding:"Required"`
	D float64 `binding:"Required"`
}

type StructWithPrivateFields struct {
	A int `binding:"Required"` // must be validated
	b int `binding:"Required"` // will not be used
}

type StructWithInterface struct {
	A interface{} `binding:"Required"`
}
type StructWithSliceInts struct {
	A []int `binding:"Required"`
}
type StructWithSliceStructs struct {
	A []StructWithInt `binding:"Required"`
}
type StructWithSliceInterfaces struct {
	A []interface{} `binding:"Required"`
}
type StructWithStruct struct {
	A StructWithInt `binding:"Required"`
}
type StructWithStructPointer struct {
	A *StructWithInt `binding:"Required"`
}
type StructWithValidation struct {
	A int
}

func (sv StructWithValidation) Validate() error {
	if sv.A < 10 {
		return errors.New("too small")
	}
	return nil
}

type StructWithPointerValidation struct {
	A int
}

func (sv *StructWithPointerValidation) Validate() error {
	if sv.A < 10 {
		return errors.New("too small")
	}
	return nil
}

type bindTarget struct {
	Message string `json:"message"`
}

// endlessJSONReader supplies a JSON string longer than the body limit without
// allocating the full payload in the test.
type endlessJSONReader struct {
	prefix []byte
	read   int64
}

func (r *endlessJSONReader) Read(p []byte) (int, error) {
	if len(r.prefix) > 0 {
		n := copy(p, r.prefix)
		r.prefix = r.prefix[n:]
		r.read += int64(n)
		return n, nil
	}
	for i := range p {
		p[i] = 'A'
	}
	r.read += int64(len(p))
	return len(p), nil
}

func TestBindRejectsOversizedBody(t *testing.T) {
	const testLimit = 1024
	body := &endlessJSONReader{prefix: []byte(`{"message":"`)}
	req := httptest.NewRequest(http.MethodPost, "/", body)
	req.Header.Set("Content-Type", "application/json")

	var target bindTarget
	err := bindWithLimit(req, &target, testLimit)
	var maxBytesErr *http.MaxBytesError
	if !errors.As(err, &maxBytesErr) {
		t.Fatalf("expected a body size error, got %v", err)
	}
	if body.read > testLimit+1 {
		t.Fatalf("Bind read %d bytes, want at most %d", body.read, testLimit+1)
	}
}

func TestBindAcceptsBodyWithinLimit(t *testing.T) {
	message := strings.Repeat("A", 1024)
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"message":"`+message+`"}`))
	req.Header.Set("Content-Type", "application/json")

	var target bindTarget
	if err := Bind(req, &target); err != nil {
		t.Fatalf("Bind failed on small body: %v", err)
	}
	if target.Message != message {
		t.Fatal("Bind did not decode the request body")
	}
}

func TestValidationSuccess(t *testing.T) {
	var nilInterface *StructWithPointerValidation

	for _, x := range []interface{}{
		nil,
		42,
		"foo",
		struct{ A int }{},
		StructWithInt{42},
		StructWithPrimitives{42, "foo", true, 12.34},
		StructWithPrivateFields{12, 0},
		StructWithInterface{"foo"},
		StructWithSliceInts{[]int{1, 2, 3}},
		StructWithSliceInterfaces{[]interface{}{1, 2, 3}},
		StructWithSliceStructs{[]StructWithInt{{1}, {2}}},
		StructWithStruct{StructWithInt{3}},
		StructWithStructPointer{&StructWithInt{3}},
		StructWithValidation{42},
		&StructWithPointerValidation{42},
		nilInterface,
	} {
		if err := validate(x); err != nil {
			t.Error("Validation failed:", x, err)
		}
	}
}
func TestValidationFailure(t *testing.T) {
	for i, x := range []interface{}{
		StructWithInt{0},
		StructWithPrimitives{0, "foo", true, 12.34},
		StructWithPrimitives{42, "", true, 12.34},
		StructWithPrimitives{42, "foo", false, 12.34},
		StructWithPrimitives{42, "foo", true, 0},
		StructWithPrivateFields{0, 1},
		StructWithInterface{},
		StructWithInterface{nil},
		StructWithSliceInts{},
		StructWithSliceInts{[]int{}},
		StructWithSliceStructs{[]StructWithInt{}},
		StructWithSliceStructs{[]StructWithInt{{0}, {2}}},
		StructWithSliceStructs{[]StructWithInt{{2}, {0}}},
		StructWithSliceInterfaces{[]interface{}{}},
		StructWithSliceInterfaces{nil},
		StructWithStruct{StructWithInt{}},
		StructWithStruct{StructWithInt{0}},
		StructWithStructPointer{},
		StructWithStructPointer{&StructWithInt{}},
		StructWithValidation{2},
		&StructWithPointerValidation{2},
	} {
		if err := validate(x); err == nil {
			t.Error("Validation should fail:", i, x)
		}
	}
}
