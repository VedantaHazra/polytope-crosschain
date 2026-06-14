# Polytope: Combinatorial Matrix Routing for ERC-7683

Polytope is an offline Max-SMT optimization infrastructure built for cross-chain intent solvers. It utilizes the Microsoft Z3 Theorem Prover to solve the multi-dimensional knapsack problem inherent in managing multi-token solver inventories under overlapping slippage and gas constraints.

## Why SMT over Fractional Knapsacks?

Production routing bots typically sort intents by ROI and execute sequentially. When a solver hits a hard limit on a single asset (e.g., exhausting USDC inventory), the sequence breaks, leaving complex combinations of WETH or ARB trades unrouted.

Polytope evaluates the mempool globally. By warm-starting with a greedy heuristic and deploying binary-search iterative tightening, Polytope intentionally bypasses local maxima, packing denser batches that utilize idle treasury capital to extract a mathematically guaranteed maximal block yield.

## Benchmark

In a backtest against 15,000 historical Arbitrum intents routed by the Across Relayer, Polytope extracted a **+5.53% net absolute profit** over a fractional knapsack baseline, with a median execution latency of ~355ms per block. See `BENCHMARK.md` for full metrics.

## Quick Start (Mac/Linux)

Polytope requires the C++ Z3 engine to compile.

```bash
# 1. Install Z3
brew install z3

# 2. Link CGO Paths (For Apple Silicon M1/M2/M3)
export CGO_CFLAGS="-I/opt/homebrew/include"
export CGO_LDFLAGS="-L/opt/homebrew/lib"

# 3. Run the Backtest
cd solver
go run *.go
```
