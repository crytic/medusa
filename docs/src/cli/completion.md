# `completion`

`medusa` provides the ability to generate autocompletion scripts for a given shell.
Once the autocompletion script is ran for a given shell, `medusa`'s commands and flags can be tab-autocompleted.
The following shells are supported:

1. `bash`
2. `zsh`
3. `Powershell`

To understand how to run the autocompletion script for a given shell, run the following command:

```shell
medusa completion --help
```

Once you know how to run the autocompletion script, retrieve the script for that given shell using the following command:

```shell
medusa completion <shell>
```
