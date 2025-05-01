package core

import (
	"math"
	"sort"
)

// --- Discussion: Duration Reduction Strategies ---
/*
// Reducing the number of buckets in Outcomes[Duration] distributions is necessary
// to manage complexity during composition. However, different strategies have
// trade-offs regarding accuracy and preservation of statistical properties.
// We evaluated two main approaches:
//
// 1. InterpolateDurations:
//    - Method: Builds the empirical CDF of the input distribution and linearly
//      interpolates latency values at target quantiles (e.g., the midpoints
//      P10, P30, P50, P70, P90 for 5 buckets) of the output distribution.
//    - Pros: Simple concept, produces a smooth output distribution, relatively fast.
//    - Cons:
//        - Poor Mean Preservation (for skewed/tailed distributions): Because it samples
//          specific quantiles, it can significantly underestimate the impact of low-probability,
//          high-latency tails on the mean. In tests, the mean shifted considerably.
//        - Poor Tail Preservation: It effectively ignores the shape of the original
//          distribution beyond the highest sampled quantile (e.g., P90 for 5 buckets).
//          This leads to significant underestimation of high percentiles like P99/P999
//          if the original tail was heavier.
//    - Use Case: Suitable when a smooth, fixed-size approximation is needed and
//      exact preservation of mean or tail statistics is less critical.
//
// 2. ReduceDurationsAdaptive (using AdaptiveReduce + DurationSignificanceWeightedLatencyDelta):
//    - Method: Assigns an "importance" score to each input bucket based on its
//      probability weight and the magnitude of latency change compared to its
//      neighbors (with an optional boost for endpoint buckets). It keeps the
//      'targetBuckets' most important buckets and discards the rest. Requires
//      pre-sorted input for the significance function. Output is re-sorted and
//      renormalized.
//    - Pros:
//        - Better Mean Preservation: By keeping high-weight buckets and points of
//          significant change, it tends to preserve the overall mean much better
//          than quantile interpolation, especially for non-uniform distributions.
//        - Better Tail Preservation (with endpoint boost): Boosting the significance
//          of the min/max latency buckets significantly increases the chance they
//          are retained, preserving P99/P100 much more effectively than interpolation.
//        - Conceptually Simple Selection: Removes "less interesting" points.
//    - Cons:
//        - Output Distribution Shape: Can be less smooth than interpolation as it's
//          just a selection of original points. Gaps might appear.
//        - Percentile Shifts: While P99 might be preserved, intermediate percentiles
//          (like P90 in the tests) can shift significantly if those original buckets
//          are deemed less important than tail buckets needed to preserve P99. The
//          output percentiles reflect the kept points, not necessarily the original percentiles.
//        - Requires Sorting: Input needs sorting for the delta-based significance function.
//        - Significance Function Tuning: The effectiveness depends on the quality
//          of the significance function (e.g., how much boost to give endpoints).
//    - Use Case: Preferred when preserving the overall mean and critical tail
//      percentiles (like P99) is more important than maintaining a perfectly smooth
//      distribution shape or preserving all intermediate percentiles exactly. This
//      generally provides a better approximation for SLO analysis.
//
// 3. (Future) ReduceDurationsPercentileAnchor:
//    - Method: Adapt the `ReduceAccessResultsPercentileAnchor` logic. Explicitly
//      select buckets closest to key percentiles (P50, P90, P99, P999, etc.) and
//      fill remaining slots with highest-weight buckets.
//    - Pros: Best control over preserving specific, critical percentiles.
//    - Cons: More complex implementation than AdaptiveReduce; may not preserve
//      mean as well if high-weight buckets don't align with anchor percentiles.
//
// 4. (Future) DurationRange` Approach (Alternative Data Structure):
//    - Concept: Instead of tracking latency as `Outcomes[Duration]` (point values), track it as `Outcomes[DurationRange]` where `DurationRange = {Min, Mode, Max}`.
//    - Pros
//        * Allows using reduction algorithms designed for ranges (`MergeOverlapping...`, `InterpolateRangedResults`
//        adapted for `DurationRange`). These might preserve overall min/max bounds and central tendency (via Mode)
//        effectively during reduction.
//        *   Bundles latency uncertainty (Min/Max) directly with the most likely value (Mode).
//        *   Could simplify the VM back to a single track (`Outcomes[V]`, where `V` might be `bool`, `enum`,
//        `DurationRange`, etc.) if *all* latency is represented this way, eliminating the need for the separate latency
//        track and potential synchronization issues.
//    - Cons:
//        *   Requires defining Min/Mode/Max for all primitive latencies.
//        *   All latency calculations become range-based (e.g., adding ranges).
//        *   Metrics rely on `ModeDuration` as the point estimate, which might not always be accurate if the range is wide or skewed.
//        *   If still using a dual track, complexity shifts to the Reducer Registry needing combiners for `V + DurationRange`.
//    Conclusion: A viable alternative, primarily attractive for its potential to improve reduction fidelity using
//    range-aware algorithms. Kept in consideration if `ReduceDurationsAdaptive` proves insufficient for specific
//    accuracy needs, particularly regarding the preservation of the absolute Min/Max latency bounds during reduction.
//    It represents a shift in how latency is fundamentally represented.
//
// **Current Default Choice:** `ReduceDurationsAdaptive` (with endpoint boost) is
// chosen as the default in the VM because it demonstrated superior preservation
// of the mean and critical P99 tail compared to `InterpolateDurations` in tests,
// which is generally more important for performance analysis and SLO checking.
// However, the strategy is configurable via `vm.SetDurationReducer`.
//
*/
type DurationReducerFunc func(input *Outcomes[Duration], targetBuckets int) *Outcomes[Duration]

type DurationRange struct {
	MinDuration  Duration
	ModeDuration Duration // Most likely duration
	MaxDuration  Duration
}

// Implement Metricable for DurationRange (using Mode for point estimates)
func (dr DurationRange) IsSuccess() bool      { return true } // Duration outcomes are implicitly successful paths
func (dr DurationRange) GetLatency() Duration { return dr.ModeDuration }

// cdfPoint represents a point on the Cumulative Distribution Function.
type cdfPoint struct {
	CumulativeWeight float64
	Latency          Duration
}

// InterpolateDurations reduces the number of buckets in an Outcomes[Duration] distribution
// by linearly interpolating along the cumulative distribution function (CDF).
// It aims to preserve the overall shape and statistical properties of the distribution.
func InterpolateDurations(input *Outcomes[Duration], targetBuckets int) *Outcomes[Duration] {
	// --- Handle Edge Cases ---
	if input == nil || input.Len() == 0 {
		return &Outcomes[Duration]{} // Return empty, non-nil outcome
	}
	if targetBuckets <= 0 {
		// log.Printf("WARN: InterpolateDurations called with targetBuckets <= 0")
		return &Outcomes[Duration]{}
	}

	originalTotalWeight := input.TotalWeight()
	if originalTotalWeight <= 1e-12 { // Effectively zero weight
		// log.Printf("WARN: InterpolateDurations called with zero total weight input")
		return &Outcomes[Duration]{}
	}

	if input.Len() <= targetBuckets {
		return input.Copy() // No reduction needed, return a copy
	}

	// --- Prepare Input Data ---
	// Create a mutable copy of buckets and sort by latency
	sortedBuckets := make([]Bucket[Duration], input.Len())
	copy(sortedBuckets, input.Buckets)
	sort.SliceStable(sortedBuckets, func(i, j int) bool {
		return sortedBuckets[i].Value < sortedBuckets[j].Value
	})

	// --- Build Empirical CDF ---
	cdf := make([]cdfPoint, 0, len(sortedBuckets)+1)
	minLatency := sortedBuckets[0].Value
	cdf = append(cdf, cdfPoint{CumulativeWeight: 0.0, Latency: minLatency}) // P0 point

	currentCumulativeWeight := 0.0
	for _, bucket := range sortedBuckets {
		if bucket.Weight < 0 {
			continue
		}
		potentialNewWeight := currentCumulativeWeight + bucket.Weight

		// Add point only if weight increases significantly OR latency changes
		// Threshold for weight increase helps avoid issues with tiny weights
		weightIncreasedSignificantly := (potentialNewWeight > currentCumulativeWeight+1e-12)
		lastCDFLatency := -1.0 // Sentinel value
		if len(cdf) > 0 {
			lastCDFLatency = cdf[len(cdf)-1].Latency
		}

		if weightIncreasedSignificantly || lastCDFLatency != bucket.Value {
			// If latency is same as last point, just update weight
			if len(cdf) > 0 && lastCDFLatency == bucket.Value {
				cdf[len(cdf)-1].CumulativeWeight = potentialNewWeight
			} else {
				// Add point for the *end* of this bucket's weight range
				// Ensure we don't add a point with the same weight as the previous one if latency didn't change
				// (This case should be rare with the outer check, but good safeguard)
				if !(len(cdf) > 0 && cdf[len(cdf)-1].CumulativeWeight >= potentialNewWeight-1e-12 && lastCDFLatency == bucket.Value) {
					cdf = append(cdf, cdfPoint{
						CumulativeWeight: potentialNewWeight,
						Latency:          bucket.Value,
					})
				}
			}
		}
		currentCumulativeWeight = potentialNewWeight
	}

	// Ensure CDF ends exactly at total weight
	if len(cdf) > 0 {
		// If last calculated weight is less than total, adjust or add final point
		if cdf[len(cdf)-1].CumulativeWeight < originalTotalWeight-1e-12 {
			lastLat := cdf[len(cdf)-1].Latency
			// Check if adding a new point with same latency is needed
			if cdf[len(cdf)-1].CumulativeWeight < originalTotalWeight {
				cdf = append(cdf, cdfPoint{CumulativeWeight: originalTotalWeight, Latency: lastLat})
			}
		}
		// Ensure last point has exactly the total weight if it was close
		cdf[len(cdf)-1].CumulativeWeight = originalTotalWeight
	}

	if len(cdf) <= 1 {
		return &Outcomes[Duration]{}
	}
	// --- Interpolate New Buckets ---
	output := &Outcomes[Duration]{}
	newBucketWeight := originalTotalWeight / float64(targetBuckets)
	cdfIndex := 0

	for i := range targetBuckets {
		targetCumulativeWeight := originalTotalWeight * (float64(i) + 0.5) / float64(targetBuckets)

		// Find first CDF point STRICTLY GREATER than target cumulative weight
		upperIdx := -1
		for idx := cdfIndex; idx < len(cdf); idx++ {
			// Use slightly larger tolerance when comparing target to CDF weight
			if cdf[idx].CumulativeWeight > targetCumulativeWeight+1e-9 {
				upperIdx = idx
				break
			}
		}

		var cdfLower, cdfUpper cdfPoint
		if upperIdx == -1 {
			cdfUpper = cdf[len(cdf)-1]
			// If target weight is very close to total weight, use the last two points for interpolation
			if len(cdf) > 1 && targetCumulativeWeight < cdfUpper.CumulativeWeight+1e-9 {
				cdfLower = cdf[len(cdf)-2]
			} else {
				cdfLower = cdfUpper // Target is at or beyond P100, clamp to max latency
			}
			cdfIndex = len(cdf) - 1
		} else if upperIdx == 0 {
			// This means target is <= cdf[0].CumulativeWeight (which should be 0)
			// Assign the minimum latency (cdf[0].Latency)
			cdfLower = cdf[0]
			cdfUpper = cdf[0]
			cdfIndex = 0
		} else {
			cdfUpper = cdf[upperIdx]
			cdfLower = cdf[upperIdx-1]
			cdfIndex = upperIdx - 1
		}

		// Perform linear interpolation
		var interpolatedLatency Duration
		weightRange := cdfUpper.CumulativeWeight - cdfLower.CumulativeWeight
		latencyRange := cdfUpper.Latency - cdfLower.Latency

		if weightRange < 1e-12 || latencyRange < 1e-12 {
			// If points are identical in weight or latency, use the latency of the lower point.
			// If the target weight *exactly* matches cdfUpper weight, we should use cdfUpper latency.
			if math.Abs(targetCumulativeWeight-cdfUpper.CumulativeWeight) < 1e-9 {
				interpolatedLatency = cdfUpper.Latency
			} else {
				interpolatedLatency = cdfLower.Latency
			}
		} else {
			proportion := (targetCumulativeWeight - cdfLower.CumulativeWeight) / weightRange
			if proportion < 0.0 {
				proportion = 0.0
			}
			if proportion > 1.0 {
				proportion = 1.0
			}
			interpolatedLatency = cdfLower.Latency + proportion*latencyRange
		}

		if interpolatedLatency < 0 {
			interpolatedLatency = 0
		}

		output.Add(newBucketWeight, interpolatedLatency)
	}

	// --- Renormalize Weights ---
	finalTotalWeight := output.TotalWeight()
	// Check relative difference for renormalization trigger
	if math.Abs(finalTotalWeight-originalTotalWeight) > 1e-9*math.Abs(originalTotalWeight) && finalTotalWeight > 1e-12 {
		renormFactor := originalTotalWeight / finalTotalWeight
		if !math.IsNaN(renormFactor) && !math.IsInf(renormFactor, 0) {
			for i := range output.Buckets {
				output.Buckets[i].Weight *= renormFactor
			}
		}
	}

	return output
}

// Adaptively reduce an outcome list to fit into a set number of buckets
// by removing the least significant buckets.
// The input distribution 'o' should ideally be sorted if the significance
// function relies on neighbouring values (like delta-based ones).
// AdaptiveReduce sorts internally by *importance*, not the original value order.
func AdaptiveReduce[V any](o *Outcomes[V], maxBuckets int, sigFunc SignificanceFunction[V]) (out *Outcomes[V]) {
	if o == nil || o.Len() <= maxBuckets {
		// Return original or copy if no reduction needed or input is nil
		if o != nil {
			return o.Copy()
		}
		return nil // Or return empty? Let's return nil for nil input.
	}

	type BucketWithImportance struct {
		Bucket[V]  // Embed original bucket
		Importance float64
		Index      int // Original index, useful for stable sorting if importance is equal
	}

	bucketsWithImportance := make([]BucketWithImportance, o.Len())
	for i, bucket := range o.Buckets {
		importance := sigFunc(o, i)
		// Add a small epsilon based on index for stable sort if importance is identical
		// This helps break ties consistently, though not strictly necessary for correctness.
		// importance += float64(i) * 1e-12

		bucketsWithImportance[i] = BucketWithImportance{
			Bucket:     bucket,
			Importance: importance,
			Index:      i,
		}
	}

	// Sort buckets by importance (DESCENDING - keep most important)
	sort.SliceStable(bucketsWithImportance, func(i, j int) bool {
		// Primary sort: Higher importance comes first
		if bucketsWithImportance[i].Importance != bucketsWithImportance[j].Importance {
			return bucketsWithImportance[i].Importance > bucketsWithImportance[j].Importance
		}
		// Secondary sort: Maintain original relative order for stability if importance is same
		return bucketsWithImportance[i].Index < bucketsWithImportance[j].Index
	})

	// Keep the top 'maxBuckets' most important buckets
	out = &Outcomes[V]{} // Initialize empty output
	// Inherit And func? Maybe not, caller should set reducer context.

	// Add the selected buckets to the output
	selectedBuckets := make([]Bucket[V], maxBuckets)
	for i := 0; i < maxBuckets; i++ {
		selectedBuckets[i] = bucketsWithImportance[i].Bucket // Add the original Bucket struct
	}
	out.Buckets = selectedBuckets

	// Note: Output buckets are NOT necessarily sorted by Value anymore.
	// Renormalization and sorting should happen *after* calling AdaptiveReduce.
	return out
}

// DurationSignificanceWeightedLatencyDelta calculates significance for a Duration bucket.
// Boosts endpoints to help preserve tails.
// Input Outcomes 'o' MUST be pre-sorted by Latency.
func DurationSignificanceWeightedLatencyDelta(o *Outcomes[Duration], i int) float64 {
	if o == nil || i < 0 || i >= o.Len() {
		return 0
	}

	bucketWeight := o.Buckets[i].Weight
	currentLatency := o.Buckets[i].Value // Direct access for Duration

	if bucketWeight <= 1e-12 {
		return 0
	}

	var prevDelta, nextDelta float64 = 0, 0
	// Scale epsilon based on latency to handle large/small values better
	// Ensure epsilon is positive even if latency is 0
	epsilon := 1e-9 * math.Max(currentLatency, 1e-9)

	if i > 0 {
		prevDelta = math.Abs(currentLatency - o.Buckets[i-1].Value)
	}
	if i < o.Len()-1 {
		nextDelta = math.Abs(currentLatency - o.Buckets[i+1].Value)
	}

	maxDelta := 0.0
	if o.Len() == 1 {
		maxDelta = currentLatency // Base significance on magnitude if only one point
	} else if i == 0 {
		maxDelta = nextDelta // Use only delta to next for first point
	} else if i == o.Len()-1 {
		maxDelta = prevDelta // Use only delta to prev for last point
	} else {
		maxDelta = math.Max(prevDelta, nextDelta) // Use max delta for intermediate points
	}

	// Base significance = Weight * (MaxLatencyDifference + Epsilon)
	significance := bucketWeight * (maxDelta + epsilon)

	// *** ADD ENDPOINT BOOST ***
	endpointBoostFactor := 5.0 // Boost factor
	if i == 0 || i == o.Len()-1 {
		// Additive boost might be better than multiplicative if base significance is near zero
		// Let's try multiplicative boost for simplicity first.
		// Add a minimum boost component related to weight ensure endpoints aren't ignored if maxDelta is ~0
		minBoost := bucketWeight * epsilon * endpointBoostFactor * 0.1 // Small fraction of epsilon boost
		significance = significance*endpointBoostFactor + minBoost
	}

	return significance
}

// --- Helper Function using AdaptiveReduce for Durations ---
// ReduceDurationsAdaptive helper function (includes renormalization and final sort)
func ReduceDurationsAdaptive(input *Outcomes[Duration], targetBuckets int) *Outcomes[Duration] {
	if input == nil || input.Len() <= targetBuckets {
		if input != nil {
			return input.Copy()
		}
		return &Outcomes[Duration]{}
	}

	// --- Defensively sort input here ---
	// Although caller *should* sort, doing it here makes the function safer.
	// Create a copy first to avoid modifying original input slice.
	inputCopy := input.Copy()
	sort.SliceStable(inputCopy.Buckets, func(i, j int) bool {
		return inputCopy.Buckets[i].Value < inputCopy.Buckets[j].Value
	})
	// --- End defensive sort ---

	reduced := AdaptiveReduce(inputCopy, targetBuckets, DurationSignificanceWeightedLatencyDelta)

	if reduced == nil || reduced.Len() == 0 {
		return reduced
	} // Return if reduction failed or resulted in empty

	// Renormalize weights
	originalTotalWeight := input.TotalWeight() // Use original total weight
	finalTotalWeight := reduced.TotalWeight()
	// Check relative difference for renormalization trigger
	if math.Abs(finalTotalWeight-originalTotalWeight) > 1e-9*math.Abs(originalTotalWeight) && finalTotalWeight > 1e-12 {
		renormFactor := originalTotalWeight / finalTotalWeight
		if !math.IsNaN(renormFactor) && !math.IsInf(renormFactor, 0) {
			for i := range reduced.Buckets {
				reduced.Buckets[i].Weight *= renormFactor
			}
		}
	}

	// Sort the final output by latency for consistency
	sort.SliceStable(reduced.Buckets, func(i, j int) bool {
		return reduced.Buckets[i].Value < reduced.Buckets[j].Value
	})

	return reduced
}
