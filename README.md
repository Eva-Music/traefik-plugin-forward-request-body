# forward x-www-form-urlencoded request body for keycloak token endpoint

> example:
```
plugin:
      traefik-plugin-forward-request-body:
          url: http://.../protocol/openid-connect/token
```

> add access_token to Authorization header
