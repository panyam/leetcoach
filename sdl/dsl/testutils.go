package dsl

import (
	"fmt"
	"strconv"

	"github.com/panyam/leetcoach/sdl/core"
)

type Disk struct {
	Name        string
	ProfileName string // e.g., "SSD", "HDD" - VM uses this to lookup profile
}

func NewDisk(name, profile string) *Disk {
	if profile == "" {
		profile = "SSD" // Default
	}
	return &Disk{Name: name, ProfileName: profile}
}

// Read returns an  expression representing a disk read operation.
// It instructs the VM to look up the read profile based on d.ProfileName.
func (d *Disk) Read() Expr {
	return &InternalCallExpr{
		FuncName: "GetDiskReadProfile",
		Args:     []Expr{&LiteralExpr{Kind: "STRING", Value: d.ProfileName}},
	}
}

// Write returns an  expression representing a disk write operation.
func (d *Disk) Write() Expr {
	return &InternalCallExpr{
		FuncName: "GetDiskWriteProfile",
		Args:     []Expr{&LiteralExpr{Kind: "STRING", Value: d.ProfileName}},
	}
}

// ReadProcessWrite returns an  expression for Read -> Process -> Write.
// processingTime is an Expr yielding Outcomes[Duration].
func (d *Disk) ReadProcessWrite(processingTime Expr) Expr {
	read := d.Read()
	write := d.Write()

	// Represents Read -> ProcessingTime -> Write sequentially
	step1 := &AndExpr{Left: read, Right: processingTime}
	step2 := &AndExpr{Left: step1, Right: write}
	return step2
}

func litStr(v string) Expr { return &LiteralExpr{Kind: "STRING", Value: v} }
func litInt(v int) Expr    { return &LiteralExpr{Kind: "INT", Value: strconv.Itoa(v)} }
func litFloat(v float64) Expr {
	return &LiteralExpr{Kind: "FLOAT", Value: strconv.FormatFloat(v, 'f', -1, 64)}
}
func litBool(v bool) Expr            { return &LiteralExpr{Kind: "BOOL", Value: strconv.FormatBool(v)} }
func ident(n string) Expr            { return &IdentifierExpr{Name: n} }
func member(r Expr, m string) Expr   { return &MemberAccessExpr{Receiver: r, Member: m} }
func call(f Expr, args ...Expr) Expr { return &CallExpr{Function: f, Args: args} }
func and(l, r Expr) Expr             { return &AndExpr{Left: l, Right: r} }
func par(l, r Expr) Expr             { return &ParallelExpr{Left: l, Right: r} }
func internalCall(fName string, args ...Expr) Expr {
	return &InternalCallExpr{FuncName: fName, Args: args}
}

// Mock state creators
func mockState(val any, latMs float64) *VarState {
	var valOutcome any
	switch v := val.(type) {
	case bool:
		valOutcome = (&core.Outcomes[bool]{}).Add(1.0, v)
	case int64:
		valOutcome = (&core.Outcomes[int64]{}).Add(1.0, v)
	case float64:
		valOutcome = (&core.Outcomes[float64]{}).Add(1.0, v)
	case string:
		valOutcome = (&core.Outcomes[string]{}).Add(1.0, v)
	case core.AccessResult:
		valOutcome = (&core.Outcomes[core.AccessResult]{}).Add(1.0, v)
	default:
		panic(fmt.Sprintf("mockState: unsupported value type %T", val))
	}

	latOutcome := (&core.Outcomes[core.Duration]{}).Add(1.0, core.Millis(latMs))

	return &VarState{ValueOutcome: valOutcome, LatencyOutcome: latOutcome}
}

// *** Local test helper for mean duration ***
func meanDurationLatency(o *core.Outcomes[core.Duration]) core.Duration {
	if o == nil || o.Len() == 0 {
		return 0
	}
	totalWeight := 0.0
	weightedSum := 0.0
	for _, b := range o.Buckets {
		totalWeight += b.Weight
		weightedSum += b.Value * b.Weight
	}
	if totalWeight <= 1e-12 {
		return 0
	}
	return weightedSum / totalWeight
}
