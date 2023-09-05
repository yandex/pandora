
variable_source "users" "file/csv" {
  file              = "files/users.csv"
  fields            = ["user_id", "name", "pass"]
  ignore_first_line = true
  delimiter         = ";"
}
variable_source "filter_src" "file/json" {
  file = "files/filter.json"
}
variable_source "variables" "variables" {
}

request "auth_req" {
  method  = "POST"
  headers = {
    Content-Type = "application/json"
    Useragent    = "Tank"
  }
  tag = "auth"

  preprocessor {
    mapping = {
      user_id = "source.users[0].user_id"
    }
  }

  body = <<EOF
{"user_id":  {{.preprocessor.user_id}}}
EOF
  uri  = "/auth"


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
      "Content-Type" = "application/json"
    }
    body        = ["token"]
    status_code = 200

    size {
      val = 10000
      op  = ">"
    }
  }

  templater = "text"
}
request "list_req" {
  method  = "GET"
  headers = {
    Authorization = "Bearer {{.request.auth_req.token}}"
    Content-Type  = "application/json"
    Useragent     = "Tank"
  }
  tag = "list"
  uri = "/list"

  postprocessor "var/jsonpath" {
    mapping = {
      item_id = "$.items[0]"
      items   = "$.items"
    }
  }

  templater = "text"
}
request "item_req" {
  method  = "POST"
  headers = {
    Authorization = "Bearer {{.request.auth_req.token}}"
    Content-Type  = "application/json"
    Useragent     = "Tank"
  }
  tag  = "item_req"
  body = <<EOF
{"item_id": {{.preprocessor.item}}}
EOF
  uri  = "/item"

  preprocessor {
    mapping = {
      item = "request.list_req.items[3]"
    }
  }

  templater = "text"
}

scenario "scenario1" {
  weight           = 50
  min_waiting_time = 500
  shoot            = [
    "auth_req(1)",
    "sleep(100)",
    "list_req(1)",
    "sleep(100)",
    "item_req(3)"
  ]
}
scenario "scenario2" {
  weight           = 50
  min_waiting_time = 500
  shoot            = [
    "auth_req(1)",
    "sleep(100)",
    "list_req(1)",
    "sleep(100)",
    "item_req(2)"
  ]
}
