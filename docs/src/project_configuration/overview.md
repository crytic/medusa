# Configuration Overview

Medusa's project configuration provides extensive and granular control over the execution of the fuzzer. The project
configuration file is a `.json` file that is broken down into three core components.
* [Fuzzing Configuration](./fuzzing_config.md): The fuzzing configuration dictates the parameters with which the fuzzer will execute.
* [Compilation Configuration](./compilation_config.md): The compilation configuration dictates how to compile the fuzzing target
* [Logging Configuration](./logging_config.md): The logging configuration dictates when and where to log events.

To generate a project configuration file, run [`medusa init`](../cli/init.md).

You can also view this [example project configuration file](../static/medusa.json) for visualization.