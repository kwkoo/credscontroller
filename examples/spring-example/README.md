# Spring Example - A vault aware application

In this example we are deploying a spring application that connects to Vault using the token retrieved by the init container and retrieves a secret.

The below picture shows the workflow.

You need to have Vault and Vault-Controller installed as explained [here](../../README.md)

Enable the kv secrets engine

```
export VAULT_TOKEN=$ROOT_TOKEN

vault secrets enable \
  -tls-skip-verify \
  -path=secret kv
```

Create a policy that allows the spring-example role to read only from the spring-example generic backend

```
vault policy write \
  -tls-skip-verify \
  spring-example \
  ./examples/spring-example/spring-example.hcl
```

Create a secret for the application to consume

```
vault kv put \
  -tls-skip-verify \
  secret/spring-example \
  password=pwd
```

Before you create a new project for the example app, we need to allow the example app to pull the `vault-controller` image (you can safely ignore the warning message):

```
oc policy add-role-to-user \
  -n vault-controller \
  system:image-puller \
  system:serviceaccount:spring-example:default
```

Build the application

```
oc new-project spring-example

oc new-build redhat-openjdk18-openshift:1.4~https://github.com/raffaelespazzoli/credscontroller \
  --context-dir=examples/spring-example \
  --name spring-example
  
oc logs -f bc/spring-example
```

Join the network with vault-controller (skip this step if your cluster is not configured with the multi-tenant network):

```
oc adm pod-network join-projects --to vault-controller spring-example
```

Deploy the spring example app

```
oc create -f ./examples/spring-example/spring-example.yaml

oc expose svc spring-example
```

Now you should be able to call a service that returns the secret

```
export SPRING_EXAMPLE_ADDR=http://`oc get route | grep -m1 spring | awk '{print $2}'`

curl $SPRING_EXAMPLE_ADDR/secret
```
