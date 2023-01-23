# go-proxy

Simple implementation of a HTTP forwarding proxy.

## Usage
Create a CA for the proxy. This creates a private key `proxy-ca.key` and a corresponding certificate `proxy-ca.crt`.
These will be used to sign the certificates for the TLS interception.
```
go-proxy -create-ca
```

Run the proxy:
```
go-proxy
```

The proxy will listen under `0.0.0.0:8080`.

## Clients
To make HTTP requests over the proxy you have to configure your HTTP clients that they:
* Use the proxy
* Trust the CA of the proxy (`proxy-ca.crt`)

### curl
```
curl --cacert proxy-ca.crt --proxy http://localhost:8080 https://example.com

```

### Chromium
```
chromium --proxy-server=http://localhost:8080
```

Add the CA certificate of the proxy to the trust store.
Go to `Settings -> Privacy and security -> Security -> Manage certificates` or just type `chrome://settings/certificates` in the address bar.
Then click `Import` and select the `proxy-ca.crt` file.
