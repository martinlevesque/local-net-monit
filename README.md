# local-net-monit

# can detect wireless vs wired based on the delay

# env variables

- `REMOTE_PORT_CHECKER_BASE_URL` - Base url of the remote port checker service, the server located under cmd/remotePortCheckServer
- `MONITOR_PUBLIC_PORTS` (true/false) - Default: true
- `MONITOR_LOCAL_PORTS` (true/false) - Default: true
- `STATUS_PUBLIC_PORTS` (true/false) - Default: true
- `STATUS_LOCALE_PORTS` (true/false) - Default: false
- `PUBLIC_PORTS_FULL_CHECK_INTERVAL_MINUTES` - Default: 120. The interval in minutes to do a full check of the public ports
- `LOCAL_PORTS_FULL_CHECK_INTERVAL_MINUTES` - Default: 60. The interval in minutes to do a full check of the local ports
- `NB_PUBLIC_PORTS_TO_CHECK_PER_BATCH` - Default: 20. The number of public ports to check concurrently
- `SNAPSHOT_STORAGE_PATH` - Default: localPortsMonit.json
