# GatherYourDeals-data

This is the database solution repository for GatherYourDeals project. It provides the standard format of how purchase data should be recorded and a client-server database solution.

## Overview

This repository provides:
- Standard data format for purchase records
- Client-server database architecture
- Peer-to-peer connection capabilities
- Flexible authentication system

## Documentation

- **[Connection and Authentication](docs/connection_and_auth.md)** - Details on local/cloud hosting, peer-to-peer connections, and authentication design
- **[Data Format](docs/data_format.md)** - Standard format for purchase data, ETL process, and database schema

## Key Features

- **Location-agnostic**: Host locally or in the cloud, connect across different hosts
- **Decentralized auth**: Client-side credential generation and management
- **Flexible schema**: Customizable fields with metadata validation
- **Access control**: Owner-based write access with read-only key sharing

