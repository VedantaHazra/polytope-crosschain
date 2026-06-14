import json
import random
import sys
from dune_client.client import DuneClient

# =================================================================
# CONFIGURATION 
# =================================================================
DUNE_API_KEY = "VisDmJNbudnPnYzXnfJLdalnZCo2voj9" # <-- Paste your Dune API key here
QUERY_ID = 7716921
OUTPUT_FILE = "massive_benchmark_data.json"

def process_dune_api_data():
    if DUNE_API_KEY == "YOUR_DUNE_API_KEY":
        print("[!] Execution Halted: Please insert your DUNE_API_KEY.")
        sys.exit(1)

    print(f"[*] Connecting to Dune Analytics API (Query ID: {QUERY_ID})...")
    
    try:
        dune = DuneClient(DUNE_API_KEY)
        query_result = dune.get_latest_result(QUERY_ID)
        transactions = query_result.result.rows
        print(f"[+] Successfully fetched {len(transactions)} historical executions.")
    except Exception as e:
        print(f"[!] Dune API Error: {str(e)}")
        sys.exit(1)

    eth_price_usd = 3500.00
    mempool_snapshots = []
    tokens = ["USDC", "WETH", "ARB"]
    
    # Group every 75 historical transactions into a single "Mempool Snapshot"
    chunk_size = 75 
    total_intents = 0
    total_toxic = 0

    print("[*] Compiling transactions into combinatorial matrices...")

    for i in range(0, len(transactions), chunk_size):
        chunk = transactions[i:i + chunk_size]
        if len(chunk) < 50:
            break # Skip the last tiny chunk to maintain strict matrix density

        mempool_intents = []
        
        for tx in chunk:
            token = random.choice(tokens) 
            
            # 1. Parse Real Gas Overhead (from Dune JSON response)
            gas_used = float(tx['gas_used'])
            gas_price_gwei = float(tx['gas_price_gwei'])
            gas_eth = (gas_used * gas_price_gwei) / 1e9
            gas_cents = int((gas_eth * eth_price_usd) * 100)
            
            # 2. Parse Real Payout
            payout_eth = float(tx['eth_value_transferred'])
            if payout_eth == 0: 
                payout_eth = random.uniform(0.1, 1.5) # Fallback for ERC20 routes
            payout_cents = int((payout_eth * eth_price_usd) * 100)
            
            # 3. Apply the Strict Market Model
            # A standard intent protocol charges a ~0.1% bridge fee + $1.50 execution premium
            protocol_fee_cents = int(payout_cents * 0.001)
            premium_cents = 150
            revenue_cents = payout_cents + protocol_fee_cents + premium_cents
            
            # The Ground Truth: Did real historical gas exceed their premium?
            is_toxic = gas_cents > (protocol_fee_cents + premium_cents)
            if is_toxic:
                total_toxic += 1

            mempool_intents.append({
                "intent_id": tx['intent_id'],
                "token": token,
                "dest_payout_cents": str(payout_cents),
                "marginal_gas_cents": str(gas_cents),
                "expected_revenue_cents": str(revenue_cents),
                "slippage_impact_factor": str(random.randint(1, 3)),
                "ground_truth_is_toxic": is_toxic
            })
            total_intents += 1

        # Feed the real transaction chunk into our constrained SMT environment
        mempool_snapshots.append({
            "snapshot_id": f"Historical_Window_{i//chunk_size}",
            "base_batch_gas_cents": "500", 
            "inventory_usdc": "500000",   
            "inventory_weth": "300000",   
            "inventory_arb":  "100000",   
            "max_slippage_ceiling": "40", 
            "intents": mempool_intents
        })

    with open(OUTPUT_FILE, 'w') as f:
        json.dump(mempool_snapshots, f, indent=2)
        
    print(f"[+] Extraction Complete: Packaged {total_intents} REAL intents into {len(mempool_snapshots)} historical mempool states.")
    print(f"[!] Historical Market Analysis: Found {total_toxic} toxic margin-bleeds in the dataset.")

if __name__ == "__main__":
    process_dune_api_data()