```json
{
  "fuzzing": {
    "workers": 10,
    "workerResetLimit": 50,
    "timeout": 0,
    "testLimit": 0,
    "callSequenceLength": 100,
    "corpusDirectory": "",
    "coverageEnabled": true,
    "deploymentOrder": [],
    "deployerAddress": "0x1111111111111111111111111111111111111111",
    "senderAddresses": [
      "0x1111111111111111111111111111111111111111",
      "0x2222222222222222222222222222222222222222",
      "0x3333333333333333333333333333333333333333"
    ],
    "blockNumberDelayMax": 60480,
    "blockTimestampDelayMax": 604800,
    "blockGasLimit": 125000000,
    "transactionGasLimit": 12500000,
    "testing": {
      "stopOnFailedTest": true,
      "assertionTesting": {
        "enabled": false,
        "testViewMethods": false
      },
      "propertyTesting": {
        "enabled": true,
        "testPrefixes": ["fuzz_"]
      }
    }
  },
  "compilation": {
    "platform": "crytic-compile",
    "platformConfig": {
      "target": ".",
      "solcVersion": "",
      "exportDirectory": "",
      "args": []
    }
  }
}
```
