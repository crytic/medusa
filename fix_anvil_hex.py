import json
import re

# Helper function to pad odd-length hex strings
def pad_hex(value):
    if isinstance(value, str) and re.match(r'^0x[0-9a-fA-F]+$', value):
        if len(value) % 2 != 0:
            return '0x0' + value[2:]
    if isinstance(value, str) and value == "0x":
        return '0x00'
    return value

# Recursively process the input dictionary
def process_dict(d):
    new_dict = {}
    for key, value in d.items():
        new_key = pad_hex(key)  # Fix the key if necessary
        if isinstance(value, dict):
            new_dict[new_key] = process_dict(value)
        elif isinstance(value, str):
            new_dict[new_key] = pad_hex(value)
        else:
            new_dict[new_key] = value
    return new_dict



# Load the input JSON file
with open('alloc.json') as f:
    data = json.load(f)

# Process the input dictionary
fixed = process_dict(data)

# Write the processed dictionary to the output JSON file
with open('output.json', 'w') as f:
    json.dump(fixed, f, indent=4)