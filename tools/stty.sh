#!/bin/bash

# Save terminal settings before running the program
stty -g > before.txt

# Run your program
go run . k3d create

# Save terminal settings after running the program
stty -g > after.txt

# Compare the settings
diff before.txt after.txt

# Clean up
rm before.txt after.txt
