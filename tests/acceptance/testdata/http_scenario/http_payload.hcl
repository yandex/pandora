
variable_source "users" "file/csv" {
  file              = "testdata/http_scenario/users.csv"
  fields            = ["user_id", "name", "pass"]
  ignore_first_line = true
  delimiter         = ","
}
variable_source "filter_src" "file/json" {
  file = "testdata/http_scenario/filter.json"
}
variable_source "global" "variables" {
  variables = {
    id = "randInt(10,20)"
  }
}
request "auth_req" {
  method = "POST"
  uri    = "/auth"
  headers = {
    Content-Type = "application/json"
    Useragent    = "Yandex"
    Global-Id = "{{ .source.global.id }}"
  }
  tag       = "auth"
  body      = <<EOF
{"user_id":  {{.request.auth_req.preprocessor.user_id}}, "name":"{{.request.auth_req.preprocessor.rand_name}}", "uuid":"{{uuid}}"}
EOF
  templater {
    type = "html"
  }

  preprocessor {
    mapping = {
      user_id = "source.users[next].user_id"
      rand_name = "randString(5, abc)"
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
  headers = {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
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
  headers = {
    Authorization = "Bearer {{.request.auth_req.postprocessor.token}}"
    Content-Type  = "application/json"
    Useragent     = "Yandex"
  }
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
