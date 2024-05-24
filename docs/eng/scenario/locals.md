[Home](../../index.md)

---

# Locals

Foreword:

Creating a script from HCL consists of two stages:
- Parsing stage - converts the HCL file into the internal structure of the generator.
- Execution stage - involves the steps of:
    - Preprocessing
    - Templating
    - Postprocessing

This article focuses on the parsing stage.

## HCL

Just like in Terraform, you can use the `locals` block, which allows you to create additional variables. 
It is important to note that these variables are only utilized during the parsing of HCL and cannot be used 
during the execution stage.

### Functions

You can use [HCL functions](functions.md#hcl-functions).

### Example of Use

You can use `locals` variables to define common headers.

Note the use of the `merge()` function.

```hcl
locals {
  common_headers = {
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
  next = "next"
}
locals {
  // Merge the new variable with the local variable local.common_headers
  auth_headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  next = "next"
}

request "list_req" {
  // Merge the new variable with the local variable local.common_headers
  method = "GET"
  headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  tag = "list"
  uri = "/list"

  postprocessor "var/jsonpath" {
    mapping = {
      item_id = "$.items[0]"
      items   = "$.items"
    }
  }
}
```

## YAML

In YAML format, you can use anchors.

For common variables, you can use the `locals` helper block.

```yaml
locals:
  global-headers: &global-headers
    Content-Type: application/json
    Useragent: Yandex

requests:
  - name: auth_req
    headers:
      <<: *global-headers
  - name: list_req
    headers: &auth-headers
      <<: *global-headers
      Authorization: Bearer {{.request.auth_req.postprocessor.token}}
  - name: order_req
    headers:
      <<: *auth-headers
```

---

- [Scenario generator / HTTP](../scenario-http-generator.md)
- [Scenario generator / gRPC](../scenario-grpc-generator.md)

---

[Home](../../index.md)
