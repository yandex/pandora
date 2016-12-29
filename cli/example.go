package cli

// TODO: print example from ../example/
// TODO: update example to YAML
const exampleConfig string = `
{
  "Pools": [
    {
      "Name": "Pool#0",
      "Gun": {
        "GunType": "spdy",
        "Parameters": {
          "Target": "localhost:3000"
        }
      },
      "AmmoProvider": {
        "AmmoType": "jsonline/spdy",
        "AmmoSource": "./example/data/ammo.jsonline"
        "Passes": 10,
      },
      "ResultListener": {
        "ListenerType": "log/simple",
        "Destination": ""
      },
      "UserLimiter": {
        "LimiterType": "periodic",
        "Parameters": {
          "BatchSize": 3,
          "MaxCount": 9,
          "Period": 1
        }
      },
      "StartupLimiter": {
        "LimiterType": "periodic",
        "Parameters": {
          "BatchSize": 2,
          "MaxCount": 5,
          "Period": 0.1
        }
      }
    },
    {
      "Name": "Pool#1",
      "Gun": {
        "GunType": "log"
      },
      "AmmoProvider": {
        "AmmoType": "dummy/log",
        "AmmoSource": ""
      },
      "ResultListener": {
        "ListenerType": "log/simple",
        "Destination": ""
      },
      "UserLimiter": {
        "LimiterType": "periodic",
        "Parameters": {
          "BatchSize": 3,
          "MaxCount": 9,
          "Period": 1
        }
      },
      "StartupLimiter": {
        "LimiterType": "periodic",
        "Parameters": {
          "BatchSize": 1,
          "MaxCount": 4,
          "Period": 0.2
        }
      }
    }
  ]
}
`
