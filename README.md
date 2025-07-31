# General description
The purpose of this tool for https://github.com/Netcracker/qubership-apihub is to automate operation groups creation.

# Steps performed
* Read all version operations (only a list, without data).
* Filter operations locally by custom criteria. The logic is simply hardcoded in the script.
* Send a request to create a group.
* Send a request to set the content of the group, use selected operations.

# How to run
Compile by `go build .` in the sources folder.

## Run arguments
Example:
`-apihubURL http://127.0.0.1:8081 -packageId WS.TEST -version 123 -group test -token dslfjsdnfckjdenacknewkdnskakjzxkfx`