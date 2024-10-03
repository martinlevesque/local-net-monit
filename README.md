# local-net-monit

## Features

- Monitor local ports

## Installation

todo

## Environment variables (localNetMonit service)

- `REMOTE_PORT_CHECKER_BASE_URL` - Base url of the remote port checker service, the server located under cmd/remotePortCheckServer. 
- `MONITOR_PUBLIC_PORTS` (true/false) - Default: true. When enabled, the service will monitor the public ports by calling `REMOTE_PORT_CHECKER_BASE_URL`.
- `MONITOR_LOCAL_PORTS` (true/false) - Default: true. When enabled, the service will monitor the local ports.
- `STATUS_PUBLIC_PORTS` (true/false) - Default: true. When enabled, the status endpoint /status will return non 200 if at least one public port is not yet verified.
- `STATUS_LOCAL_PORTS` (true/false) - Default: false. When enabled, the status endpoint /status will return non 200 if at least one local port is not yet verified.
- `PUBLIC_PORTS_FULL_CHECK_INTERVAL_MINUTES` - Default: 120. The interval in minutes to do a full check of the public ports
- `LOCAL_PORTS_FULL_CHECK_INTERVAL_MINUTES` - Default: 60. The interval in minutes to do a full check of the local ports
- `NB_PUBLIC_PORTS_TO_CHECK_PER_BATCH` - Default: 20. The number of public ports to check concurrently
- `SNAPSHOT_STORAGE_PATH` - Default: localNetMonit.json. The path to store the snapshot by the localNetMonit service
