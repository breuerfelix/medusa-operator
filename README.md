# medusa-operator

This operator installs and configures [Medusa](https://github.com/medusajs/medusa) and [Medusa Admin](https://github.com/medusajs/admin) in a kubernetes cluster based on a `Medusa` CRD.  
An example instance for the CRD is:

```yaml
apiVersion: app.medusa.fbr.ai/v1
kind: Medusa
metadata:
  namespace: medusa
  name: testing
spec:
  backend:
    image: ghcr.io/breuerfelix/medusa-operator/backend:latest
    domain: medusa.fbr.ai
    redisURL: "redis://redis-master.redis:6379?password=secretpassword&channelPrefix=testing-user"
  admin:
    image: ghcr.io/breuerfelix/medusa-operator/admin:latest
    domain: medusa-admin.fbr.ai
```

In order for the operator to function properly you need a Postgres database and a Redis instance.

