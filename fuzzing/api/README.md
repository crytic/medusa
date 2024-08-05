# Medusa Fuzzing API

## Description
This API is served when the fuzzing process starts and is terminated automatically when fuzzing ends.

## Configurable Parameters
Configuration parameters exist for the API and can be provided in the config file under the new `apiConfig` object.

- **enabled**: Whether the API should be enabled.
    - **Default**: False


- **port**: The port where the API should run on. If the provided port is unavailable, the API will be served on the next available port in increments of 1.
    - **Default**: 8080


- **wsUpdateInterval**: The interval (in seconds) with which the API will send updates via websocket connections.
    - **Default**: 5

## Routes

### Main Routes

- **GET /env**: Returns the environment information. This endpoint returns the current environment information as a JSON response. The client can make a GET request to this endpoint to retrieve the environment information.

- **GET /fuzzing**: Returns the fuzzing information. This endpoint returns the current fuzzing information as a JSON response. The client can make a GET request to this endpoint to retrieve the fuzzing information.

- **GET /logs**: Returns the logs. This endpoint returns the logs as a JSON response. The client can make a GET request to this endpoint to retrieve the logs.

- **GET /coverage**: Returns the coverage information. This endpoint returns the current coverage information as a JSON response. The client can make a GET request to this endpoint to retrieve the coverage information.

- **GET /corpus**: Returns the corpus information. This endpoint returns the current corpus information as a JSON response. The client can make a GET request to this endpoint to retrieve the corpus information.


### Websocket Routes

- **GET /ws/env**: Handles WebSocket connections for the environment. The WebSocket connection is used to stream real-time updates of the environment information. The client can connect to this endpoint to receive updates whenever the environment changes.


- **GET /ws/fuzzing**: Handles WebSocket connections for fuzzing. The WebSocket connection is used to stream real-time updates of the fuzzing information. The client can connect to this endpoint to receive updates whenever the fuzzing status changes.


- **GET /ws/logs**: Handles WebSocket connections for logs. The WebSocket connection is used to stream real-time updates of the logs. The client can connect to this endpoint to receive updates whenever new logs are generated.


- **GET /ws/coverage**: Handles WebSocket connections for coverage. The WebSocket connection is used to stream real-time updates of the coverage information. The client can connect to this endpoint to receive updates whenever the coverage changes.


- **GET /ws/corpus**: Handles WebSocket connections for the corpus. The WebSocket connection is used to stream real-time updates of the corpus. The client can connect to this endpoint to receive updates whenever the corpus changes.


- **GET /ws**: Handles WebSocket connections. This endpoint is a catch-all for all other WebSocket routes. The client can connect to this endpoint to receive updates for all the available WebSocket routes.

### Example return data

- Get file
  - Request example:
    ```bash
      curl http://localhost:8080/file?path=/root/audit-metroid/lib/forge-std/src/Base.sol
        ```
    - Response example:
      ```
    `// SPDX-License-Identifier: MIT
    pragma solidity >=0.6.2 <0.9.0;

    import {StdStorage
    } from "./StdStorage.sol";
    import {Vm, VmSafe
    } from "./Vm.sol";
    
    abstract contract CommonBase {
    // Cheat code address, 0x7109709ECfa91a80626fF3989D68f67F5b1DD12D.
    address internal constant VM_ADDRESS = address(uint160(uint256(keccak256("hevm cheat code"))));
    // console.sol and console2.sol work by executing a staticcall to this address.
    address internal constant CONSOLE = 0x000000000000000000636F6e736F6c652e6c6f67;
    // Used when deploying with create2, https://github.com/Arachnid/deterministic-deployment-proxy.
    address internal constant CREATE2_FACTORY = 0x4e59b44847b379578588920cA78FbF26c0B4956C;
    // Default address for tx.origin and msg.sender, 0x1804c8AB1F12E6bbf3894d4083f33e07309d1f38.
    address internal constant DEFAULT_SENDER = address(uint160(uint256(keccak256("foundry default caller"))));
    // Address of the test contract, deployed by the DEFAULT_SENDER.
    address internal constant DEFAULT_TEST_CONTRACT = 0x5615dEB798BB3E4dFa0139dFa1b3D433Cc23b72f;
    // Deterministic deployment address of the Multicall3 contract.
    address internal constant MULTICALL3_ADDRESS = 0xcA11bde05977b3631167028862bE2a173976CA11;
    // The order of the secp256k1 curve.
    uint256 internal constant SECP256K1_ORDER =
    115792089237316195423570985008687907852837564279074904382605163141518161494337;
    
        uint256 internal constant UINT256_MAX =
            115792089237316195423570985008687907853269984665640564039457584007913129639935;
    
        Vm internal constant vm = Vm(VM_ADDRESS);
        StdStorage internal stdstore;
    }
    
    abstract contract TestBase is CommonBase {}
    
    abstract contract ScriptBase is CommonBase {
    VmSafe internal constant vmSafe = VmSafe(VM_ADDRESS);
    }
      ```
- Env info
```json
{
  "config": {
    "fuzzing": {
      "workers": 10,
      "workerResetLimit": 50,
      "timeout": 0,
      "testLimit": 0,
      "shrinkLimit": 5000,
      "callSequenceLength": 100,
      "corpusDirectory": "info",
      "coverageEnabled": true,
      "targetContracts": [
        "MedusaTest"
      ],
      "predeployedContracts": {},
      "targetContractsBalances": [],
      "constructorArgs": {},
      "deployerAddress": "0x30000",
      "senderAddresses": [
        "0x10000",
        "0x20000",
        "0x30000"
      ],
      "blockNumberDelayMax": 60480,
      "blockTimestampDelayMax": 604800,
      "blockGasLimit": 125000000,
      "transactionGasLimit": 12500000,
      "testing": {
        "stopOnFailedTest": false,
        "stopOnFailedContractMatching": false,
        "stopOnNoTests": false,
        "testAllContracts": false,
        "traceAll": false,
        "assertionTesting": {
          "enabled": true,
          "testViewMethods": false,
          "panicCodeConfig": {
            "failOnCompilerInsertedPanic": false,
            "failOnAssertion": true,
            "failOnArithmeticUnderflow": false,
            "failOnDivideByZero": false,
            "failOnEnumTypeConversionOutOfBounds": false,
            "failOnIncorrectStorageAccess": false,
            "failOnPopEmptyArray": false,
            "failOnOutOfBoundsArrayAccess": false,
            "failOnAllocateTooMuchMemory": false,
            "failOnCallUninitializedVariable": false
          }
        },
        "propertyTesting": {
          "enabled": true,
          "testPrefixes": [
            "property_"
          ]
        },
        "optimizationTesting": {
          "enabled": true,
          "testPrefixes": [
            "optimize_"
          ]
        },
        "targetFunctionSignatures": null,
        "excludeFunctionSignatures": null
      },
      "chainConfig": {
        "codeSizeCheckDisabled": true,
        "cheatCodes": {
          "cheatCodesEnabled": true,
          "enableFFI": false
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
    },
    "logging": {
      "level": "info",
      "logDirectory": "log",
      "noColor": false
    },
    "apiConfig": {
      "enabled": true,
      "port": 8080,
      "wsUpdateInterval": 5
    }
  },
  "solcVersion": "0.8.15",
  "system": [
   ...
  ]
}
```

- Fuzzing info
```json
{
    "metrics": {},
    "testCases": [
        {
            "ID": "PROPERTY-InnerInnerDeployment-property-inner-inner-deployment()",
            "LogMessage": {},
            "Message": "[RUNNING] Property Test: InnerInnerDeployment.property_inner_inner_deployment()",
            "Name": "Property Test: InnerInnerDeployment.property_inner_inner_deployment()",
            "Status": "RUNNING"
        },
        {
            "ID": "PROPERTY-InnerDeployment-property-inner-deployment()",
            "LogMessage": {},
            "Message": "[RUNNING] Property Test: InnerDeployment.property_inner_deployment()",
            "Name": "Property Test: InnerDeployment.property_inner_deployment()",
            "Status": "RUNNING"
        },
        {
            "ID": "ASSERTION-InnerInnerDeployment-otherInnerInner()",
            "LogMessage": {},
            "Message": "[RUNNING] Assertion Test: InnerInnerDeployment.otherInnerInner()",
            "Name": "Assertion Test: InnerInnerDeployment.otherInnerInner()",
            "Status": "RUNNING"
        },
        {
            "ID": "ASSERTION-InnerDeployment-deployInnerInner()",
            "LogMessage": {},
            "Message": "[RUNNING] Assertion Test: InnerDeployment.deployInnerInner()",
            "Name": "Assertion Test: InnerDeployment.deployInnerInner()",
            "Status": "RUNNING"
        },
        {
            "ID": "ASSERTION-InnerDeployment-otherInner()",
            "LogMessage": {},
            "Message": "[RUNNING] Assertion Test: InnerDeployment.otherInner()",
            "Name": "Assertion Test: InnerDeployment.otherInner()",
            "Status": "RUNNING"
        },
        {
            "ID": "ASSERTION-InnerDeploymentFactory-deployInner()",
            "LogMessage": {},
            "Message": "[RUNNING] Assertion Test: InnerDeploymentFactory.deployInner()",
            "Name": "Assertion Test: InnerDeploymentFactory.deployInner()",
            "Status": "RUNNING"
        },
        {
            "ID": "ASSERTION-InnerDeploymentFactory-wee()",
            "LogMessage": {},
            "Message": "[RUNNING] Assertion Test: InnerDeploymentFactory.wee()",
            "Name": "Assertion Test: InnerDeploymentFactory.wee()",
            "Status": "RUNNING"
        }
    ]
}
```

- Logs info
```json
{
"logs": "⇾ Setting up base chain\n⇾ Initializing and validating corpus call sequences\n⇾ corpus: health: 77%, sequences: 1372 (1061 valid, 311 invalid)\n⇾ Fuzzing with 10 workers\n⇾ fuzz: elapsed: 0s, calls: 0 (0/sec), seq/s: 0, coverage: 1061\n⇾ fuzz: elapsed: 3s, calls: 23480 (7825/sec), seq/s: 103, coverage: 1061\n⇾ fuzz: elapsed: 6s, calls: 41060 (5858/sec), seq/s: 63, coverage: 1061\n⇾ fuzz: elapsed: 9s, calls: 58579 (5831/sec), seq/s: 61, coverage: 1061\n⇾ fuzz: elapsed: 12s, calls: 69870 (3688/sec), seq/s: 37, coverage: 1061\n⇾ fuzz: elapsed: 15s, calls: 80661 (3583/sec), seq/s: 36, coverage: 1061\n⇾ fuzz: elapsed: 18s, calls: 101069 (6778/sec), seq/s: 69, coverage: 1061\n⇾ fuzz: elapsed: 21s, calls: 115915 (4932/sec), seq/s: 49, coverage: 1061\n⇾ fuzz: elapsed: 24s, calls: 131499 (5186/sec), seq/s: 51, coverage: 1061\n"
}
```

- Coverage info
```json

```

- Corpus info
```json
{
    "unexecutedCallSequences": [
        [
            {
                "call": {
                    "from": "0x0000000000000000000000000000000000010000",
                    "to": "0xa647ff3c36cfab592509e13860ab8c4f28781a66",
                    "nonce": 0,
                    "value": "0x0",
                    "gasLimit": 12500000,
                    "gasPrice": "0x1",
                    "gasFeeCap": "0x0",
                    "gasTipCap": "0x0",
                    "data": "0x946c3724",
                    "dataAbiValues": {
                        "methodSignature": "deployInner()",
                        "inputValues": []
                    },
                    "AccessList": null,
                    "SkipAccountChecks": false
                },
                "blockNumberDelay": 0,
                "blockTimestampDelay": 0
            },
            {
                "call": {
                    "from": "0x0000000000000000000000000000000000030000",
                    "to": "0xa647ff3c36cfab592509e13860ab8c4f28781a66",
                    "nonce": 1,
                    "value": "0x0",
                    "gasLimit": 12500000,
                    "gasPrice": "0x1",
                    "gasFeeCap": "0x0",
                    "gasTipCap": "0x0",
                    "data": "0x946c3724",
                    "dataAbiValues": {
                        "methodSignature": "deployInner()",
                        "inputValues": []
                    },
                    "AccessList": null,
                    "SkipAccountChecks": false
                },
                "blockNumberDelay": 56076,
                "blockTimestampDelay": 360624
            },
            {
                "call": {
                    "from": "0x0000000000000000000000000000000000030000",
                    "to": "0x54919a19522ce7c842e25735a9cfecef1c0a06da",
                    "nonce": 2,
                    "value": "0x0",
                    "gasLimit": 12500000,
                    "gasPrice": "0x1",
                    "gasFeeCap": "0x0",
                    "gasTipCap": "0x0",
                    "data": "0xc23ac55e",
                    "dataAbiValues": {
                        "methodSignature": "otherInner()",
                        "inputValues": []
                    },
                    "AccessList": null,
                    "SkipAccountChecks": false
                },
                "blockNumberDelay": 1,
                "blockTimestampDelay": 134226
            },
            {
                "call": {
                    "from": "0x0000000000000000000000000000000000020000",
                    "to": "0xa647ff3c36cfab592509e13860ab8c4f28781a66",
                    "nonce": 0,
                    "value": "0x0",
                    "gasLimit": 12500000,
                    "gasPrice": "0x1",
                    "gasFeeCap": "0x0",
                    "gasTipCap": "0x0",
                    "data": "0x946c3724",
                    "dataAbiValues": {
                        "methodSignature": "deployInner()",
                        "inputValues": []
                    },
                    "AccessList": null,
                    "SkipAccountChecks": false
                },
                "blockNumberDelay": 39558,
                "blockTimestampDelay": 130342
            }
      ]
    ]
}
```