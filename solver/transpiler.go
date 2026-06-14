package main

import (
	"github.com/mitchellh/go-z3"
)

// OptimizeComplexMempool now accepts a "Warm Start" baseline profit from the Greedy heuristic
func OptimizeComplexMempool(state *ParsedSystemState, baselineProfitCents int64) ([]string, string) {
	cfg := z3.NewConfig()
	// Raise the guillotine to 200ms. We have a guaranteed floor, so we can afford to let Z3 think a bit longer.
	cfg.SetParamValue("timeout", "200") 
	defer cfg.Close()
	
	ctx := z3.NewContext(cfg)
	defer ctx.Close()

	intSort := ctx.IntSort()
	zero := ctx.Int(0, intSort)

	invUSDC := ctx.Int(int(state.InventoryUSDC.Int64()), intSort)
	invWETH := ctx.Int(int(state.InventoryWETH.Int64()), intSort)
	invARB := ctx.Int(int(state.InventoryARB.Int64()), intSort)
	maxSlippage := ctx.Int(int(state.MaxSlippageCeiling.Int64()), intSort)
	baseGas := ctx.Int(int(state.BaseBatchGasCents.Int64()), intSort)

	usedUSDC := ctx.Int(0, intSort)
	usedWETH := ctx.Int(0, intSort)
	usedARB := ctx.Int(0, intSort)
	totalSlippage := ctx.Int(0, intSort)
	totalRevenue := ctx.Int(0, intSort)
	totalMarginalGas := ctx.Int(0, intSort)
	totalPayout := ctx.Int(0, intSort)

	var maxPossibleRevenue int64
	decisions := make([]*z3.AST, len(state.Intents))
	var baseConstraints []*z3.AST

	for i, intent := range state.Intents {
		sym := ctx.Symbol(intent.IntentID)
		decisions[i] = ctx.Const(sym, intSort)
		
		baseConstraints = append(baseConstraints, decisions[i].Ge(zero))
		baseConstraints = append(baseConstraints, decisions[i].Le(ctx.Int(1, intSort)))

		payout := ctx.Int(int(intent.DestPayoutCents.Int64()), intSort)
		revenue := ctx.Int(int(intent.ExpectedRevenueCents.Int64()), intSort)
		gas := ctx.Int(int(intent.MarginalGasCents.Int64()), intSort)
		slip := ctx.Int(int(intent.SlippageImpactFactor.Int64()), intSort)

		maxPossibleRevenue += intent.ExpectedRevenueCents.Int64()
		activePayout := payout.Mul(decisions[i])
		
		if intent.Token == "USDC" { usedUSDC = usedUSDC.Add(activePayout) }
		if intent.Token == "WETH" { usedWETH = usedWETH.Add(activePayout) }
		if intent.Token == "ARB"  { usedARB = usedARB.Add(activePayout) }

		totalSlippage = totalSlippage.Add(slip.Mul(decisions[i]))
		totalRevenue = totalRevenue.Add(revenue.Mul(decisions[i]))
		totalMarginalGas = totalMarginalGas.Add(gas.Mul(decisions[i]))
		totalPayout = totalPayout.Add(activePayout)
	}

	baseConstraints = append(baseConstraints, usedUSDC.Le(invUSDC))
	baseConstraints = append(baseConstraints, usedWETH.Le(invWETH))
	baseConstraints = append(baseConstraints, usedARB.Le(invARB))
	baseConstraints = append(baseConstraints, totalSlippage.Le(maxSlippage))

	totalCosts := totalPayout.Add(totalMarginalGas).Add(baseGas)
	netProfit := totalRevenue.Sub(totalCosts)

	// --- WARM START BINARY SEARCH ---
	// Start the search strictly AT or ABOVE the Greedy baseline!
	low := baselineProfitCents 
	if low < 500 {
	    low = 500
	}
	high := maxPossibleRevenue 
	
	var bestRouted []string
	foundValid := false
	iterations := 0
	maxIterations := 20 
	epsilon := int64(100) 

	for low <= high && iterations < maxIterations {
		iterations++
		mid := low + (high-low)/2
		
		if (high - low) <= epsilon {
			break
		}

		s := ctx.NewSolver()
		for _, constraint := range baseConstraints {
			s.Assert(constraint)
		}
		
		s.Assert(netProfit.Ge(ctx.Int(int(mid), intSort)))

		if s.Check() == z3.True {
			foundValid = true
			model := s.Model()
			
			currentRouted := make([]string, 0)
			for i, intent := range state.Intents {
				val := model.Eval(decisions[i])
				if val != nil && val.String() == "1" {
					currentRouted = append(currentRouted, intent.IntentID)
				}
			}
			bestRouted = currentRouted
			model.Close()
			s.Close()

			low = mid + 1 
		} else {
			s.Close()
			high = mid - 1 
		}
	}

	if !foundValid {
		return nil, "UNSAT"
	}

	return bestRouted, "SAT"
}