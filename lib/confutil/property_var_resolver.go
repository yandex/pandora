package confutil

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// Resolve custom tag token with property variable value. Allow read from properties file
// for example: secret: '${property: /etc/datasources/secret.properties#tvm_secret}'
var PropertyTagResolver TagResolver = propertyTokenResolver

func propertyTokenResolver(in string) (string, error) {
	split := strings.SplitN(in, "#", 2)
	filename, property := split[0], split[1]
	file, err := os.Open(filename)
	if err != nil {
		return "", fmt.Errorf("cannot open file: '%v'", filename)
	}

	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			kv := strings.SplitN(line, "=", 2)
			if kv[0] == property {
				return kv[1], nil
			}
		}
	}

	return "", fmt.Errorf("no such property '%v', in file '%v'", property, filename)
}
