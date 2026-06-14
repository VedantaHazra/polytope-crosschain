package main

import (
	"math/big"
	"strings"
)

type RawSystemState struct {
	SnapshotID         string             `json:"snapshot_id"`
	BaseBatchGasCents  string             `json:"base_batch_gas_cents"`
	InventoryUSDC      string             `json:"inventory_usdc"`
	InventoryWETH      string             `json:"inventory_weth"`
	InventoryARB       string             `json:"inventory_arb"`
	MaxSlippageCeiling string             `json:"max_slippage_ceiling"`
	Intents            []RawComplexIntent `json:"intents"`
}

type RawComplexIntent struct {
	IntentID             string `json:"intent_id"`
	Token                string `json:"token"`
	DestPayoutCents      string `json:"dest_payout_cents"`
	MarginalGasCents     string `json:"marginal_gas_cents"`
	ExpectedRevenueCents string `json:"expected_revenue_cents"`
	SlippageImpactFactor string `json:"slippage_impact_factor"`
	GroundTruthIsToxic   bool   `json:"ground_truth_is_toxic"`
}

type ParsedSystemState struct {
	SnapshotID         string
	BaseBatchGasCents  *big.Int
	InventoryUSDC      *big.Int
	InventoryWETH      *big.Int
	InventoryARB       *big.Int
	MaxSlippageCeiling *big.Int
	Intents            []ParsedComplexIntent
}

type ParsedComplexIntent struct {
	IntentID             string
	Token                string
	DestPayoutCents      *big.Int
	MarginalGasCents     *big.Int
	ExpectedRevenueCents *big.Int
	SlippageImpactFactor *big.Int
	GroundTruthIsToxic   bool
}

func SafeParse(valStr string) *big.Int {
	n := new(big.Int)
	n.SetString(strings.TrimSpace(valStr), 10)
	return n
}

func ParseComplexState(raw RawSystemState) *ParsedSystemState {
	parsed := &ParsedSystemState{
		SnapshotID:         raw.SnapshotID,
		BaseBatchGasCents:  SafeParse(raw.BaseBatchGasCents),
		InventoryUSDC:      SafeParse(raw.InventoryUSDC),
		InventoryWETH:      SafeParse(raw.InventoryWETH),
		InventoryARB:       SafeParse(raw.InventoryARB),
		MaxSlippageCeiling: SafeParse(raw.MaxSlippageCeiling),
	}
	for _, rawIntent := range raw.Intents {
		parsed.Intents = append(parsed.Intents, ParsedComplexIntent{
			IntentID:             rawIntent.IntentID,
			Token:                rawIntent.Token,
			DestPayoutCents:      SafeParse(rawIntent.DestPayoutCents),
			MarginalGasCents:     SafeParse(rawIntent.MarginalGasCents),
			ExpectedRevenueCents: SafeParse(rawIntent.ExpectedRevenueCents),
			SlippageImpactFactor: SafeParse(rawIntent.SlippageImpactFactor),
			GroundTruthIsToxic:   rawIntent.GroundTruthIsToxic,
		})
	}
	return parsed
}