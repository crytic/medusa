import sys
import json
import binascii

with open('alloc.json', 'r') as f:
    data = json.load(f)

# Iterate over the key-value pairs

def validate_json(data):
    for key, value in data.items():
        if isinstance(value, dict):
            validate_json(value)
        if isinstance(value, str):
            if value.startswith("0x"):
                value = value[2:]
            print(value)
            # try:
                # Try to decode the value as hex
            bytes_value = int(value, 16).to_bytes((len(value) + 1) // 2, byteorder='big')
            # except (binascii.Error, ValueError):
                # Print the key and value if hex decoding fails
            print(f"Key: {key}, Value: {value} - cannot be hex decoded")
            # sys.exit(1)

validate_json(data)