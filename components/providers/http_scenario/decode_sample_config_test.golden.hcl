
variable_source "users" "file/csv" {
  file              = "files/users.csv"
  fields            = ["user_id", "name", "pass"]
  ignore_first_line = true
  delimiter         = ";"
}
variable_source "users2" "file/csv" {
  file              = "files/users2.csv"
  fields            = ["user_id2", "name2", "pass2"]
  ignore_first_line = false
  delimiter         = ";"
}
variable_source "filter_src" "file/json" {
  file = "files/filter.json"
}
variable_source "filter_src2" "file/json" {
  file = "files/filter2.json"
}
variable_source "variables" "variables" {
  variables = {
    var1 = "var"
    var2 = "2"
    var3 = "false"
  }
}

request "auth_req" {
  method = "POST"
  headers = {
    Content-Type = "application/json"
    Useragent    = "Tank"
  }
  tag  = "auth"
  body = "{\"user_id\":  {{.preprocessor.user_id}}}"
  uri  = "/auth"

  preprocessor {
    mapping = {
      user_id = "source.users[0].user_id"
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
    body        = ["token", "auth"]
    status_code = 200

    size {
      val = 10000
      op  = ">"
    }
  }

  templater = "text"
}
request "list_req" {
  method = "GET"
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

  templater = "html"
}
request "item_req" {
  method = "POST"
  headers = {
    Authorization = "Bearer {{.request.auth_req.token}}"
    Content-Type  = "application/json"
    Useragent     = "Tank"
  }
  tag  = "item_req"
  body = "{\"item_id\": {{.preprocessor.item}}}"
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
  shoot            = ["auth_req(1)", "sleep(100)", "list_req(1)", "sleep(100)", "item_req(3)"]
}
scenario "scenario2" {
  weight           = 40
  min_waiting_time = 400
  shoot            = ["auth_req(2)", "sleep(200)", "list_req(2)", "sleep(200)", "item_req(4)"]
}
