variables = {
  hostname = "localhost"
}

variable_source "users" "file/csv" {
  file             = "files/users.csv"
  fields           = ["user_id", "name", "pass"]
  skip_header      = true
  header_as_fields = false
}
variable_source "users2" "file/csv" {
  file             = "files/users2.csv"
  fields           = ["user_id2", "name2", "pass2"]
  skip_header      = false
  header_as_fields = true
}
variable_source "filter_src" "file/json" {
  file = "files/filter.json"
}
variable_source "filter_src2" "file/json" {
  file = "files/filter2.json"
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
    variables = {
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
    body = ["token", "auth"]
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
    variables = {
      item = "request.list_req.items[3]"
    }
  }
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
