# Kerbecs

<img align="right" width="159px" src="https://github.com/BK1031/Kerbecs/blob/main/assets/kerbecs.png?raw=true" alt="kerbecs-logo">

[![Build Status](https://github.com/BK1031/Kerbecs/actions/workflows/test.yml/badge.svg)](https://github.com/BK1031/Kerbecs/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/BK1031/Kerbecs/graph/badge.svg?token=R4NMABYGOZ)](https://codecov.io/gh/BK1031/Kerbecs)
[![GoDoc](https://pkg.go.dev/badge/github.com/bk1031/kerbecs?status.svg)](https://pkg.go.dev/github.com/bk1031/kerbecs?tab=doc)
[![Docker Pulls](https://img.shields.io/docker/pulls/bk1031/kerbecs?style=flat-square)](https://hub.docker.com/repository/docker/bk1031/kerbecs)
[![Release](https://img.shields.io/github/release/bk1031/kerbecs.svg?style=flat-square)](https://github.com/bk1031/kerbecs/releases)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


Kerbecs is a cloud-native API gateway written in [Go](https://go.dev/).
It is designed to be fast, lightweight, and highly extensible.
Kerbecs is also platform-agnostic, and can run in the cloud, a container, or even on bare-metal,
making it perfect for both local development and production environments.

## Getting Started

The easiest way to get started with Kerbecs is to use the official Docker image.
You can pull it from [Docker Hub](https://hub.docker.com/r/bk1031/kerbecs).

```bash
$ docker run -d -p 10310:10310 bk1031/kerbecs:latest
```

Alternatively if you have an existing compose file, you can add Kerbecs as a service.
This way you can easily connect Kerbecs to your existing database.

```yml
kerbecs:
    image: bk1031/kerbecs:latest
    restart: unless-stopped
    environment:
      PORT: "10311"
      STORAGE_MODE: "sql"
      DB_DRIVER: "postgres"
      DB_HOST: "localhost"
      DB_PORT: "5432"
      DB_NAME: "kerbecs"
      DB_USER: "postgres"
      DB_PASSWORD: "password"
    ports:
      - "10311:10311"
```