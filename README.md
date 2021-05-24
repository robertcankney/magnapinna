# Magnapinna
Magnapinna is client and server software that enables clients in private IP ranges to establish remote shell sessions over gRPC. It uses a publicly accessible server to broker communication between multiple clients, providing a simple interface that bypasses the need for a VPN.

## Current status
The shell management code is mostly done (unfortunately Linux systems programming is not the Go topic with the most documentation available), and client and server code is in progress.
