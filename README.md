# Go Turbo

The `goturbo` project is a toolkit designed to enhance the efficiency of Golang development. Currently, most tools are related to code generation.

The currently supported commands include:

- [x] [generate](https://github.com/x5iu/genx): Execute the `go generate` command for the entire project with a single command,
  no matter how deeply it is hidden.
- [ ] derive: Implementing various interfaces derived for types
  - [x] [lombok](https://github.com/x5iu/visc): Somewhat similar to Java's Project Lombok, it generates getters/setters/constructors for structs.
  - [ ] ……
- [x] upgrade: A tool used to determine the next semantic version, for example, from v1.0.20 to v1.0.21.
- [x] merge: Merge multiple `.go` files, suitable for streamlining the results of code generation.