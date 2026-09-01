package binding

import (
	"errors"
	"io"
	"net/http"

	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

func (yamlBinding) Name() string { return "yaml" }

func (yamlBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	return decodeYAML(req.Body, obj)
}

func decodeYAML(r io.Reader, obj any) error {
	if err := yaml.NewDecoder(r).Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}

func (tomlBinding) Name() string { return "toml" }

func (tomlBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	return decodeTOML(req.Body, obj)
}

func decodeTOML(r io.Reader, obj any) error {
	if err := toml.NewDecoder(r).Decode(obj); err != nil {
		return err
	}
	return validate(obj)
}
