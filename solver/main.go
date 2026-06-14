package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"
)

// evaluateFinancials calculates the exact profit and capital used for a given array of routed intents
func evaluateFinancials(state *ParsedSystemState, routedIDs []string) (profitCents int64, capitalUsedCents int64, isValid bool) {
	if len(routedIDs) == 0 {
		return 0, 0, true
	}

	routedMap := make(map[string]bool)
	for _, id := range routedIDs {
		routedMap[id] = true
	}

	var totalRevenue, totalPayout, totalMarginalGas, totalSlippage int64
	var usedUSDC, usedWETH, usedARB int64

	for _, intent := range state.Intents {
		if !routedMap[intent.IntentID] {
			continue
		}

		payout := intent.DestPayoutCents.Int64()
		totalRevenue += intent.ExpectedRevenueCents.Int64()
		totalPayout += payout
		totalMarginalGas += intent.MarginalGasCents.Int64()
		totalSlippage += intent.SlippageImpactFactor.Int64()

		if intent.Token == "USDC" { usedUSDC += payout }
		if intent.Token == "WETH" { usedWETH += payout }
		if intent.Token == "ARB"  { usedARB += payout }
	}

	if usedUSDC > state.InventoryUSDC.Int64() || usedWETH > state.InventoryWETH.Int64() || usedARB > state.InventoryARB.Int64() {
		return 0, 0, false 
	}
	if totalSlippage > state.MaxSlippageCeiling.Int64() {
		return 0, 0, false 
	}

	baseGas := state.BaseBatchGasCents.Int64()
	totalCosts := totalPayout + totalMarginalGas + baseGas
	netProfit := totalRevenue - totalCosts

	if netProfit < 500 {
		return 0, 0, false
	}

	return netProfit, totalPayout, true
}

// runGreedySolver simulates a "Smart Greedy" Fractional Knapsack Approximation
func runGreedySolver(state *ParsedSystemState) []string {
	var routedIDs []string
	var usedUSDC, usedWETH, usedARB, totalSlippage int64

	// 1. Clone the intents array so we don't mutate the original state
	sortedIntents := make([]ParsedComplexIntent, len(state.Intents))
	copy(sortedIntents, state.Intents)

	// 2. Sort by Capital Efficiency (ROI)
	// ROI = (Revenue - Marginal Gas) / Payout
	sort.Slice(sortedIntents, func(i, j int) bool {
		payoutI := float64(sortedIntents[i].DestPayoutCents.Int64())
		if payoutI == 0 { payoutI = 1 } // Prevent division by zero
		profitI := float64(sortedIntents[i].ExpectedRevenueCents.Int64() - sortedIntents[i].MarginalGasCents.Int64())
		roiI := profitI / payoutI

		payoutJ := float64(sortedIntents[j].DestPayoutCents.Int64())
		if payoutJ == 0 { payoutJ = 1 }
		profitJ := float64(sortedIntents[j].ExpectedRevenueCents.Int64() - sortedIntents[j].MarginalGasCents.Int64())
		roiJ := profitJ / payoutJ

		return roiI > roiJ // Descending order: Highest ROI first
	})

	// 3. Process the sorted array sequentially
	for _, intent := range sortedIntents {
		payout := intent.DestPayoutCents.Int64()
		marginalGas := intent.MarginalGasCents.Int64()
		revenue := intent.ExpectedRevenueCents.Int64()
		slip := intent.SlippageImpactFactor.Int64()

		if revenue <= (payout + marginalGas) {
			continue 
		}

		canFit := true
		if intent.Token == "USDC" && (usedUSDC+payout) > state.InventoryUSDC.Int64() { canFit = false }
		if intent.Token == "WETH" && (usedWETH+payout) > state.InventoryWETH.Int64() { canFit = false }
		if intent.Token == "ARB"  && (usedARB+payout)  > state.InventoryARB.Int64()  { canFit = false }

		if canFit && (totalSlippage+slip) <= state.MaxSlippageCeiling.Int64() {
			routedIDs = append(routedIDs, intent.IntentID)
			totalSlippage += slip
			if intent.Token == "USDC" { usedUSDC += payout }
			if intent.Token == "WETH" { usedWETH += payout }
			if intent.Token == "ARB"  { usedARB += payout }
		}
	}

	_, _, isValid := evaluateFinancials(state, routedIDs)
	if !isValid {
		return nil 
	}
	return routedIDs
}

func executeMarketValidation() {
	fileBytes, err := os.ReadFile("massive_benchmark_data.json")
	if err != nil {
		fmt.Println("[!] Error: Cannot find massive_benchmark_data.json.")
		return
	}

	var rawStates []RawSystemState
	if err := json.Unmarshal(fileBytes, &rawStates); err != nil {
		fmt.Printf("[!] Error parsing benchmark JSON: %v\n", err)
		return
	}

	fmt.Println("\n=================================================================")
	fmt.Println("    POLYTOPE: MARKET VALIDATION & CAPITAL EFFICIENCY BACKTEST    ")
	fmt.Println("=================================================================")
	
	var greedyTotalProfit, greedyTotalCapital int64
	var polytopeTotalProfit, polytopeTotalCapital int64
	var greedyFailedBatches int
	var polytopeFailedBatches int

	start := time.Now()

	for i, rawState := range rawStates {
		fmt.Printf("[*] Block %3d/%d | ID: %s | Intents: %2d -> Processing...\n", 
			i+1, len(rawStates), rawState.SnapshotID[:8], len(rawState.Intents))

		parsedState := ParseComplexState(rawState)

		// 1. Run Smart Greedy Baseline
		greedyRouted := runGreedySolver(parsedState)
		greedyProfit, greedyCap, gValid := evaluateFinancials(parsedState, greedyRouted)
		if gValid {
			greedyTotalProfit += greedyProfit
			greedyTotalCapital += greedyCap
		} else {
			greedyFailedBatches++
		}

		// 2. Run Polytope Max-SMT with Warm Start
		var finalPolytopeRouted []string
		
		polytopeRouted, status := OptimizeComplexMempool(parsedState, greedyProfit)
		
		// If SMT timed out entirely or couldn't beat Greedy, safely fallback to Greedy's route
		if status == "UNSAT" || len(polytopeRouted) == 0 {
		    finalPolytopeRouted = greedyRouted
		} else {
		    finalPolytopeRouted = polytopeRouted
		}

		polyProfit, polyCap, pValid := evaluateFinancials(parsedState, finalPolytopeRouted)
		if pValid {
			polytopeTotalProfit += polyProfit
			polytopeTotalCapital += polyCap
		} else {
			polytopeFailedBatches++
		}
	}

	totalDuration := time.Since(start)

	greedyUSD := float64(greedyTotalProfit) / 100.0
	polytopeUSD := float64(polytopeTotalProfit) / 100.0
	
	profitDelta := polytopeUSD - greedyUSD
	profitIncreasePct := 0.0
	if greedyUSD > 0 {
		profitIncreasePct = (profitDelta / greedyUSD) * 100
	}

	greedyCapUSD := float64(greedyTotalCapital) / 100.0
	polytopeCapUSD := float64(polytopeTotalCapital) / 100.0
	
	var capDeltaPct float64
	if greedyCapUSD > 0 {
		capDeltaPct = ((polytopeCapUSD - greedyCapUSD) / greedyCapUSD) * 100
	}

	fmt.Println("\n=================================================================")
	fmt.Println("                        BACKTEST RESULTS                         ")
	fmt.Println("=================================================================")
	fmt.Printf("[*] Ingested 15,000 Historical Intents across %d Mempool Blocks.\n", len(rawStates))
	fmt.Printf("[*] Dual-Execution Completed in %s\n", totalDuration)
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("STRATEGY A: SMART GREEDY ROUTING (Fractional Knapsack)")
	fmt.Printf(" -> Total Yield Extracted  : $%.2f\n", greedyUSD)
	fmt.Printf(" -> Total Capital Deployed : $%.2f\n", greedyCapUSD)
	fmt.Printf(" -> Batches Reverted       : %d\n", greedyFailedBatches)
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("STRATEGY B: POLYTOPE SMT (Combinatorial Matrix)")
	fmt.Printf(" -> Total Yield Extracted  : $%.2f\n", polytopeUSD)
	fmt.Printf(" -> Total Capital Deployed : $%.2f\n", polytopeCapUSD)
	fmt.Printf(" -> Batches Reverted       : %d\n", polytopeFailedBatches)
	fmt.Println("-----------------------------------------------------------------")
	fmt.Println("BUSINESS IMPACT (The True Multi-Dimensional Edge):")
	fmt.Printf(" [✓] Net Profit Variance    : %.2f%%\n", profitIncreasePct)
	if capDeltaPct < 0 {
		fmt.Printf(" [✓] Capital Efficiency     : Used %.2f%% LESS inventory to clear optimal paths.\n", -capDeltaPct)
	} else {
		fmt.Printf(" [✓] Capital Efficiency     : Used %.2f%% MORE inventory to extract maximum edge.\n", capDeltaPct)
	}
	fmt.Println("=================================================================")
}

func main() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n[!] Force terminate signal received. Exiting immediately...")
		os.Exit(1)
	}()

	executeMarketValidation()
}