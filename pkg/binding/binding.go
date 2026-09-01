// Package binding implements request binding with the exact semantics the
// project relied on before the move to Echo: the content-type dispatch table,
// the raw encoding/json decoder errors and the `binding` struct tag used by
// go-playground/validator.
//
// Echo's DefaultBinder cannot be used directly because it rejects unknown
// content types with 415 instead of falling back to form decoding, it rewrites
// JSON decoder errors, and it performs no validation. Controllers put the
// binding error text straight into the response body, so those differences are
// observable by clients.
package binding

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

// Content-Type values used by the dispatch table.
const (
	MIMEJSON              = "application/json"
	MIMEHTML              = "text/html"
	MIMEXML               = "application/xml"
	MIMEXML2              = "text/xml"
	MIMEPlain             = "text/plain"
	MIMEPOSTForm          = "application/x-www-form-urlencoded"
	MIMEMultipartPOSTForm = "multipart/form-data"
	MIMEPROTOBUF          = "application/x-protobuf"
	MIMEYAML              = "application/x-yaml"
	MIMEYAML2             = "application/yaml"
	MIMETOML              = "application/toml"
)

const defaultMemory = 32 << 20

// Binding describes a request decoder.
type Binding interface {
	Name() string
	Bind(*http.Request, any) error
}

var (
	JSON          Binding = jsonBinding{}
	XML           Binding = xmlBinding{}
	Form          Binding = formBinding{}
	FormPost      Binding = formPostBinding{}
	FormMultipart Binding = formMultipartBinding{}
	Query         Binding = queryBinding{}
	ProtoBuf      Binding = protobufBinding{}
	YAML          Binding = yamlBinding{}
	TOML          Binding = tomlBinding{}
)

// Default returns the decoder for a request method and content type.
func Default(method, contentType string) Binding {
	if method == http.MethodGet {
		return Form
	}

	switch contentType {
	case MIMEJSON:
		return JSON
	case MIMEXML, MIMEXML2:
		return XML
	case MIMEPROTOBUF:
		return ProtoBuf
	case MIMEYAML, MIMEYAML2:
		return YAML
	case MIMETOML:
		return TOML
	case MIMEMultipartPOSTForm:
		return FormMultipart
	default: // case MIMEPOSTForm:
		return Form
	}
}

// FilterFlags trims the parameters off a Content-Type header value, so that
// "application/json; charset=utf-8" still selects the JSON decoder.
func FilterFlags(content string) string {
	for i, char := range content {
		if char == ' ' || char == ';' {
			return content[:i]
		}
	}
	return content
}

// StructValidator validates a decoded payload.
type StructValidator interface {
	ValidateStruct(any) error
	Engine() any
}

// Validator is applied by every decoder after a successful decode.
var Validator StructValidator = &defaultValidator{}

func validate(obj any) error {
	if Validator == nil {
		return nil
	}
	return Validator.ValidateStruct(obj)
}

type defaultValidator struct {
	once     sync.Once
	validate *validator.Validate
}

// SliceValidationError collects the errors of every element of a slice payload.
type SliceValidationError []error

func (err SliceValidationError) Error() string {
	if len(err) == 0 {
		return ""
	}

	var b strings.Builder
	for i := range err {
		if err[i] != nil {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString("[" + strconv.Itoa(i) + "]: " + err[i].Error())
		}
	}
	return b.String()
}

func (v *defaultValidator) ValidateStruct(obj any) error {
	if obj == nil {
		return nil
	}

	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		if value.Elem().Kind() != reflect.Struct {
			return v.ValidateStruct(value.Elem().Interface())
		}
		return v.validateStruct(obj)
	case reflect.Struct:
		return v.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(SliceValidationError, 0)
		for i := range count {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

func (v *defaultValidator) validateStruct(obj any) error {
	v.lazyinit()
	return v.validate.Struct(obj)
}

func (v *defaultValidator) Engine() any {
	v.lazyinit()
	return v.validate
}

func (v *defaultValidator) lazyinit() {
	v.once.Do(func() {
		v.validate = validator.New()
		v.validate.SetTagName("binding")
	})
}

type (
	jsonBinding          struct{}
	xmlBinding           struct{}
	formBinding          struct{}
	formPostBinding      struct{}
	formMultipartBinding struct{}
	queryBinding         struct{}
	protobufBinding      struct{}
	yamlBinding          struct{}
	tomlBinding          struct{}
)

func (jsonBinding) Name() string { return "json" }

func (jsonBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	return decodeJSON(req.Body, obj)
}

func decodeJSON(r io.Reader, obj any) error {
	if err := json.NewDecoder(r).Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

func (xmlBinding) Name() string { return "xml" }

func (xmlBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	if err := xml.NewDecoder(req.Body).Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

func (formBinding) Name() string { return "form" }

func (formBinding) Bind(req *http.Request, obj any) error {
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := req.ParseMultipartForm(defaultMemory); err != nil && !errors.Is(err, http.ErrNotMultipart) {
		return err
	}
	if err := mapForm(obj, req.Form); err != nil {
		return err
	}
	return validate(obj)
}

func (formPostBinding) Name() string { return "form-urlencoded" }

func (formPostBinding) Bind(req *http.Request, obj any) error {
	if err := req.ParseForm(); err != nil {
		return err
	}
	if err := mapForm(obj, req.PostForm); err != nil {
		return err
	}
	return validate(obj)
}

func (formMultipartBinding) Name() string { return "multipart/form-data" }

func (formMultipartBinding) Bind(req *http.Request, obj any) error {
	if err := req.ParseMultipartForm(defaultMemory); err != nil {
		return err
	}
	if err := mappingByPtr(obj, (*multipartRequest)(req), "form"); err != nil {
		return err
	}
	return validate(obj)
}

func (queryBinding) Name() string { return "query" }

func (queryBinding) Bind(req *http.Request, obj any) error {
	if err := mapForm(obj, req.URL.Query()); err != nil {
		return err
	}
	return validate(obj)
}

func (protobufBinding) Name() string { return "protobuf" }

// Bind drains the body and reports that the payload is not a protobuf message.
// None of the request payloads in this project is a generated protobuf message,
// so this is the only branch the previous implementation could ever reach.
func (protobufBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	if _, err := io.ReadAll(req.Body); err != nil {
		return err
	}
	return errors.New("obj is not ProtoMessage")
}
