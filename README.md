# zteOnu
This is a fork from original [project](https://github.com/Septrum101/zteOnu) with changes from this [fork](https://github.com/stich86/zteOnu) and then some.

You **MUST** manually spoof the MAC address of your NIC to `00:07:29:55:35:57` before running this tool. The tool will strictly enforce this check.

Current supported options:

```
./zteOnu -h

Flags:
  -h, --help          help for zteOnu
  -i, --ip string     ONU IP address (default "192.168.1.1")

  -u, --user string   Factory mode auth username (If not provided, a known list will be used)
  -p, --pass string   Factory mode auth password (If not provided, a known list will be used)
      --port int      ONU http port (default 80)
      --seclvl int    Security level for telnet access, if you got "Permission Denied", try 2 or 1.
                      Use with --telnet flag (default 3)
      --telnet        Enable permanent telnet (port 2323, user: tadmin, pass: dynamically generated)
      --zadmin        Add zadmin user (password: dynamically generated)
      --disable-lan-v6 Disable RA Service and DHCPv6 server for LAN
      --reboot        Reboot the ONU after applying changes
      --tp int        ONU telnet port (default 23)
```

# What's different from original one

- Added all known user/password combinations in a loop; the binary will attempt all of them to enable Telnet.
- Added the --seclvl parameter (default: 3) to change the Telnet access level and avoid the "Access Denied" error.
- Modify login retries up to 5 attempts.
- Changed to use the default HTTP port 80 instead of 8080.
- Fix Orange Bridge mode: Disable RA Service and DHCPv6 server for LAN (use `--disable-lan-v6`).

# Tested ONTs

| Hardware  | Firmware      | Result     | Issues                                        |
|-----------|---------------|------------|-----------------------------------------------|
| F6605RV3  | F660505RO     | Ok         |                                               |
| F6605RV3  | F660506RO     | Ok         |                                               |
| F6605RV3  | F660504RT     | Ok ?       |                                               |
| F6600V903 | F660012RO     | Ok         | Permanent Telnet doesn't have full privileges |
