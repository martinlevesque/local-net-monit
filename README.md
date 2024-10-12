# local-net-monit

`local-net-monit` allows you to monitor the ports on your local network, for each given accessible host from the machine running the service.
It also allows to monitor the ports accessible from the internet, by calling a second service which needs to setup externally of your LAN.

See below a screenshot of the web interface (`cmd/localNetMonit`), which shows the status of the ports monitored by the service.

![image](https://github.com/user-attachments/assets/a868b8cb-c88c-48b1-a907-90cb765e66e6)

As you can see, on the top of the screen, public ports are shown and below are the ports accessible from the local network for all machines currently up.

Furthermore, the dashboard interface allows you to verify each given port by clicking on a port, you'll see the following interface:

![image](https://github.com/user-attachments/assets/d652efed-701f-4096-9c2f-d0eca359ab04)

An unverified port (shown in red) can be used to alert you if a new port is unverified and could be a potential security issue. You can review it and mark as verified if it's a known port expected to be up.
A JSON endpoint is exposed `/status` from the service `cmd/localNetMonit` which can be monitored with status checks tool such as uptimerobot or any other similar service.


## Services

There are 2 services in this repository:

- `cmd/localNetMonit` - Monitor local ports and public ports, and provide a web interface to visualize ports status.
- `cmd/remotePortCheckServer` - A service to check the status of the public ports. When public ports monitoring is enabled (see environment variables section), this server is called by the `localNetMonit` service to check the status of the public ports. A sample instance is deployed here https://remote-port-checker-server.fly.dev/ (see the `fly.toml` file in the root of the repository). This service is need for monitoring public ports as it needs to be deployed externally of your LAN.


## Installation

A docker-compose.yml file is provided to run the services. You can run the services with the following command to boot up localNetMonit:

```bash
docker compose up local-net-monit --build -d
```

The default port is 10001. Environments variables can be specified in .env, see the section below for the list of environment variables.

The `remotePortCheckServer` service can be deployed on a cloud provider such as fly.io, see the `fly.toml` file in the root of the repository for an example of how to deploy the service on fly.io, or you can run it locally with the following command:

```bash
docker compose up remote-port-check-server --build -d
```

It runs on port 8081 by default.

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
