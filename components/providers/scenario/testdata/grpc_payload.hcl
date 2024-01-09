variable_source "users" "file/csv" {
  file              = "testdata/users.csv"
  fields            = ["user_id", "login", "pass"]
  ignore_first_line = true
  delimiter         = ","
}
variable_source "filter_src" "file/json" {
  file = "testdata/filter.json"
}
variable_source "variables" "variables" {
  variables = {
    header = "yandex"
    b      = "s"
  }
}

call "auth_req" {
  call     = "target.TargetService.Auth"
  tag      = "auth"
  metadata = {
    "metadata" = "server.proto"
  }
  preprocessor "prepare" {
    mapping = {
      user = "source.users[next]"
    }
  }
  payload = <<EOF
{"login": "{{.request.auth_req.preprocessor.user.login}}", "pass": "{{.request.auth_req.preprocessor.user.pass}}"}
EOF
  postprocessor "assert/response" {
    payload     = ["token"]
    status_code = 200
  }
}

call "list_req" {
  call     = "target.TargetService.List"
  tag      = "list"
  metadata = {
    "metadata" = "server.proto"
  }
  payload = <<EOF
{"user_id": {{.request.auth_req.postprocessor.userId}}, "token": "{{.request.auth_req.postprocessor.token}}"}
EOF
}

call "order_req" {
  call     = "target.TargetService.Order"
  tag      = "order"
  metadata = {
    "metadata" = "server.proto"
  }
  payload = <<EOF
{"user_id": {{.request.auth_req.postprocessor.userId}}, "item_id": {{.request.order_req.preprocessor.item_id}}, "token": "{{.request.auth_req.postprocessor.token}}"}
EOF
  preprocessor "prepare" {
    mapping = {
      item_id = "request.list_req.postprocessor.result[rand].itemId"
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
