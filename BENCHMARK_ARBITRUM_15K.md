# Polytope SMT Optimization: Empirical Benchmark Report

## Methodology & Environment

- **Target Environment:** ERC-7683 Cross-Chain Intent Routing (Arbitrum Destination)
- **Order Flow Source:** Across Protocol Relayer (Contract: `0x428a...`) via Dune Analytics API
- **Dataset Size:** 15,000 historical intents chronologically chunked into 200 block-mempool windows.
- **Solver Inventory Constraints:** $5,000 USDC, $3,000 WETH, $1,000 ARB per block.
- **Execution Timeout:** 200ms hard ceiling per block via Z3.

## The Baselines Tested

1. **Strategy A (Smart Greedy):** The institutional standard. A fractional knapsack approximation sorting all intents by capital efficiency: `ROI = (Expected Revenue - Marginal Gas) / Destination Payout`. Executes sequentially until a token constraint or slippage limit is breached.
2. **Strategy B (Polytope):** A Max-SMT Z3 optimization engine utilizing heuristic seeding (warm-started with Strategy A's floor) and binary-search iterative tightening to discover the absolute global maximum yield.

## Raw Execution Output

```text
=================================================================
                        BACKTEST RESULTS
=================================================================
[*] Ingested 15,000 Historical Intents across 200 Mempool Blocks.
[*] Dual-Execution Completed in 1m11.73151675s (Avg: ~358ms/block)
-----------------------------------------------------------------
STRATEGY A: SMART GREEDY ROUTING (Fractional Knapsack)
 -> Total Yield Extracted  : $3658.71
 -> Total Capital Deployed : $1490193.81
 -> Batches Reverted       : 0
-----------------------------------------------------------------
STRATEGY B: POLYTOPE SMT (Combinatorial Matrix)
 -> Total Yield Extracted  : $3861.01
 -> Total Capital Deployed : $1677813.22
 -> Batches Reverted       : 0
-----------------------------------------------------------------
BUSINESS IMPACT (The True Multi-Dimensional Edge):
 [✓] Net Profit Variance    : +5.53%
 [✓] Capital Efficiency     : Used 12.59% MORE inventory to extract maximum edge.
=================================================================
```
