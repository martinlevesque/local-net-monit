# local-net-monit

`local-net-monit` allows you to monitor the ports on your local network, for each given accessible host from the machine running the service.
It also allows to monitor the ports accessible from the internet, by calling a second service which needs to setup externally of your LAN.

See below a screenshot of the web interface (`cmd/localNetMonit`), which shows the status of the ports monitored by the service.

![image](https://github.com/user-attachments/assets/a868b8cb-c88c-48b1-a907-90cb765e66e6)




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
- `WEB_ROOT_ALLOWED_ORIGIN_IP_PATTERN`. Example ^10\.0\.0\..*$ When set, only allow origin IPs matching the regex, for requests on the web root.
