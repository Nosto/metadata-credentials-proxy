# Local AWS metadata credentials proxy

This metadata proxy can be used to pass AWS credentials to docker containers
needing access tokens. It uses user credentials from environmental variables
or ~/.aws/config and uses them to get STS assume role credentials. The user
credentials are read when the proxy starts and the STS credentials are cached.
The cached credentials are updated automatically when they expire.

The metadata proxy responds to the following queries:

* http://169.254.169.254/latest/meta-data/iam/security-credentials
* http://169.254.169.254/latest/meta-data/iam/security-credentials/dev

The IAM role is determined by looking at the env variable IAM_ROLE of the
requesting docker container. It needs to include the full arn of the role, not
just the role name. If a role has been given as a command line parameter, it
is used as default role when container does not have IAM_ROLE set.

This takes away the need for copying AWS credentials inside docker images when
building outside AWS.

## Security

Like the metadata service on Amazon EC2 instance, the service does not provide
any kind of authentication. Any process that is able to access the service can
get credentials. On macOS limiting access to only other docker containers
running in same network prevents applications on the host from accessing it, but
other containers may still forward requests to it. The service should not be
run on any publicly available host.

## Usage on macOS

The metadata proxy needs to run inside docker to be able to inspect the
env variables of other containers. For this it also needs access to docker.sock.

The metadata proxy needs to run with IP 169.254.169.254. A separate network
named metadata should be created for this:

$ docker network create -d bridge --subnet 169.254.169.0/24 metadata

To build and start the container on macOS, run the run.sh script:

$ ./run.sh [default iam role]

For other containers to be able to access the metadata proxy, they need to be
in the metadata network. Something like this is needed in docker-compose.yml:

```
networks:
  default:
    external:
      name: metadata
```

To set the IAM_ROLE in docker-compose.yml, add a IAM_ROLE variable like this:

```
version: "2"

services:
  example:
    build: .
    environment:
      - IAM_ROLE=arn:aws:iam::1234567890:role/example_role
```

## Network layout on macOS

Example with a build container that requires AWS IAM credentials. The build
container needs to be connected to the same metadata bridge where metadata
container runs.

```
┌───────────┐ ┌──────────┐
│ metadata  │ │  Build   │   ┌───────────┐  ┌────────────┐
│ container │ │container │   │Container X│  │Container Y │
└───────────┘ └──────────┘   └───────────┘  └────────────┘
      ▲             ▲              ▲               ▲
      └───────┬─────┘              └──────┬────────┘
              ▼                           ▼
     ┌────────────────┐          ┌────────────────┐
     │ metadata bridge│          │ docker0 bridge │
     └────────────────┘          └────────────────┘
              │                           │
              └──────────┬────────────────┘
                         ▼
                    ┌────────┐
                    │ Docker │
                    └────────┘
                         │
                         ▼
                 ┌───────────────┐
                 │   OS X host   │
                 └───────────────┘                        
```

## Testing

To test that it works, first start a container in metadata network:

```
docker run --network metadata -it ubuntu bash
apt-get update && apt-get install curl
```

And then query the proxy:

```
curl http://169.254.169.254/latest/meta-data/iam/security-credentials/
```

```
curl http://169.254.169.254/latest/meta-data/iam/security-credentials/dev
```
