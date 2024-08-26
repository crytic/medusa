import os
import json
import sys 

def load_json_files_from_subdirectory(subdirectory):
    json_data = []
    for root, _, files in os.walk(subdirectory):
        for file in files:
            if file.endswith('.json'):
                with open(os.path.join(root, file), 'r') as f:
                    data = json.load(f)
                    json_data.extend(data)
    return json_data

def extract_unique_methods(transactions):
    unique_methods = set()
    for tx in transactions:
        call_data = tx.get('call', {})
        data_abi_values = call_data.get('dataAbiValues', {})
        method_signature = data_abi_values.get('methodSignature', '')
        if method_signature:
            unique_methods.add(method_signature)
    return unique_methods

def compare_methods(subdirectory1, subdirectory2):
    transactions1 = load_json_files_from_subdirectory(subdirectory1)
    transactions2 = load_json_files_from_subdirectory(subdirectory2)

    unique_methods1 = extract_unique_methods(transactions1)
    unique_methods2 = extract_unique_methods(transactions2)

    only_in_subdir1 = unique_methods1 - unique_methods2
    only_in_subdir2 = unique_methods2 - unique_methods1

    return only_in_subdir1, only_in_subdir2

def main(subdirectory1, subdirectory2):

    only_in_subdir1, only_in_subdir2 = compare_methods(subdirectory1, subdirectory2)

    print(f"Methods only in {subdirectory1}:")
    if len(only_in_subdir1) == 0:
        print("  <None>")
    else:
        for method in only_in_subdir1:
            print(f"-  {method}")
    print("\n")
    

    print(f"Methods only in {subdirectory2}:")
    if len(only_in_subdir2) == 0:
        print("  <None>")
    else:
        for method in only_in_subdir2:
            print(f"-  {method}")
    print("\n")

if __name__ == '__main__':
    if len(sys.argv) != 3:
        print("Usage: python3 unique.py <corpus1> <corpus2>")
        print("Compares the unique methods in the two given corpora.")
        sys.exit(1)
    main(sys.argv[1], sys.argv[2])
