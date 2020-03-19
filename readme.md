# A notebook for automation tasks

## How to setup ?

Run `./setup.sh` to download all go dependencies

Run `go run server.go` to start the http server that listen on port 1234.

Access `http://localhost:1234` and start typing your command.

## Project structure

- config: storing all yaml files containing main logic of the application, all these files are parsed to generate auto-suggestions for searching on UI.
- formula: when you want to run your custom script that has much more complex logic beyond yaml files.
- public: containing assets for UI.
- src/common: all functions that can does not depend on anything except golang standard lib.
- src/core: all functions and structs that depends on everything except handlers. It's used for core logic of the application.
- src/handlers: all handlers to be used for http server.
- src/repository: for models and utility functions related to interacting with db.
- src/yaml_config: for data structures matching with yaml files and with specific logic for every type of yaml file.
 
## Known issues
- Source code contains a lots of experimental and unused codes, that causes inconvenience in maintaining.
- Some processes that have been killed but still running and sending stdout/stderr through socket.
- Sometimes iframe stop working, you need to reload webpage in order for iframe to work again and continue receiving livestream output from server.
- Sometimes server dies because of concurrent write (no mutex lock issue)

## How to contribute ?

All pull requests are welcome.

For hotfix, PR to master.

For new features and experimental features, PR to develop.