[Home](../../index.md)

---

# Randomization Functions

You can use functions to generate random values:
- randInt
- randString
- uuid

These functions can be utilized in different parts of the scenarios with specific usage characteristics:
- [In Templates](#in-templates)
- [In the Data Source - variables](#in-the-data-source---variables)
- [In Preprocessors](#in-preprocessors)

## Usage

### uuid

Generates a random uuid v4.

### randInt

Generates a pseudorandom number.

Arguments are optional. Calling the function without arguments will generate a random number in the range of 0-9.

Providing one argument will generate a random number in the range from 0 to that number.

Providing two arguments will generate a random number in the range between these two numbers.

### randString

Generates a random string.

Arguments are optional. Calling the function without arguments will generate one random character.

Providing one argument (a number) X will generate a string of length X.

Providing a second argument (a string of characters) Y will use only characters from the specified string Y for generation.

## In Templates

Since the standard Go templating engine is used, it is possible to use built-in functions. More details about these 
functions can be found at [Go template functions](https://pkg.go.dev/text/template#hdr-Functions).

### uuid

```gotemplate
{% raw %}{{ uuid }}{% endraw %}
```

### randInt

no arguments
```gotemplate
{% raw %}{{ randInt }}{% endraw %}
```

1 argument
```gotemplate
{% raw %}{{ randInt 10 }}{% endraw %}
```

2 arguments
```gotemplate
{% raw %}{{ randInt 100 200 }}{% endraw %}
```

2 arguments using source variable
```gotemplate
{% raw %}{{ randInt 200 .source.global.max_rand_int }}{% endraw %} 
```

### randString

no arguments
```gotemplate
{% raw %}{{ randString }}{% endraw %}
```

1 argument
```gotemplate
{% raw %}{{ randString 10 }}{% endraw %}
```

2 arguments
```gotemplate
{% raw %}{{ randString 10 abcde }}{% endraw %}
```

2 arguments using source variable
```gotemplate
{% raw %}{{ randString 20 .source.global.letters }}{% endraw %}
```

## In the Data Source - variables

You can use random value generation functions in the `variables` type data source.

Function calls should be passed as strings (in quotes).

```terraform
variable_source "global" "variables" {
  variables = {
    my_uuid = "uuid()"
    my_random_int1 = "randInt()"                # no arguments
    my_random_int2 = "randInt(10)"              # 1 argument
    my_random_int3 = "randInt(100, 200)"        # 2 arguments
    my_random_string1 = "randString()"          # no arguments
    my_random_string2 = "randString(10)"        # 1 argument
    my_random_string3 = "randString(100, abcde)" # 2 arguments
  }
}
```

## In Preprocessors

You can use random value generation functions in preprocessors.

```terraform
preprocessor {
  mapping = {
    my_uuid = "uuid()"
    my_random_int1 = "randInt()"                # no arguments
    my_random_int2 = "randInt(10)"              # 1 argument
    my_random_int3 = "randInt(100, 200)"        # 2 arguments
    my_random_int4 = "randInt(100, .request.my_req_name.postprocessor.var_from_response)" # 2 arguments, using from response of request my_req_name
    my_random_string1 = "randString()"          # no arguments
    my_random_string2 = "randString(10)"        # 1 argument
    my_random_string3 = "randString(100, abcde)" # 2 arguments
    my_random_string4 = "randString(100, .request.my_req_name.postprocessor.var_from_response)"  # 2 arguments, using from response of request my_req_name
  }
}
```

# HCL functions

You can use follow function 

- [coalesce](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/coalesce)
- [coalescelist](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/coalescelist)
- [compact](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/compact)
- [concat](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/concat)
- [distinct](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/distinct)
- [element](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/element)
- [flatten](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/flatten)
- [index](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/index-fn)
- [keys](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/keys)
- [lookup](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/lookup)
- [merge](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/merge)
- [reverse](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/reverse)
- [slice](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/slice)
- [sort](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/sort)
- [split](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/string/split)
- [values](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/values)
- [zipmap](https://developer.hashicorp.com/packer/docs/templates/hcl_templates/functions/collection/zipmap)


---

- [Scenario generator / HTTP](../scenario-http-generator.md)
- [Scenario generator / gRPC](../scenario-grpc-generator.md)

---

[Home](../../index.md)
