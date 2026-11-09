# Extra trusted roots

Every `*.pem` file in this directory is compiled into goat-client and trusted,
in addition to the system roots, when it dials a management or signal server.

The public tree ships no certificate here, so a public build trusts the system
roots only. A deployment whose servers present certificates from a private CA
copies that CA's root certificate into this directory as a `.pem` file before
building.
