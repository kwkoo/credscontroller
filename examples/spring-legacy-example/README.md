# Spring Legacy Example - A vault unaware application

In this example we are deploying a spring application that is unware of Vault and expect a secret to be available at a given location.

The below picture shows the workflow.

You need to have Vault and Vault-Controller installed as explained [here](../../README.md)

Enable the kv secrets engine if you haven't already done so

```
export VAULT_TOKEN=$ROOT_TOKEN

vault secrets enable \
  -tls-skip-verify \
  -path=secret kv
```


Create a policy that allows the `spring-legacy-example` role to read only from the spring-example generic backend

```
export VAULT_TOKEN=$ROOT_TOKEN

vault policy write \
  -tls-skip-verify \
  spring-legacy-example \
  ./examples/spring-legacy-example/spring-legacy-example.hcl 
```

Create a secret for the application to consume

```
vault kv put \
  -tls-skip-verify \
  secret/spring-legacy-example \
  password=pwd 
```

Before you create a new project for the example app, we need to allow the example app to pull the `vault-controller` image (you can safely ignore the warning message):

```
oc policy add-role-to-user \
  -n vault-controller \
  system:image-puller \
  system:serviceaccount:spring-legacy-example:default
```

Build the application

```
oc new-project spring-legacy-example

oc new-build redhat-openjdk18-openshift:1.4~https://github.com/raffaelespazzoli/credscontroller \
  --context-dir=examples/spring-legacy-example \
  --name spring-legacy-example
  
oc logs -f bc/spring-legacy-example
```

Join the network with vault-controller (skip this step if your cluster is not configured with the multi-tenant network):


```
oc adm pod-network join-projects --to vault-controller spring-legacy-example
```

Deploy the spring legacy example app

```
oc create -f ./examples/spring-legacy-example/spring-legacy-example.yaml

oc expose svc spring-legacy-example
```

now you should be able to call a service that returns the secret

```
export SPRING_LEGACY_EXAMPLE_ADDR=http://`oc get route | grep -m1 spring | awk '{print $2}'`

curl $SPRING_LEGACY_EXAMPLE_ADDR/secret
```