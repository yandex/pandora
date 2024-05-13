locals {
  common_headers = {
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
  next = "next"
}
locals {
  auth_headers = merge(local.common_headers, {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
  })
  next = "next"
}
variable_source "users" "file/csv" {
  file              = "testdata/users.csv"
  fields            = ["user_id", "name", "pass"]
  ignore_first_line = true
  delimiter         = ","
}
variable_source "filter_src" "file/json" {
  file = "testdata/filter.json"
}
variable_source "variables" "variables" {
  variables = {
    header = "yandex"
    b = "s"
  }
}
request "auth_req" {
  method = "POST"
  uri    = "/auth"
  headers   = local.common_headers
  tag       = "auth"
  body      = <<EOF
{"user_id":  {{.request.auth_req.preprocessor.user_id}}}
EOF
  templater {
    type = "html"
  }

  preprocessor {
    mapping = {
      user_id = "source.users[${local.next}].user_id"
    }
  }
  postprocessor "var/header" {
    mapping = {
      Content-Type      = "Content-Type|upper"
      httpAuthorization = "Http-Authorization"
    }
  }
  postprocessor "var/jsonpath" {
    mapping = {
      token = "$.auth_key"
    }
  }
  postprocessor "assert/response" {
    headers = {
      Content-Type = "json"
    }
    body = ["key"]
    size {
      val = 40
      op  = ">"
    }
  }
  postprocessor "assert/response" {
    body = ["auth"]
  }
}
request "list_req" {
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
request "order_req" {
  method = "POST"
  uri    = "/order"
  headers = local.auth_headers
  tag  = "order_req"
  body = <<EOF
{"item_id": {{.request.order_req.preprocessor.item}}}
EOF

  preprocessor {
    mapping = {
      item = "request.list_req.postprocessor.items[next]"
    }
  }
}

request "order_req2" {
  method = "POST"
  uri    = "/order"
  headers = {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
  tag  = "order_req"
  body = <<EOF
{"item_id": {{.request.order_req2.preprocessor.item}}  }
EOF

  preprocessor {
    mapping = {
      item = "request.list_req.postprocessor.items[next]"
    }
  }
}

scenario "scenario_name" {
  weight           = 50
  min_waiting_time = 10
  requests         = [
    "auth_req(1)",
    "sleep(100)",
    "list_req(1)",
    "sleep(100)",
    "order_req(3)"
  ]
}

scenario "scenario_2" {
  requests         = [
    "auth_req(1)",
    "sleep(100)",
    "list_req(1)",
    "sleep(100)",
    "order_req(2)"
  ]
}
