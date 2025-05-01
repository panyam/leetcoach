package dsl

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/panyam/leetcoach/sdl/core"
)

var (
	ErrInvalidMemberAccess = errors.New("invalid member access")
	ErrUnsupportedMember   = errors.New("unsupported member accessed")
)

// evalMemberAccessExpr handles receiver.Member access.
// Assumes Member refers to a parameter/field of the receiver object.
// Retrieves the deterministic value of the field, wraps it in a new
// zero-latency VarState, and returns that state.
func (v *VM) evalMemberAccessExpr(expr *MemberAccessExpr) (*VarState, error) {
	// 1. Evaluate the Receiver expression
	receiverState, err := v.Eval(expr.Receiver)
	if err != nil {
		var retVal *ReturnValue // Check for early return
		if errors.As(err, &retVal) {
			return retVal.Value, err
		}
		return nil, fmt.Errorf("error evaluating receiver for member access '%s': %w", expr.Member, err)
	}
	if receiverState == nil || receiverState.ValueOutcome == nil {
		return nil, fmt.Errorf("%w: receiver evaluated to nil state or nil value outcome for member '%s'", ErrInvalidMemberAccess, expr.Member)
	}

	// 2. Get the underlying Go object (receiver)
	//    It might be stored directly, or within a deterministic Outcome.
	var deterministicReceiverObj any
	receiverValueRaw := receiverState.ValueOutcome // This is the 'any' from VarState

	receiverValAsValue := reflect.ValueOf(receiverValueRaw) // Reflect value of the 'any'
	receiverTypAsType := reflect.TypeOf(receiverValueRaw)   // Reflect type of the 'any'

	isOutcomeType := false
	if receiverTypAsType != nil && receiverTypAsType.Kind() == reflect.Ptr {
		elemType := receiverTypAsType.Elem()
		// *** Use reflect.Type methods ***
		if elemType.Kind() == reflect.Struct &&
			elemType.Name() == "Outcomes" && // Check struct name via Type
			elemType.PkgPath() == "github.com/panyam/leetcoach/sdl/core" { // Check package path via Type
			isOutcomeType = true
		}
	}

	if isOutcomeType {
		// If it's an outcome, ensure it's deterministic
		getValueMethod := receiverValAsValue.MethodByName("GetValue") // Check reflect.Value for method
		if !getValueMethod.IsValid() {
			// This indicates an internal issue - we identified it as Outcome but has no GetValue?
			return nil, fmt.Errorf("internal error: receiver outcome type %T missing GetValue for field access '%s'", receiverValueRaw, expr.Member)
		}
		getValueResults := getValueMethod.Call(nil)
		if len(getValueResults) != 2 || !getValueResults[1].Bool() {
			return nil, fmt.Errorf("%w: receiver '%s' outcome must hold a deterministic value (single bucket) for field access '%s'", ErrInvalidMemberAccess, expr.Receiver, expr.Member)
		}
		deterministicReceiverObj = getValueResults[0].Interface() // Get the inner Go object
	} else {
		// If it's not an outcome object, assume it's the raw Go object already
		deterministicReceiverObj = receiverValueRaw
	}

	// 3. Access the field on the deterministic object using reflection
	objVal := reflect.ValueOf(deterministicReceiverObj)
	if !objVal.IsValid() { // Check if the deterministic object itself is nil
		return nil, fmt.Errorf("%w: receiver object resolved to nil before accessing field '%s'", ErrInvalidMemberAccess, expr.Member)
	}
	if objVal.Kind() == reflect.Ptr {
		if objVal.IsNil() {
			return nil, fmt.Errorf("%w: receiver object (%T) is nil pointer, cannot access field '%s'", ErrInvalidMemberAccess, deterministicReceiverObj, expr.Member)
		}
		objVal = objVal.Elem()
	}
	if objVal.Kind() != reflect.Struct {
		return nil, fmt.Errorf("%w: receiver object (%T) is not a struct, cannot access field '%s'", ErrInvalidMemberAccess, deterministicReceiverObj, expr.Member)
	}

	fieldVal := objVal.FieldByName(expr.Member)
	if !fieldVal.IsValid() {
		return nil, fmt.Errorf("%w: field '%s' not found in receiver type %T", ErrNotFound, expr.Member, deterministicReceiverObj)
	}
	if !fieldVal.CanInterface() {
		return nil, fmt.Errorf("%w: cannot access unexported field '%s' in receiver type %T", ErrInvalidMemberAccess, expr.Member, deterministicReceiverObj)
	}

	fieldValueInterface := fieldVal.Interface()

	// 4. Wrap the deterministic field value in a new VarState with zero latency
	var fieldValueOutcome any
	var wrapErr error

	// Handle common types expected for parameters/fields
	switch fv := fieldValueInterface.(type) {
	case bool:
		fieldValueOutcome = (&core.Outcomes[bool]{}).Add(1.0, fv)
	case int:
		fieldValueOutcome = (&core.Outcomes[int64]{}).Add(1.0, int64(fv))
	case int64:
		fieldValueOutcome = (&core.Outcomes[int64]{}).Add(1.0, fv)
	case uint:
		fieldValueOutcome = (&core.Outcomes[int64]{}).Add(1.0, int64(fv))
	case uint64:
		fieldValueOutcome = (&core.Outcomes[int64]{}).Add(1.0, int64(fv))
	case string:
		fieldValueOutcome = (&core.Outcomes[string]{}).Add(1.0, fv)
	// case float64: fieldValueOutcome = (&core.Outcomes[float64]{}).Add(1.0, fv)
	case core.Duration:
		fieldValueOutcome = (&core.Outcomes[core.Duration]{}).Add(1.0, fv)
	// Pass through if the field itself is an Outcome type
	case *core.Outcomes[bool]:
		fieldValueOutcome = fv
	case *core.Outcomes[int64]:
		fieldValueOutcome = fv
	// case *core.Outcomes[float64]:
	case *core.Outcomes[core.Duration]:
		fieldValueOutcome = fv
	case *core.Outcomes[string]:
		fieldValueOutcome = fv
	// Add other Outcome types if needed
	default:
		// Maybe the field holds another component instance?
		// Check if the type fv might be a known component type (this gets complex)
		// For now, assume fields hold primitives or Outcomes.
		wrapErr = fmt.Errorf("unsupported field type %T encountered for member '%s'", fv, expr.Member)
	}
	if wrapErr != nil {
		return nil, wrapErr
	}
	if fieldValueOutcome == nil {
		return nil, fmt.Errorf("internal error: failed to wrap field value for member '%s'", expr.Member)
	}

	// 5. Return the new VarState
	return &VarState{
		ValueOutcome:   fieldValueOutcome,
		LatencyOutcome: zeroLatencyOutcome(), // Accessing field has no latency
	}, nil
}
