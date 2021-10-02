package filters

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Kaese72/sdup-lib/sduptemplates"
)

type Operator string

const (
	Equal        Operator = "eq"
	LessThan     Operator = "lt"
	LessEqual    Operator = "lte"
	GreaterThan  Operator = "gt"
	GreaterEqual Operator = "gte"
)

type AttributeFilterKey sduptemplates.AttributeKey

const ErrNotCompositeIdentifier = "could not split identifier"

func (afKey AttributeFilterKey) KeyValKeys() (attribute string, key string, err error) {
	s := strings.SplitN(string(afKey), ".", 2)
	if len(s) != 2 {
		err = errors.New(ErrNotCompositeIdentifier)

	} else {
		attribute = s[0]
		key = s[1]
	}
	return
}

type AttributeFilter struct {
	Operator Operator `json:"operator"`
	// The Value is either a string, float32 or boolean
	// eg. "some string"
	// eg. 69.420 //FIXME To what accuracy do we do numeric comparisons?
	// eg. false
	// eg.
	Value interface{} `json:"value"`
	// Key contains a string indicating what attribute we should filter on or a dot separated attribute.key identifer for keyval
	// eg. "active"
	// eg. "colorxy.x"
	Key AttributeFilterKey `json:"key"`
}

func (filter AttributeFilter) GetOperator() (op Operator, err error) {
	switch filter.Operator {
	case Equal:
		// Valid for all state types
		op = Equal
	case LessThan:
		// Valid for numeric state types
		op = LessThan
	case LessEqual:
		// Valid for numeric state types
		op = LessEqual
	case GreaterThan:
		// Valid for numeric state types
		op = GreaterThan
	case GreaterEqual:
		// Valid for numeric state types
		op = GreaterEqual
	default:
		err = fmt.Errorf("unknown operator: %s", filter.Operator)
	}
	return
}

type AttributeFilters []AttributeFilter

// id=[{"value": 123, "operator": "eq|lt|gt|gte", "key": "<attribute-key>.<sub-key>"}]
