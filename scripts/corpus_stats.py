import os
import json
from collections import Counter
import sys

def load_json_files_from_subdirectory(subdirectory):
    json_data = []
    for root, _, files in os.walk(subdirectory):
        for file in files:
            if file.endswith('.json'):
                with open(os.path.join(root, file), 'r') as f:
                    data = json.load(f)
                    json_data.append(data)
    return json_data


def analyze_transactions(transactions, method_counter):

    for tx in transactions:
        call_data = tx.get('call', {})
        data_abi_values = call_data.get('dataAbiValues', {})
        method_signature = data_abi_values.get('methodSignature', '')

        method_counter[method_signature] += 1



def main(subdirectory):
    transaction_seqs = load_json_files_from_subdirectory(subdirectory)

    method_counter = Counter()
    total_length = 0

    for seq in transaction_seqs:
        analyze_transactions(seq, method_counter)
        total_length += len(seq)

    average_length = total_length // len(transaction_seqs)

    print(f"Number of Sequences in {subdirectory}: {len(transaction_seqs)}")
    print("\n")

    print(f"Average Length of Transactions List: {average_length}")
    print("\n")
    print("Frequency of Methods Called:")
    for method, count in method_counter.most_common():
        print(f"-  {method}: {count}")
    print("\n")
    print(f"Number of Unique Methods: {len(method_counter)}")
    print("\n")

if __name__ == '__main__':
    if len(sys.argv) != 2:
        print("Usage: python3 corpus_stats.py <corpus>")
        print("Computes statistics on the transactions in the given corpus.")
        sys.exit(1)
    main(sys.argv[1])
