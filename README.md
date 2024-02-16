# Process Tracker

### Support
- Darwin (MacOS Intel)
- Linux
- Windows

### Usage
```bash
# Help
./proctracker --help

# List available process (only main process)
./proctracker --list

# List all process found
./proctracker --list -all

# Find specify process
./proctracker --find=<proc_name>

# Start tracker (require config file, default is config.yml)
./proctracker

# Start tracker with specify custom config file location
./proctracker --config-file=<custom config location file>

# Generate config file with specific location
./proctracker --gen-config --config-file=custom.yml
```

### Example config file
- config.yml
```yaml
process:
  name: nginx
  interval: 10s # default is 60s
alert:
  line:
    to: U12345...
    token: eyxyz...
  discord: 
    webhook: https://discord.com/api/webhooks/123/xyz
```
