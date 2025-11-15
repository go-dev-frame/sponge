#!/usr/bin/env bash
# Usage: ./yamlParser.sh <yaml_file> <key_path>
# Example: ./yamlParser.sh config.yaml '.http.tls.keyFile'

yaml_file="$1"
key_path="$2"

if [[ -z "$yaml_file" || -z "$key_path" ]]; then
  echo "Usage: $0 <yaml_file> <key_path>"
  exit 1
fi

if [[ ! -f "$yaml_file" ]]; then
  echo "File not found: $yaml_file"
  exit 1
fi

key_path="${key_path#.}"
keys_for_awk=$(echo "$key_path" | tr '.' ' ')

awk -v keys="$keys_for_awk" '
BEGIN {
    # Split the input key path string into an array
    split(keys, path, " ")
    path_len = length(path)
    current_level = 1
    # indent_levels array stores the indentation length for each level
    # Set indent_levels[0] to -1 to prevent errors in the first level check
    indent_levels[0] = -1
}

# Skip empty lines or lines that are purely comments
/^[[:space:]]*#|^[[:space:]]*$/ { next }

{
    # Match indentation at the start of the line and the key
    # m[1] is the indentation, m[2] is the key
    if (match($0, /^([[:space:]]*)([^:]+):/, m)) {
        indent = length(m[1])
        key = m[2]

        # Adjust current level by comparing current indentation with the previous level
        # If current indent is less than or equal to previous level, step back in hierarchy
        while (indent <= indent_levels[current_level - 1]) {
            current_level--
        }

        # If the key on the current line matches the key in the path we are looking for
        if (key == path[current_level]) {
            # Record the indentation of the current level
            indent_levels[current_level] = indent

            # If we have matched the last level of the path, extract and process its value
            if (current_level == path_len) {
                # Extract everything after the colon as the initial value
                value = $0
                sub(/^[^:]+:[[:space:]]*/, "", value)

                # --- Clean the value ---
                # 1. Remove comments at the end of the line
                sub(/[[:space:]]*#.*$/, "", value)
                # 2. Remove all leading/trailing whitespace
                gsub(/^[[:space:]]+|[[:space:]]+$/, "", value)
                # 3. Remove double quotes surrounding the value
                gsub(/^"|"$/, "", value)

                print value
                exit
            }

            # If not the last level, increment level and continue matching downwards
            current_level++
        }
    }
}
' "$yaml_file"
