# kubecheck

Kubecheck is an open source golang library published on Github (https://github.com/StenaIT/kubecheck).
It aims to deliver common, configurable healthchecks for Kubernetes and AWS infrastructure. It does this by exposing golang packages that can be used to define a set of healtchecks for the infrastructure it monitors.


## HTTP Server
Kubecheck also comes with an HTTP server that exposes pre-defined routes.
- `/` = An index for the configured checks
- `/checks/` = Performs all healthchecks and reports the result
- `/checks/<name>` = Performs a single healthcheck and reports the result

If a check passes, the status code 200 OK will be returned.  
If a check fails, the status code 429 Failed Dependency will be returned.

## Authentication
Kubecheck does not provide built in authentication. Instead it is recommended that you use something like a reverse proxy with support for basic auth to protect Kubecheck when exposed to the internet.

## Example usage
A basic example is provided in the examples directory of this repository.

## Checks

### What is a check
A "check" is one or more assertions made against a single resource or a group of similar resources.  

A "single resource" may be something like a loadbalancer or a DNS record.  
A "group of resources" may be something like a Traefik (reverse proxy) cluster or Kubernetes nodes.

### Passed/Failed checks

A check can have a `status` of `passed` or `failed`.

The response object of a `failed` check includes additional data that is not returned for a `passed` check.
The `input` and `output` fields are only included for a `failed` check and may contain "schemaless" data. In other words, avoid parsing the data of these fields unless you really need to. In the future, a fixed schema may be applied that make parsing these fields easier.

### What to do when a check fails

The first step is to look at the JSON response for the check that failed. In case of a failed check, details of the assertions made against the resource(s) are returned along with a failed status code.

Here's an example response for a failed check:

```json
{
  "traefik-ping": {
    "description": "Performs a HTTP GET request against the Traefik ping endpoint to verify that the reverse proxy is responding",
    "status": "failed",
    "input": {
      "url": "https://mydomain.io/ping"
    },
    "output": [
      {
        "name": "HTTPStatusCode",
        "result": "passed",
        "assertions": [
          {
            "type": "Equals",
            "result": "failed",
            "expected": 200,
            "actual": 504
          }
        ]
      },
      {
        "name": "HTTPResponseBody",
        "result": "passed",
        "assertions": [
          {
            "type": "Equals",
            "result": "failed",
            "expected": "OK",
            "actual": "Gateway Timeout"
          }
        ]
      },
      {
        "name": "Certificate",
        "result": "passed",
        "entity": {
          "Subject": "CN=*.mydomain.io",
          "Issuer": "CN=Amazon,OU=Server CA 1B,O=Amazon,C=US"
        },
        "assertions": [
          {
            "type": "Expires",
            "result": "passed",
            "expected": "after 7 days",
            "actual": "in 244 days"
          }
        ]
      },
      {
        "name": "ResponseTime",
        "result": "passed",
        "assertions": [
          {
            "type": "LessThen",
            "result": "passed",
            "expected": "500ms",
            "actual": "29.875093ms"
          }
        ]
      }
    ]
  }
}
```

In the example above, it is possible to see the `input` of the check, which is the value(s) used to configure the check.  
The `output` property contains detailed information on each assertion made against the resource(s) and the outcome of that assertion. In some cases the assertion output contains additional information about the entity in question.


## Hooks

Kubecheck has basic support for webhooks, allowing services that support them to get notified.
See the example for more info on how to set it up.
