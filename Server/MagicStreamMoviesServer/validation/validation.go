// Package validation provides the validator shared by the whole application and registers the project's custom rules.
package validation

import (
	"errors"
	"reflect"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
)

var (
	imdbIDPattern    = regexp.MustCompile(`^tt\d{7,10}$`)
	youTubeIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{11}$`)
)

// New returns a validator with the custom rules registered. *validator.Validate is safe for concurrent use; create it once at startup and reuse it.
func New() *validator.Validate {
	v := validator.New(validator.WithRequiredStructEnabled())
	// Use JSON field names (first_name) in error messages instead of Go field names (FirstName).
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		name, _, _ := strings.Cut(f.Tag.Get("json"), ",")
		if name == "" || name == "-" {
			return f.Name
		}
		return name
	})
	mustRegister(v, "imdb_id", imdbIDPattern)
	mustRegister(v, "youtube_id", youTubeIDPattern)
	return v
}

func mustRegister(v *validator.Validate, tag string, pattern *regexp.Regexp) {
	err := v.RegisterValidation(tag, func(fl validator.FieldLevel) bool {
		return pattern.MatchString(fl.Field().String())
	})
	if err != nil {
		panic(err)
	}
}

// Describe turns validation errors into client-friendly details (field and rule names only, no internal structure).
func Describe(err error) []string {
	var verrs validator.ValidationErrors
	if !errors.As(err, &verrs) {
		return nil
	}
	details := make([]string, 0, len(verrs))
	for _, fe := range verrs {
		rule := fe.Tag()
		if fe.Param() != "" {
			rule += "=" + fe.Param()
		}
		details = append(details, fe.Field()+": "+rule)
	}
	return details
}
